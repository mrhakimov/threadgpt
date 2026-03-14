package db

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"
)

// setupTestServer creates a fake Supabase server that returns messages in desc order
// (newest first), simulating what real Supabase does with order=created_at.desc
func setupTestServer(t *testing.T, allMessages []Message) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		order := q.Get("order")
		limitStr := q.Get("limit")
		offsetStr := q.Get("offset")

		limit := 10
		offset := 0
		if limitStr != "" {
			limit = 0
			for _, c := range limitStr {
				limit = limit*10 + int(c-'0')
			}
		}
		if offsetStr != "" {
			offset = 0
			for _, c := range offsetStr {
				offset = offset*10 + int(c-'0')
			}
		}

		// Sort messages according to requested order
		msgs := make([]Message, len(allMessages))
		copy(msgs, allMessages)
		if order == "created_at.desc" {
			sort.Slice(msgs, func(i, j int) bool {
				return msgs[i].CreatedAt > msgs[j].CreatedAt
			})
		} else {
			sort.Slice(msgs, func(i, j int) bool {
				return msgs[i].CreatedAt < msgs[j].CreatedAt
			})
		}

		// Apply offset and limit
		if offset >= len(msgs) {
			msgs = []Message{}
		} else {
			msgs = msgs[offset:]
			if limit < len(msgs) {
				msgs = msgs[:limit]
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msgs)
	}))
	return srv
}

// makeMessages creates N messages with sequential created_at timestamps
func makeMessages(n int) []Message {
	msgs := make([]Message, n)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		msgs[i] = Message{
			ID:        string(rune('a' + i)),
			SessionID: "sess1",
			Role:      "user",
			Content:   string(rune('A' + i)),
			CreatedAt: base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
		}
	}
	return msgs
}

func TestGetMessagesDesc_InitialPage(t *testing.T) {
	// 25 messages total. Initial load: offset=0, limit=10 → should return msgs 15..24 (newest 10) in asc order
	allMsgs := makeMessages(25)
	srv := setupTestServer(t, allMsgs)
	defer srv.Close()

	os.Setenv("SUPABASE_URL", srv.URL)
	os.Setenv("SUPABASE_SERVICE_KEY", "test")

	msgs, err := GetMessagesDesc("sess1", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 10 {
		t.Fatalf("expected 10 messages, got %d", len(msgs))
	}
	// Should be in ascending order (oldest of the page first)
	for i := 1; i < len(msgs); i++ {
		if msgs[i].CreatedAt < msgs[i-1].CreatedAt {
			t.Errorf("messages not in asc order at index %d: %s < %s", i, msgs[i].CreatedAt, msgs[i-1].CreatedAt)
		}
	}
	// Should be the newest 10: indices 15..24
	if msgs[0].ID != string(rune('a'+15)) {
		t.Errorf("expected first msg to be index 15 (%c), got %s", 'a'+15, msgs[0].ID)
	}
	if msgs[9].ID != string(rune('a'+24)) {
		t.Errorf("expected last msg to be index 24 (%c), got %s", 'a'+24, msgs[9].ID)
	}
}

func TestGetMessagesDesc_LoadMorePage(t *testing.T) {
	// After loading 10 newest, load-more sends offset=10 → should return msgs 5..14 (next older 10) in asc order
	allMsgs := makeMessages(25)
	srv := setupTestServer(t, allMsgs)
	defer srv.Close()

	os.Setenv("SUPABASE_URL", srv.URL)
	os.Setenv("SUPABASE_SERVICE_KEY", "test")

	msgs, err := GetMessagesDesc("sess1", 10, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 10 {
		t.Fatalf("expected 10 messages, got %d", len(msgs))
	}
	// Should be in ascending order
	for i := 1; i < len(msgs); i++ {
		if msgs[i].CreatedAt < msgs[i-1].CreatedAt {
			t.Errorf("messages not in asc order at index %d", i)
		}
	}
	// Should be indices 5..14
	if msgs[0].ID != string(rune('a'+5)) {
		t.Errorf("expected first msg index 5 (%c), got %s", 'a'+5, msgs[0].ID)
	}
	if msgs[9].ID != string(rune('a'+14)) {
		t.Errorf("expected last msg index 14 (%c), got %s", 'a'+14, msgs[9].ID)
	}
}

func TestGetMessagesDesc_FullCombinedOrder(t *testing.T) {
	// Simulate: load page 1 (offset=0), then load page 2 (offset=10), prepend.
	// Combined result should be msgs 5..24 in ascending order.
	allMsgs := makeMessages(25)
	srv := setupTestServer(t, allMsgs)
	defer srv.Close()

	os.Setenv("SUPABASE_URL", srv.URL)
	os.Setenv("SUPABASE_SERVICE_KEY", "test")

	page1, _ := GetMessagesDesc("sess1", 10, 0)  // newest 10: indices 15..24 asc
	page2, _ := GetMessagesDesc("sess1", 10, 10) // next older 10: indices 5..14 asc

	// Frontend prepends: [...page2, ...page1]
	combined := append(page2, page1...)

	// Verify overall ascending order
	for i := 1; i < len(combined); i++ {
		if combined[i].CreatedAt < combined[i-1].CreatedAt {
			t.Errorf("combined not in asc order at index %d: %s before %s",
				i, combined[i-1].ID, combined[i].ID)
		}
	}
	// Should start at index 5, end at index 24
	if combined[0].ID != string(rune('a'+5)) {
		t.Errorf("combined should start at index 5, got %s", combined[0].ID)
	}
	if combined[19].ID != string(rune('a'+24)) {
		t.Errorf("combined should end at index 24, got %s", combined[19].ID)
	}
}
