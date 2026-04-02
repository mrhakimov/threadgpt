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

type application struct {
	auth     *service.AuthService
	chat     *service.ChatService
	history  *service.HistoryService
	sessions *service.SessionService
	threads  *service.ThreadService
}

var (
	defaultApp     *application
	defaultAppOnce sync.Once
	uuidRe         = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func app() *application {
	defaultAppOnce.Do(func() {
		store := data.NewSupabaseStore()
		assistant := data.NewOpenAIClient()
		auth := service.NewAuthService()

		defaultApp = &application{
			auth:     auth,
			chat:     service.NewChatService(store, store, assistant),
			history:  service.NewHistoryService(store, store),
			sessions: service.NewSessionService(store, assistant),
			threads:  service.NewThreadService(store, store, assistant),
		}
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
