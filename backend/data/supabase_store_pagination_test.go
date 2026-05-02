package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"threadgpt/domain"
	"time"
)

func setupConversationServer(t *testing.T, allRefs []domain.ConversationRef) *httptest.Server {
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

		refs := make([]domain.ConversationRef, len(allRefs))
		copy(refs, allRefs)
		if order == "created_at.desc" {
			sort.Slice(refs, func(i, j int) bool {
				return refs[i].CreatedAt > refs[j].CreatedAt
			})
		} else {
			sort.Slice(refs, func(i, j int) bool {
				return refs[i].CreatedAt < refs[j].CreatedAt
			})
		}

		if offset >= len(refs) {
			refs = []domain.ConversationRef{}
		} else {
			refs = refs[offset:]
			if limit < len(refs) {
				refs = refs[:limit]
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(refs)
	}))
	return srv
}

func makeConversationRefs(n int) []domain.ConversationRef {
	refs := make([]domain.ConversationRef, n)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		refs[i] = domain.ConversationRef{
			ConversationID: fmt.Sprintf("conv-%012d", i+1),
			SessionID:      "sess1",
			CreatedAt:      base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
		}
	}
	return refs
}

func TestListBySessionDesc_InitialPageIsAscendingWithinNewestWindow(t *testing.T) {
	allRefs := makeConversationRefs(25)
	srv := setupConversationServer(t, allRefs)
	defer srv.Close()

	os.Setenv("SUPABASE_URL", srv.URL)
	os.Setenv("SUPABASE_SERVICE_KEY", "test")

	store := NewSupabaseStore()
	refs, err := store.ListBySessionDesc(context.Background(), "sess1", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 10 {
		t.Fatalf("expected 10 refs, got %d", len(refs))
	}
	for i := 1; i < len(refs); i++ {
		if refs[i].CreatedAt < refs[i-1].CreatedAt {
			t.Fatalf("refs not in ascending order at index %d", i)
		}
	}
	if refs[0].ConversationID != "conv-000000000016" {
		t.Fatalf("unexpected first ref: %+v", refs[0])
	}
	if refs[9].ConversationID != "conv-000000000025" {
		t.Fatalf("unexpected last ref: %+v", refs[9])
	}
}

func TestListBySessionDesc_LoadMorePageReturnsOlderWindowAscending(t *testing.T) {
	allRefs := makeConversationRefs(25)
	srv := setupConversationServer(t, allRefs)
	defer srv.Close()

	os.Setenv("SUPABASE_URL", srv.URL)
	os.Setenv("SUPABASE_SERVICE_KEY", "test")

	store := NewSupabaseStore()
	refs, err := store.ListBySessionDesc(context.Background(), "sess1", 10, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 10 {
		t.Fatalf("expected 10 refs, got %d", len(refs))
	}
	if refs[0].ConversationID != "conv-000000000006" || refs[9].ConversationID != "conv-000000000015" {
		t.Fatalf("unexpected paginated refs: %+v", refs)
	}
}
