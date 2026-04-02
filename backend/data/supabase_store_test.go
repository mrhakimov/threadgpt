package data

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupabaseStore_GetByID_DecodesSnakeCaseFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/v1/sessions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":            "00000000-0000-0000-0000-000000000001",
				"api_key_hash":  "hash-1",
				"assistant_id":  "assistant-1",
				"system_prompt": "Prompt title",
				"name":          "Named thread",
				"created_at":    "2024-01-01T00:00:00Z",
			},
		})
	}))
	defer srv.Close()

	t.Setenv("SUPABASE_URL", srv.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	store := NewSupabaseStore()
	session, err := store.GetByID(context.Background(), "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session == nil {
		t.Fatal("expected session, got nil")
	}
	if session.APIKeyHash != "hash-1" {
		t.Fatalf("expected api_key_hash to decode, got %q", session.APIKeyHash)
	}
	if session.AssistantID == nil || *session.AssistantID != "assistant-1" {
		t.Fatalf("expected assistant_id to decode, got %+v", session.AssistantID)
	}
	if session.SystemPrompt == nil || *session.SystemPrompt != "Prompt title" {
		t.Fatalf("expected system_prompt to decode, got %+v", session.SystemPrompt)
	}
	if session.Name == nil || *session.Name != "Named thread" {
		t.Fatalf("expected name to decode, got %+v", session.Name)
	}
	if session.CreatedAt != "2024-01-01T00:00:00Z" {
		t.Fatalf("expected created_at to decode, got %q", session.CreatedAt)
	}
}
