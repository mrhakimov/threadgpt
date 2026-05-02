package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"threadgpt/data"
	"threadgpt/domain"
	"threadgpt/repository"
	"threadgpt/service"
)

type Dependencies struct {
	Auth     *service.AuthService
	Chat     *service.ChatService
	History  *service.HistoryService
	Sessions *service.SessionService
	Threads  *service.ThreadService
}

type Application struct {
	auth     *service.AuthService
	chat     *service.ChatService
	history  *service.HistoryService
	sessions *service.SessionService
	threads  *service.ThreadService
}

var (
	defaultApp     *Application
	defaultAppOnce sync.Once
	uuidRe         = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func NewApplication(deps Dependencies) *Application {
	return &Application{
		auth:     deps.Auth,
		chat:     deps.Chat,
		history:  deps.History,
		sessions: deps.Sessions,
		threads:  deps.Threads,
	}
}

func NewDefaultApplication() *Application {
	store := data.NewSupabaseStore()
	assistant := data.NewOpenAIClient()
	auth := service.NewAuthService()

	return NewApplication(Dependencies{
		Auth:     auth,
		Chat:     service.NewChatService(store, store, assistant),
		History:  service.NewHistoryService(store, store, assistant),
		Sessions: service.NewSessionService(store, store, assistant),
		Threads:  service.NewThreadService(store, store, assistant),
	})
}

func currentApp() *Application {
	defaultAppOnce.Do(func() {
		defaultApp = NewDefaultApplication()
	})
	return defaultApp
}

func isValidUUID(value string) bool {
	return uuidRe.MatchString(value)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func decodeJSON(r *http.Request, dest any) error {
	return json.NewDecoder(r.Body).Decode(dest)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch err {
	case nil:
		return
	case domain.ErrInvalidArgument:
		http.Error(w, "invalid request", http.StatusBadRequest)
	case domain.ErrUnauthorized:
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	case domain.ErrForbidden:
		http.Error(w, "forbidden", http.StatusForbidden)
	case domain.ErrNotFound:
		http.Error(w, "not found", http.StatusNotFound)
	case domain.ErrRateLimited:
		http.Error(w, "too many requests", http.StatusTooManyRequests)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

type sseStreamWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	started bool
}

func newSSEStreamWriter(w http.ResponseWriter) repository.StreamWriter {
	stream := &sseStreamWriter{writer: w}
	if flusher, ok := w.(http.Flusher); ok {
		stream.flusher = flusher
	}
	return stream
}

func (s *sseStreamWriter) Start(sessionID string) error {
	if s.started {
		return nil
	}
	s.started = true

	s.writer.Header().Set("Content-Type", "text/event-stream")
	s.writer.Header().Set("Cache-Control", "no-cache")
	s.writer.Header().Set("Connection", "keep-alive")

	if sessionID != "" {
		payload, _ := json.Marshal(map[string]string{"session_id": sessionID})
		_, _ = fmt.Fprintf(s.writer, "data: %s\n\n", payload)
		if s.flusher != nil {
			s.flusher.Flush()
		}
	}
	return nil
}

func (s *sseStreamWriter) WriteChunk(chunk string) error {
	payload, _ := json.Marshal(map[string]string{"chunk": chunk})
	_, _ = fmt.Fprintf(s.writer, "data: %s\n\n", payload)
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}

func (s *sseStreamWriter) Close() error {
	_, _ = fmt.Fprint(s.writer, "data: [DONE]\n\n")
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}
