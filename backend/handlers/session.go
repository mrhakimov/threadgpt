package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"threadgpt/db"
)

type SessionRequest struct {
	APIKey string `json:"api_key"`
}

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

// HandleSession handles GET (list) and POST (create named session)
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
	apiKeyHash := r.URL.Query().Get("api_key_hash")
	if apiKeyHash == "" {
		http.Error(w, "api_key_hash required", http.StatusBadRequest)
		return
	}

	sessions, err := db.GetSessions(apiKeyHash)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
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
	APIKey string `json:"api_key"`
	Name   string `json:"name"`
}

func handleCreateNamedSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.APIKey == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	name := req.Name
	if name == "" {
		name = "New conversation"
	}

	hash := hashAPIKey(req.APIKey)
	session, err := db.CreateNamedSession(hash, name)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
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

// HandleSessionByID handles PATCH (rename) and DELETE for a specific session ID
func HandleSessionByID(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Path[len("/api/sessions/"):]
	if sessionID == "" {
		http.Error(w, "session id required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if err := db.RenameSession(sessionID, req.Name); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := db.DeleteSession(sessionID); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
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

	var req SessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.APIKey == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	hash := hashAPIKey(req.APIKey)
	session, err := db.GetSession(hash)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
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
