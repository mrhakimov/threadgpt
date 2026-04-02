package data

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"threadgpt/domain"
	"time"
)

// setupTestServer creates a fake Supabase server that returns messages in desc order
// (newest first), simulating what real Supabase does with order=created_at.desc
func setupTestServer(t *testing.T, allMessages []domain.Message) *httptest.Server {
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

		msgs := make([]domain.Message, len(allMessages))
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

		if offset >= len(msgs) {
			msgs = []domain.Message{}
		} else {
			msgs = msgs[offset:]
			if limit < len(msgs) {
				msgs = msgs[:limit]
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(msgs)
	}))
	return srv
}

func makeMessages(n int) []domain.Message {
	msgs := make([]domain.Message, n)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		msgs[i] = domain.Message{
			ID:        string(rune('a' + i)),
			SessionID: "sess1",
			Role:      "user",
			Content:   string(rune('A' + i)),
			CreatedAt: base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
		}
	}
	return msgs
}

func TestGetMainDesc_InitialPage(t *testing.T) {
	allMsgs := makeMessages(25)
	srv := setupTestServer(t, allMsgs)
	defer srv.Close()

	os.Setenv("SUPABASE_URL", srv.URL)
	os.Setenv("SUPABASE_SERVICE_KEY", "test")

	store := NewSupabaseStore()
	msgs, err := store.GetMainDesc(context.Background(), "sess1", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 10 {
		t.Fatalf("expected 10 messages, got %d", len(msgs))
	}
	for i := 1; i < len(msgs); i++ {
		if msgs[i].CreatedAt < msgs[i-1].CreatedAt {
			t.Errorf("messages not in asc order at index %d: %s < %s", i, msgs[i].CreatedAt, msgs[i-1].CreatedAt)
		}
	}
	if msgs[0].ID != string(rune('a'+15)) {
		t.Errorf("expected first msg to be index 15 (%c), got %s", 'a'+15, msgs[0].ID)
	}
	if msgs[9].ID != string(rune('a'+24)) {
		t.Errorf("expected last msg to be index 24 (%c), got %s", 'a'+24, msgs[9].ID)
	}
}

func TestGetMainDesc_LoadMorePage(t *testing.T) {
	allMsgs := makeMessages(25)
	srv := setupTestServer(t, allMsgs)
	defer srv.Close()

	os.Setenv("SUPABASE_URL", srv.URL)
	os.Setenv("SUPABASE_SERVICE_KEY", "test")

	store := NewSupabaseStore()
	msgs, err := store.GetMainDesc(context.Background(), "sess1", 10, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 10 {
		t.Fatalf("expected 10 messages, got %d", len(msgs))
	}
	for i := 1; i < len(msgs); i++ {
		if msgs[i].CreatedAt < msgs[i-1].CreatedAt {
			t.Errorf("messages not in asc order at index %d", i)
		}
	}
	if msgs[0].ID != string(rune('a'+5)) {
		t.Errorf("expected first msg index 5 (%c), got %s", 'a'+5, msgs[0].ID)
	}
	if msgs[9].ID != string(rune('a'+14)) {
		t.Errorf("expected last msg index 14 (%c), got %s", 'a'+14, msgs[9].ID)
	}
}

func TestGetMainDesc_FullCombinedOrder(t *testing.T) {
	allMsgs := makeMessages(25)
	srv := setupTestServer(t, allMsgs)
	defer srv.Close()

	os.Setenv("SUPABASE_URL", srv.URL)
	os.Setenv("SUPABASE_SERVICE_KEY", "test")

	store := NewSupabaseStore()
	page1, _ := store.GetMainDesc(context.Background(), "sess1", 10, 0)
	page2, _ := store.GetMainDesc(context.Background(), "sess1", 10, 10)
	combined := append(page2, page1...)

	for i := 1; i < len(combined); i++ {
		if combined[i].CreatedAt < combined[i-1].CreatedAt {
			t.Errorf("combined not in asc order at index %d: %s before %s", i, combined[i-1].ID, combined[i].ID)
		}
	}
	if combined[0].ID != string(rune('a'+5)) {
		t.Errorf("combined should start at index 5, got %s", combined[0].ID)
	}
	if combined[19].ID != string(rune('a'+24)) {
		t.Errorf("combined should end at index 24, got %s", combined[19].ID)
	}
}
