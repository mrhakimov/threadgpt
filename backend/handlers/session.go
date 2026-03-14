package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"threadgpt/db"
	"threadgpt/openai"
)

type SessionRequest struct{}

type SessionResponse struct {
	SessionID    string  `json:"session_id"`
	AssistantID  *string `json:"assistant_id"`
	SystemPrompt *string `json:"system_prompt"`
	Name         *string `json:"name"`
	IsNew        bool    `json:"is_new"`
}

func hashAPIKey(apiKey string) string {
	h := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", h)
}

// HandleSessions handles GET (list) and POST (create named session)
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
	apiKeyHash := APIKeyHashFromContext(r.Context())

	sessions, err := db.GetSessions(apiKeyHash)
	if err != nil {
		log.Printf("sessions: GetSessions error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type item struct {
		SessionID    string  `json:"session_id"`
		AssistantID  *string `json:"assistant_id"`
		SystemPrompt *string `json:"system_prompt"`
		Name         *string `json:"name"`
		CreatedAt    string  `json:"created_at"`
	}

	result := make([]item, len(sessions))
	for i, s := range sessions {
		result[i] = item{
			SessionID:    s.ID,
			AssistantID:  s.AssistantID,
			SystemPrompt: s.SystemPrompt,
			Name:         s.Name,
			CreatedAt:    s.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type CreateSessionRequest struct {
	Name string `json:"name"`
}

func handleCreateNamedSession(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1*1024)
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	hash := APIKeyHashFromContext(r.Context())
	session, err := db.CreateNamedSession(hash, name)
	if err != nil {
		log.Printf("sessions: CreateNamedSession error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := SessionResponse{
		SessionID: session.ID,
		Name:      session.Name,
		IsNew:     true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleSessionByID handles GET (fetch), PATCH (rename/update) and DELETE for a specific session ID
func HandleSessionByID(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Path[len("/api/sessions/"):]
	if sessionID == "" {
		http.Error(w, "session id required", http.StatusBadRequest)
		return
	}

	hash := APIKeyHashFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		session, err := db.GetSessionByID(sessionID)
		if err != nil {
			log.Printf("sessions: GetSessionByID error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if session == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if session.APIKeyHash != hash {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		resp := SessionResponse{
			SessionID:    session.ID,
			AssistantID:  session.AssistantID,
			SystemPrompt: session.SystemPrompt,
			Name:         session.Name,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

	case http.MethodPatch:
		// Ownership check first
		session, err := db.GetSessionByID(sessionID)
		if err != nil {
			log.Printf("sessions: GetSessionByID error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if session == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if session.APIKeyHash != hash {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
		var req struct {
			Name         string `json:"name"`
			SystemPrompt string `json:"system_prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
		if req.Name != "" {
			if err := db.RenameSession(sessionID, req.Name); err != nil {
				log.Printf("sessions: RenameSession error: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
		if req.SystemPrompt != "" {
			apiKey := APIKeyFromContext(r.Context())
			if session.AssistantID != nil {
				if err := openai.UpdateAssistantInstructions(apiKey, *session.AssistantID, req.SystemPrompt); err != nil {
					log.Printf("sessions: UpdateAssistantInstructions error: %v", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
			}
			if err := db.SetSystemPrompt(sessionID, req.SystemPrompt); err != nil {
				log.Printf("sessions: SetSystemPrompt error: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		// Ownership check first
		session, err := db.GetSessionByID(sessionID)
		if err != nil {
			log.Printf("sessions: GetSessionByID error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if session == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if session.APIKeyHash != hash {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := db.DeleteSession(sessionID); err != nil {
			log.Printf("sessions: DeleteSession error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleSession handles the legacy single-session init (used by useChat on startup)
func HandleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := APIKeyHashFromContext(r.Context())
	session, err := db.GetSession(hash)
	if err != nil {
		log.Printf("session: GetSession error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := SessionResponse{}
	if session == nil {
		resp.IsNew = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp.SessionID = session.ID
	resp.AssistantID = session.AssistantID
	resp.SystemPrompt = session.SystemPrompt
	resp.Name = session.Name

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
