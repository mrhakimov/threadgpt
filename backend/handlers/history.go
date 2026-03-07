package handlers

import (
	"encoding/json"
	"net/http"
	"threadgpt/db"
)

func HandleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// If session_id provided directly, use it
	sessionID := r.URL.Query().Get("session_id")
	if sessionID != "" {
		messages, err := db.GetMessages(sessionID)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
		return
	}

	apiKeyHash := r.URL.Query().Get("api_key_hash")
	if apiKeyHash == "" {
		apiKey := r.URL.Query().Get("api_key")
		if apiKey == "" {
			http.Error(w, "session_id or api_key_hash required", http.StatusBadRequest)
			return
		}
		apiKeyHash = hashAPIKey(apiKey)
	}

	session, err := db.GetSession(apiKeyHash)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]db.Message{})
		return
	}

	messages, err := db.GetMessages(session.ID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
