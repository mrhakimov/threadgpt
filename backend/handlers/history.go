package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"threadgpt/db"
)

const defaultMessagesLimit = 10

type messagesResponse struct {
	Messages []db.Message `json:"messages"`
	HasMore  bool         `json:"has_more"`
}

func parsePaginationParams(r *http.Request, defaultLimit int) (limit, offset int) {
	limit = defaultLimit
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 10000 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

func HandleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := APIKeyHashFromContext(r.Context())
	limit, offset := parsePaginationParams(r, defaultMessagesLimit)

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
		messages, err := db.GetMessagesDesc(sessionID, limit, offset)
		if err != nil {
			log.Printf("history: GetMessagesDesc error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{Messages: messages, HasMore: len(messages) == limit})
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
		json.NewEncoder(w).Encode(messagesResponse{Messages: []db.Message{}, HasMore: false})
		return
	}

	messages, err := db.GetMessagesDesc(session.ID, limit, offset)
	if err != nil {
		log.Printf("history: GetMessagesDesc error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messagesResponse{Messages: messages, HasMore: len(messages) == limit})
}
