package handlers

import (
	"net/http"
	"strconv"
)

type SessionResponse struct {
	SessionID    string  `json:"session_id"`
	AssistantID  *string `json:"assistant_id"`
	SystemPrompt *string `json:"system_prompt"`
	Name         *string `json:"name"`
	IsNew        bool    `json:"is_new"`
}

const defaultSessionsLimit = 20

func HandleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListSessions(w, r)
	case http.MethodPost:
		handleCreateNamedSession(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleListSessions(w http.ResponseWriter, r *http.Request) {
	limit := defaultSessionsLimit
	offset := 0
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if value := r.URL.Query().Get("offset"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	sessions, err := app().sessions.List(r.Context(), APIKeyHashFromContext(r.Context()), limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	type item struct {
		SessionID    string  `json:"session_id"`
		AssistantID  *string `json:"assistant_id"`
		SystemPrompt *string `json:"system_prompt"`
		Name         *string `json:"name"`
		CreatedAt    string  `json:"created_at"`
	}

	items := make([]item, len(sessions))
	for i, session := range sessions {
		items[i] = item{
			SessionID:    session.ID,
			AssistantID:  session.AssistantID,
			SystemPrompt: session.SystemPrompt,
			Name:         session.Name,
			CreatedAt:    session.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, struct {
		Sessions []item `json:"sessions"`
		HasMore  bool   `json:"has_more"`
	}{
		Sessions: items,
		HasMore:  len(sessions) == limit,
	})
}

type CreateSessionRequest struct {
	Name string `json:"name"`
}

func handleCreateNamedSession(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1*1024)
	var req CreateSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	name := req.Name
	if name == "" {
		name = "New conversation"
	}
	if len(name) > 256 {
		http.Error(w, "name too long", http.StatusBadRequest)
		return
	}

	session, err := app().sessions.CreateNamed(r.Context(), APIKeyHashFromContext(r.Context()), name)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, SessionResponse{
		SessionID: session.ID,
		Name:      session.Name,
		IsNew:     true,
	})
}

func HandleSessionByID(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Path[len("/api/sessions/"):]
	if sessionID == "" {
		http.Error(w, "session id required", http.StatusBadRequest)
		return
	}
	if !isValidUUID(sessionID) {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	apiKeyHash := APIKeyHashFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		session, err := app().sessions.GetByID(r.Context(), apiKeyHash, sessionID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, SessionResponse{
			SessionID:    session.ID,
			AssistantID:  session.AssistantID,
			SystemPrompt: session.SystemPrompt,
			Name:         session.Name,
		})

	case http.MethodPatch:
		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
		var req struct {
			Name         string `json:"name"`
			SystemPrompt string `json:"system_prompt"`
		}
		if err := decodeJSON(r, &req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if req.Name == "" && req.SystemPrompt == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if len(req.Name) > 256 {
			http.Error(w, "name too long", http.StatusBadRequest)
			return
		}
		if len(req.SystemPrompt) > 64*1024 {
			http.Error(w, "system prompt too long", http.StatusBadRequest)
			return
		}

		if err := app().sessions.Update(r.Context(), APIKeyFromContext(r.Context()), apiKeyHash, sessionID, req.Name, req.SystemPrompt); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		if err := app().sessions.Delete(r.Context(), apiKeyHash, sessionID); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, err := app().sessions.GetCurrent(r.Context(), APIKeyHashFromContext(r.Context()))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if session == nil {
		writeJSON(w, http.StatusOK, SessionResponse{IsNew: true})
		return
	}

	writeJSON(w, http.StatusOK, SessionResponse{
		SessionID:    session.ID,
		AssistantID:  session.AssistantID,
		SystemPrompt: session.SystemPrompt,
		Name:         session.Name,
	})
}
