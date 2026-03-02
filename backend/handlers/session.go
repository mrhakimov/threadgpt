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
	SessionID   string  `json:"session_id"`
	AssistantID *string `json:"assistant_id"`
	SystemPrompt *string `json:"system_prompt"`
	IsNew       bool    `json:"is_new"`
}

func hashAPIKey(apiKey string) string {
	h := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", h)
}

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
		// Session will be created when the first message is sent
		// Return a temporary indicator
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp.SessionID = session.ID
	resp.AssistantID = session.AssistantID
	resp.SystemPrompt = session.SystemPrompt

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
