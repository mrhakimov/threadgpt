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

	apiKeyHash := r.URL.Query().Get("api_key_hash")
	if apiKeyHash == "" {
		// Also support sending raw api_key
		apiKey := r.URL.Query().Get("api_key")
		if apiKey == "" {
			http.Error(w, "api_key_hash or api_key required", http.StatusBadRequest)
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
