package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"threadgpt/domain"
	"threadgpt/service"
	"time"
)

const (
	testAPIKey    = "sk-test-api-key-1234567890"
	testUserHash  = "user-hash"
	otherUserHash = "other-user-hash"
)

func TestHandleChat_FirstMessageStoresSystemPromptOnly(t *testing.T) {
	state := newFakeSupabaseState()
	server := state.newServer()
	defer server.Close()

	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	body := bytes.NewBufferString(`{"user_message":"You are a concise assistant","force_new":true}`)
	req := newAuthedRequest(t, http.MethodPost, "/api/chat", body, testUserHash)
	rec := httptest.NewRecorder()

	HandleChat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	payload := rec.Body.String()
	if !strings.Contains(payload, `"session_id":"00000000-0000-0000-0000-000000000001"`) {
		t.Fatalf("expected session id in SSE payload, got %q", payload)
	}
	if !strings.Contains(payload, "Context set!") {
		t.Fatalf("expected confirmation chunk in SSE payload, got %q", payload)
	}
	if !strings.Contains(payload, "data: [DONE]") {
		t.Fatalf("expected DONE event in SSE payload, got %q", payload)
	}

	snapshot := state.snapshot()
	if len(snapshot.sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(snapshot.sessions))
	}
	if snapshot.sessions[0].SystemPrompt == nil || *snapshot.sessions[0].SystemPrompt != "You are a concise assistant" {
		t.Fatalf("expected system prompt to be saved, got %+v", snapshot.sessions[0].SystemPrompt)
	}
	if len(snapshot.conversations) != 0 {
		t.Fatalf("expected no stored conversations for initial setup, got %d", len(snapshot.conversations))
	}
}

func TestHandleChat_ExistingSessionCreatesConversationAndStreamsReply(t *testing.T) {
	prompt := "You are concise"
	state := newFakeSupabaseState()
	state.sessions = append(state.sessions, domain.Session{
		ID:           validUUID(1),
		APIKeyHash:   testUserHash,
		SystemPrompt: &prompt,
		CreatedAt:    state.nextTimestamp(),
	})
	server := state.newServer()
	defer server.Close()

	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	restore := mockOpenAITransport(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/v1/conversations":
			return jsonHTTPResponse(http.StatusOK, map[string]string{"id": "conv-1"}), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/responses":
			return sseHTTPResponse(strings.Join([]string{
				`data: {"type":"response.output_text.delta","delta":"Hello"}`,
				``,
				`data: {"type":"response.output_text.delta","delta":" there"}`,
				``,
				`data: [DONE]`,
				``,
			}, "\n")), nil
		default:
			t.Fatalf("unexpected OpenAI request: %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})
	defer restore()

	body := bytes.NewBufferString(`{"user_message":"Say hi","session_id":"00000000-0000-0000-0000-000000000001"}`)
	req := newAuthedRequest(t, http.MethodPost, "/api/chat", body, testUserHash)
	rec := httptest.NewRecorder()

	HandleChat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	payload := rec.Body.String()
	if !strings.Contains(payload, `"chunk":"Hello"`) || !strings.Contains(payload, `"chunk":" there"`) {
		t.Fatalf("expected assistant chunks in stream, got %q", payload)
	}

	snapshot := state.snapshot()
	if len(snapshot.conversations) != 1 {
		t.Fatalf("expected 1 stored conversation, got %d", len(snapshot.conversations))
	}
	if snapshot.conversations[0].ConversationID != "conv-1" {
		t.Fatalf("expected stored conversation id conv-1, got %+v", snapshot.conversations[0])
	}
}

func TestHandleChat_InvalidAPIKeyReturnsStructuredErrorBeforeStreamStarts(t *testing.T) {
	prompt := "You are concise"
	state := newFakeSupabaseState()
	state.sessions = append(state.sessions, domain.Session{
		ID:           validUUID(1),
		APIKeyHash:   testUserHash,
		SystemPrompt: &prompt,
		CreatedAt:    state.nextTimestamp(),
	})
	server := state.newServer()
	defer server.Close()

	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	restore := mockOpenAITransport(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/v1/conversations":
			return jsonHTTPResponse(http.StatusOK, map[string]string{"id": "conv-1"}), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/responses":
			return jsonHTTPResponse(http.StatusUnauthorized, map[string]any{
				"error": map[string]string{
					"code":    "invalid_api_key",
					"message": "Incorrect API key provided",
				},
			}), nil
		default:
			t.Fatalf("unexpected OpenAI request: %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})
	defer restore()

	body := bytes.NewBufferString(`{"user_message":"Say hi","session_id":"00000000-0000-0000-0000-000000000001"}`)
	req := newAuthedRequest(t, http.MethodPost, "/api/chat", body, testUserHash)
	rec := httptest.NewRecorder()

	HandleChat(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected JSON error response, got %q", contentType)
	}
	if strings.Contains(rec.Body.String(), "data: ") {
		t.Fatalf("expected regular JSON error, got SSE payload %q", rec.Body.String())
	}

	var payload apiErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error.Code != "invalid_api_key" {
		t.Fatalf("expected invalid_api_key, got %+v", payload.Error)
	}
}

func TestHandleChat_QuotaExceededReturnsStructuredErrorBeforeStreamStarts(t *testing.T) {
	prompt := "You are concise"
	state := newFakeSupabaseState()
	state.sessions = append(state.sessions, domain.Session{
		ID:           validUUID(1),
		APIKeyHash:   testUserHash,
		SystemPrompt: &prompt,
		CreatedAt:    state.nextTimestamp(),
	})
	server := state.newServer()
	defer server.Close()

	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	restore := mockOpenAITransport(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/v1/conversations":
			return jsonHTTPResponse(http.StatusOK, map[string]string{"id": "conv-1"}), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/responses":
			return jsonHTTPResponse(http.StatusTooManyRequests, map[string]any{
				"error": map[string]string{
					"code":    "insufficient_quota",
					"message": "You exceeded your current quota",
				},
			}), nil
		default:
			t.Fatalf("unexpected OpenAI request: %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})
	defer restore()

	body := bytes.NewBufferString(`{"user_message":"Say hi","session_id":"00000000-0000-0000-0000-000000000001"}`)
	req := newAuthedRequest(t, http.MethodPost, "/api/chat", body, testUserHash)
	rec := httptest.NewRecorder()

	HandleChat(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rec.Code)
	}

	var payload apiErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error.Code != "quota_exceeded" {
		t.Fatalf("expected quota_exceeded, got %+v", payload.Error)
	}
}

func TestApplicationHandleAuth_InvalidAPIKeyValidationReturnsStructuredError(t *testing.T) {
	app := NewApplication(Dependencies{
		Auth:         service.NewAuthService(),
		KeyValidator: fakeKeyValidator{err: domain.ErrInvalidAPIKey},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth", bytes.NewBufferString(`{"api_key":"`+testAPIKey+`"}`))
	rec := httptest.NewRecorder()

	app.HandleAuth(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	var payload apiErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error.Code != "invalid_api_key" {
		t.Fatalf("expected invalid_api_key, got %+v", payload.Error)
	}
}

func TestHandleHistory_WithForeignSessionReturnsForbidden(t *testing.T) {
	state := newFakeSupabaseState()
	state.sessions = append(state.sessions, domain.Session{
		ID:         validUUID(1),
		APIKeyHash: otherUserHash,
		CreatedAt:  state.nextTimestamp(),
	})
	server := state.newServer()
	defer server.Close()

	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	req := newAuthedRequest(t, http.MethodGet, "/api/history", nil, testUserHash)
	req.Header.Set("X-Session-ID", validUUID(1))
	rec := httptest.NewRecorder()

	HandleHistory(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}

func TestHandleSessionByID_PatchUpdatesSystemPrompt(t *testing.T) {
	initialPrompt := "Old prompt"
	state := newFakeSupabaseState()
	state.sessions = append(state.sessions, domain.Session{
		ID:           validUUID(1),
		APIKeyHash:   testUserHash,
		SystemPrompt: &initialPrompt,
		CreatedAt:    state.nextTimestamp(),
	})
	server := state.newServer()
	defer server.Close()

	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	body := bytes.NewBufferString(`{"system_prompt":"New prompt"}`)
	req := newAuthedRequest(t, http.MethodPatch, "/api/sessions/"+validUUID(1), body, testUserHash)
	rec := httptest.NewRecorder()

	HandleSessionByID(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}

	snapshot := state.snapshot()
	if snapshot.sessions[0].SystemPrompt == nil || *snapshot.sessions[0].SystemPrompt != "New prompt" {
		t.Fatalf("expected system prompt update, got %+v", snapshot.sessions[0].SystemPrompt)
	}
}

func TestHandleThread_StreamsReplyAgainstStoredConversation(t *testing.T) {
	prompt := "You are concise"
	state := newFakeSupabaseState()
	state.sessions = append(state.sessions, domain.Session{
		ID:           validUUID(1),
		APIKeyHash:   testUserHash,
		SystemPrompt: &prompt,
		CreatedAt:    state.nextTimestamp(),
	})
	state.conversations = append(state.conversations, domain.ConversationRef{
		ConversationID: "conv-1",
		SessionID:      validUUID(1),
		CreatedAt:      state.nextTimestamp(),
	})
	server := state.newServer()
	defer server.Close()

	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	restore := mockOpenAITransport(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost && req.URL.Path == "/v1/responses" {
			return sseHTTPResponse(strings.Join([]string{
				`data: {"type":"response.output_text.delta","delta":"Follow-up"}`,
				``,
				`data: [DONE]`,
				``,
			}, "\n")), nil
		}
		t.Fatalf("unexpected OpenAI request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})
	defer restore()

	body := bytes.NewBufferString(`{"conversation_id":"conv-1","user_message":"Continue"}`)
	req := newAuthedRequest(t, http.MethodPost, "/api/thread", body, testUserHash)
	rec := httptest.NewRecorder()

	HandleThread(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"chunk":"Follow-up"`) {
		t.Fatalf("expected streamed follow-up chunk, got %q", rec.Body.String())
	}
}

func newAuthedRequest(t *testing.T, method, path string, body io.Reader, hash string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	ctx := context.WithValue(req.Context(), contextKeyAPIKey, testAPIKey)
	ctx = context.WithValue(ctx, contextKeyAPIKeyHash, hash)
	return req.WithContext(ctx)
}

func mockOpenAITransport(t *testing.T, fn func(*http.Request) (*http.Response, error)) func() {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "api.openai.com" {
			return fn(req)
		}
		return original.RoundTrip(req)
	})
	return func() {
		http.DefaultTransport = original
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonHTTPResponse(status int, body any) *http.Response {
	payload, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(payload)),
	}
}

func sseHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type fakeSupabaseSnapshot struct {
	sessions      []domain.Session
	conversations []domain.ConversationRef
}

type fakeSupabaseState struct {
	mu            sync.Mutex
	sessions      []domain.Session
	conversations []domain.ConversationRef
	nextSession   int
	nextCreated   int
}

func newFakeSupabaseState() *fakeSupabaseState {
	return &fakeSupabaseState{
		nextSession: 1,
	}
}

func (s *fakeSupabaseState) newServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.Path, "/rest/v1/") {
		case "sessions":
			s.handleSessions(w, r)
		case "session_conversations":
			s.handleConversations(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

func (s *fakeSupabaseState) handleSessions(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		items := append([]domain.Session(nil), s.sessions...)
		q := r.URL.Query()

		if id := strings.TrimPrefix(q.Get("id"), "eq."); id != "" {
			items = filterSessions(items, func(item domain.Session) bool { return item.ID == id })
		}
		if hash := strings.TrimPrefix(q.Get("api_key_hash"), "eq."); hash != "" {
			items = filterSessions(items, func(item domain.Session) bool { return item.APIKeyHash == hash })
		}
		if q.Get("order") == "created_at.desc" {
			sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
		} else {
			sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
		}

		writeTestJSON(w, applyWindow(items, q.Get("limit"), q.Get("offset")))

	case http.MethodPost:
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)

		session := domain.Session{
			ID:         validUUID(s.nextSession),
			APIKeyHash: payload["api_key_hash"],
			CreatedAt:  s.nextTimestamp(),
		}
		s.nextSession++
		if v, ok := payload["assistant_id"]; ok {
			session.AssistantID = stringPtr(v)
		}
		if v, ok := payload["system_prompt"]; ok {
			session.SystemPrompt = stringPtr(v)
		}
		if v, ok := payload["name"]; ok {
			session.Name = stringPtr(v)
		}
		s.sessions = append(s.sessions, session)
		writeTestJSON(w, []domain.Session{session})

	case http.MethodPatch:
		id := strings.TrimPrefix(r.URL.Query().Get("id"), "eq.")
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)

		for i := range s.sessions {
			if s.sessions[i].ID != id {
				continue
			}
			if v, ok := payload["assistant_id"]; ok {
				s.sessions[i].AssistantID = stringPtr(v)
			}
			if v, ok := payload["system_prompt"]; ok {
				s.sessions[i].SystemPrompt = stringPtr(v)
			}
			if v, ok := payload["name"]; ok {
				s.sessions[i].Name = stringPtr(v)
			}
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		id := strings.TrimPrefix(r.URL.Query().Get("id"), "eq.")
		var kept []domain.Session
		for _, session := range s.sessions {
			if session.ID != id {
				kept = append(kept, session)
			}
		}
		s.sessions = kept
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *fakeSupabaseState) handleConversations(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		items := append([]domain.ConversationRef(nil), s.conversations...)
		q := r.URL.Query()

		if id := strings.TrimPrefix(q.Get("conversation_id"), "eq."); id != "" {
			items = filterConversationRefs(items, func(item domain.ConversationRef) bool { return item.ConversationID == id })
		}
		if sessionID := strings.TrimPrefix(q.Get("session_id"), "eq."); sessionID != "" {
			items = filterConversationRefs(items, func(item domain.ConversationRef) bool { return item.SessionID == sessionID })
		}
		if q.Get("order") == "created_at.desc" {
			sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
		} else {
			sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
		}

		writeTestJSON(w, applyWindow(items, q.Get("limit"), q.Get("offset")))

	case http.MethodPost:
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)

		ref := domain.ConversationRef{
			ConversationID: payload["conversation_id"],
			SessionID:      payload["session_id"],
			CreatedAt:      s.nextTimestamp(),
		}
		s.conversations = append(s.conversations, ref)
		writeTestJSON(w, []domain.ConversationRef{ref})

	case http.MethodDelete:
		sessionID := strings.TrimPrefix(r.URL.Query().Get("session_id"), "eq.")
		var kept []domain.ConversationRef
		for _, ref := range s.conversations {
			if ref.SessionID != sessionID {
				kept = append(kept, ref)
			}
		}
		s.conversations = kept
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *fakeSupabaseState) snapshot() fakeSupabaseSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return fakeSupabaseSnapshot{
		sessions:      append([]domain.Session(nil), s.sessions...),
		conversations: append([]domain.ConversationRef(nil), s.conversations...),
	}
}

func (s *fakeSupabaseState) nextTimestamp() string {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ts := base.Add(time.Duration(s.nextCreated) * time.Minute).Format(time.RFC3339)
	s.nextCreated++
	return ts
}

func filterSessions(items []domain.Session, keep func(domain.Session) bool) []domain.Session {
	var filtered []domain.Session
	for _, item := range items {
		if keep(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterConversationRefs(items []domain.ConversationRef, keep func(domain.ConversationRef) bool) []domain.ConversationRef {
	var filtered []domain.ConversationRef
	for _, item := range items {
		if keep(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func applyWindow[T any](items []T, limitRaw, offsetRaw string) []T {
	limit := len(items)
	offset := 0

	if limitRaw != "" {
		var parsed int
		for _, c := range limitRaw {
			parsed = parsed*10 + int(c-'0')
		}
		limit = parsed
	}
	if offsetRaw != "" {
		for _, c := range offsetRaw {
			offset = offset*10 + int(c-'0')
		}
	}

	if offset >= len(items) {
		return []T{}
	}
	items = items[offset:]
	if limit < len(items) {
		items = items[:limit]
	}
	return items
}

func writeTestJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func stringPtr(v string) *string {
	return &v
}

func validUUID(n int) string {
	return "00000000-0000-0000-0000-" + leftPad(n)
}

func leftPad(n int) string {
	s := "000000000000"
	digits := []byte(s)
	for i := len(digits) - 1; i >= 0 && n > 0; i-- {
		digits[i] = byte('0' + (n % 10))
		n /= 10
	}
	return string(digits)
}

type fakeKeyValidator struct {
	err error
}

func (f fakeKeyValidator) ValidateAPIKey(context.Context, string) error {
	return f.err
}
