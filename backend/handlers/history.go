package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"threadgpt/db"
)

func HandleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := APIKeyHashFromContext(r.Context())

	// Optional session_id filter — passed via X-Session-ID header to avoid URL exposure
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID != "" {
		session, err := db.GetSessionByID(sessionID)
		if err != nil {
			log.Printf("history: GetSessionByID error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if session == nil || session.APIKeyHash != hash {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		messages, err := db.GetMessages(sessionID)
		if err != nil {
			log.Printf("history: GetMessages error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
		return
	}

	session, err := db.GetSession(hash)
	if err != nil {
		log.Printf("history: GetSession error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]db.Message{})
		return
	}

	messages, err := db.GetMessages(session.ID)
	if err != nil {
		log.Printf("history: GetMessages error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
