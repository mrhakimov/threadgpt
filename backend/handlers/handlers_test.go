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
	"time"
)

const (
	testAPIKey    = "sk-test-api-key-1234567890"
	testUserHash  = "user-hash"
	otherUserHash = "other-user-hash"
)

func TestHandleChat_FirstMessageCreatesAssistantSessionAndConfirmation(t *testing.T) {
	state := newFakeSupabaseState()
	server := state.newServer()
	defer server.Close()

	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	restore := mockOpenAITransport(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost && req.URL.Path == "/v1/assistants" {
			return jsonHTTPResponse(http.StatusOK, map[string]string{"id": "assistant-1"}), nil
		}
		t.Fatalf("unexpected OpenAI request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})
	defer restore()

	body := bytes.NewBufferString(`{"user_message":"You are a concise assistant","force_new":true}`)
	req := newAuthedRequest(t, http.MethodPost, "/api/chat", body, testUserHash)
	rec := httptest.NewRecorder()

	HandleChat(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	payload := rec.Body.String()
	if !strings.Contains(payload, `"session_id":"00000000-0000-0000-0000-000000000001"`) {
		t.Fatalf("expected session id in SSE payload, got %q", payload)
	}
	if !strings.Contains(payload, "Context set!") {
		t.Fatalf("expected confirmation message in SSE payload, got %q", payload)
	}
	if !strings.Contains(payload, "data: [DONE]") {
		t.Fatalf("expected DONE event in SSE payload, got %q", payload)
	}

	snapshot := state.snapshot()
	if len(snapshot.sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(snapshot.sessions))
	}
	session := snapshot.sessions[0]
	if session.APIKeyHash != testUserHash {
		t.Fatalf("expected session owner %q, got %q", testUserHash, session.APIKeyHash)
	}
	if session.AssistantID == nil || *session.AssistantID != "assistant-1" {
		t.Fatalf("expected assistant id to be saved, got %+v", session.AssistantID)
	}
	if session.SystemPrompt == nil || *session.SystemPrompt != "You are a concise assistant" {
		t.Fatalf("expected system prompt to be saved, got %+v", session.SystemPrompt)
	}

	if len(snapshot.messages) != 2 {
		t.Fatalf("expected 2 saved messages, got %d", len(snapshot.messages))
	}
	if snapshot.messages[0].Role != "user" || snapshot.messages[0].Content != "You are a concise assistant" {
		t.Fatalf("unexpected first message: %+v", snapshot.messages[0])
	}
	if snapshot.messages[1].Role != "assistant" || !strings.Contains(snapshot.messages[1].Content, "Context set!") {
		t.Fatalf("unexpected assistant confirmation message: %+v", snapshot.messages[1])
	}
}

func TestHandleChat_ExistingSessionStreamsAssistantReply(t *testing.T) {
	assistantID := "assistant-1"
	state := newFakeSupabaseState()
	state.sessions = append(state.sessions, domain.Session{
		ID:          validUUID(1),
		APIKeyHash:  testUserHash,
		AssistantID: &assistantID,
		CreatedAt:   state.nextTimestamp(),
	})
	server := state.newServer()
	defer server.Close()

	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	restore := mockOpenAITransport(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/v1/threads":
			return jsonHTTPResponse(http.StatusOK, map[string]string{"id": "thread-1"}), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/threads/thread-1/messages":
			return jsonHTTPResponse(http.StatusOK, map[string]any{"ok": true}), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/threads/thread-1/runs":
			return sseHTTPResponse(strings.Join([]string{
				`data: {"object":"thread.message.delta","delta":{"content":[{"type":"text","text":{"value":"Hello"}}]}}`,
				``,
				`data: {"object":"thread.message.delta","delta":{"content":[{"type":"text","text":{"value":" there"}}]}}`,
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

	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	payload := rec.Body.String()
	if !strings.Contains(payload, `"session_id":"00000000-0000-0000-0000-000000000001"`) {
		t.Fatalf("expected session id in stream, got %q", payload)
	}
	if !strings.Contains(payload, `"chunk":"Hello"`) || !strings.Contains(payload, `"chunk":" there"`) {
		t.Fatalf("expected assistant chunks in stream, got %q", payload)
	}
	if !strings.Contains(payload, "data: [DONE]") {
		t.Fatalf("expected DONE event in stream, got %q", payload)
	}

	snapshot := state.snapshot()
	if len(snapshot.messages) != 2 {
		t.Fatalf("expected 2 saved messages, got %d", len(snapshot.messages))
	}
	if snapshot.messages[0].Role != "user" || snapshot.messages[0].OpenAIThreadID == nil || *snapshot.messages[0].OpenAIThreadID != "thread-1" {
		t.Fatalf("unexpected saved user message: %+v", snapshot.messages[0])
	}
	if snapshot.messages[1].Role != "assistant" || snapshot.messages[1].Content != "Hello there" {
		t.Fatalf("unexpected saved assistant message: %+v", snapshot.messages[1])
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

func TestHandleSessionByID_PatchUpdatesAssistantAndSystemPrompt(t *testing.T) {
	assistantID := "assistant-1"
	initialPrompt := "Old prompt"
	state := newFakeSupabaseState()
	state.sessions = append(state.sessions, domain.Session{
		ID:           validUUID(1),
		APIKeyHash:   testUserHash,
		AssistantID:  &assistantID,
		SystemPrompt: &initialPrompt,
		CreatedAt:    state.nextTimestamp(),
	})
	state.messages = append(state.messages, domain.Message{
		ID:        validUUID(101),
		SessionID: validUUID(1),
		Role:      "user",
		Content:   initialPrompt,
		CreatedAt: state.nextTimestamp(),
	})

	server := state.newServer()
	defer server.Close()

	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_SERVICE_KEY", "test-service-key")

	restore := mockOpenAITransport(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost && req.URL.Path == "/v1/assistants/assistant-1" {
			return jsonHTTPResponse(http.StatusOK, map[string]any{"ok": true}), nil
		}
		t.Fatalf("unexpected OpenAI request: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})
	defer restore()

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
	if snapshot.messages[0].Content != "New prompt" {
		t.Fatalf("expected first user message to reflect new system prompt, got %q", snapshot.messages[0].Content)
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
	sessions []domain.Session
	messages []domain.Message
}

type fakeSupabaseState struct {
	mu          sync.Mutex
	sessions    []domain.Session
	messages    []domain.Message
	nextSession int
	nextMessage int
	nextCreated int
}

func newFakeSupabaseState() *fakeSupabaseState {
	return &fakeSupabaseState{
		nextSession: 1,
		nextMessage: 1,
	}
}

func (s *fakeSupabaseState) newServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.Path, "/rest/v1/") {
		case "sessions":
			s.handleSessions(w, r)
		case "messages":
			s.handleMessages(w, r)
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

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *fakeSupabaseState) handleMessages(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		items := append([]domain.Message(nil), s.messages...)
		q := r.URL.Query()

		if id := strings.TrimPrefix(q.Get("id"), "eq."); id != "" {
			items = filterMessages(items, func(item domain.Message) bool { return item.ID == id })
		}
		if sessionID := strings.TrimPrefix(q.Get("session_id"), "eq."); sessionID != "" {
			items = filterMessages(items, func(item domain.Message) bool { return item.SessionID == sessionID })
		}
		switch parent := q.Get("parent_message_id"); parent {
		case "is.null":
			items = filterMessages(items, func(item domain.Message) bool { return item.ParentMessageID == nil })
		case "not.is.null":
			items = filterMessages(items, func(item domain.Message) bool { return item.ParentMessageID != nil })
		default:
			if id := strings.TrimPrefix(parent, "eq."); id != "" {
				items = filterMessages(items, func(item domain.Message) bool {
					return item.ParentMessageID != nil && *item.ParentMessageID == id
				})
			}
		}
		if role := strings.TrimPrefix(q.Get("role"), "eq."); role != "" {
			items = filterMessages(items, func(item domain.Message) bool { return item.Role == role })
		}

		if q.Get("order") == "created_at.desc" {
			sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
		} else {
			sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
		}

		writeTestJSON(w, applyWindow(items, q.Get("limit"), q.Get("offset")))

	case http.MethodPost:
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)

		message := domain.Message{
			ID:        validUUID(100 + s.nextMessage),
			SessionID: stringValue(payload["session_id"]),
			Role:      stringValue(payload["role"]),
			Content:   stringValue(payload["content"]),
			CreatedAt: s.nextTimestamp(),
		}
		s.nextMessage++
		if v := stringValue(payload["openai_thread_id"]); v != "" {
			message.OpenAIThreadID = stringPtr(v)
		}
		if v := stringValue(payload["parent_message_id"]); v != "" {
			message.ParentMessageID = stringPtr(v)
		}
		s.messages = append(s.messages, message)
		writeTestJSON(w, []domain.Message{message})

	case http.MethodPatch:
		id := strings.TrimPrefix(r.URL.Query().Get("id"), "eq.")
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)

		for i := range s.messages {
			if s.messages[i].ID != id {
				continue
			}
			if v, ok := payload["content"]; ok {
				s.messages[i].Content = v
			}
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *fakeSupabaseState) snapshot() fakeSupabaseSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return fakeSupabaseSnapshot{
		sessions: append([]domain.Session(nil), s.sessions...),
		messages: append([]domain.Message(nil), s.messages...),
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

func filterMessages(items []domain.Message, keep func(domain.Message) bool) []domain.Message {
	var filtered []domain.Message
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

func stringValue(v any) string {
	s, _ := v.(string)
	return s
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
