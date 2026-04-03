package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"threadgpt/service"
)

func TestNewApplication_RequireAuthUsesInjectedDependencies(t *testing.T) {
	auth := service.NewAuthService()
	auth.SetEncryptionKey([]byte("0123456789abcdef0123456789abcdef"))
	auth.SetHashKey([]byte("fedcba9876543210fedcba9876543210"))

	expectedHash := auth.HashAPIKey(testAPIKey)
	token, err := auth.Login(testAPIKey)
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}

	app := NewApplication(Dependencies{Auth: auth})

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.AddCookie(&http.Cookie{Name: "threadgpt_token", Value: token})
	rec := httptest.NewRecorder()

	var gotHash string
	app.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		gotHash = APIKeyHashFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if gotHash != expectedHash {
		t.Fatalf("expected injected auth service hash %q, got %q", expectedHash, gotHash)
	}
}
