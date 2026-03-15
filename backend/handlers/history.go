package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"threadgpt/db"
)

const defaultMessagesLimit = 10

// MessageDTO is the public representation of a message — excludes internal fields like openai_thread_id.
type MessageDTO struct {
	ID              string  `json:"id"`
	SessionID       string  `json:"session_id"`
	Role            string  `json:"role"`
	Content         string  `json:"content"`
	ParentMessageID *string `json:"parent_message_id"`
	ReplyCount      int     `json:"reply_count"`
	CreatedAt       string  `json:"created_at"`
}

func toMessageDTO(m db.Message) MessageDTO {
	return MessageDTO{
		ID:              m.ID,
		SessionID:       m.SessionID,
		Role:            m.Role,
		Content:         m.Content,
		ParentMessageID: m.ParentMessageID,
		ReplyCount:      m.ReplyCount,
		CreatedAt:       m.CreatedAt,
	}
}

func toMessageDTOs(msgs []db.Message) []MessageDTO {
	out := make([]MessageDTO, len(msgs))
	for i, m := range msgs {
		out[i] = toMessageDTO(m)
	}
	return out
}

type messagesResponse struct {
	Messages []MessageDTO `json:"messages"`
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
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 100000 {
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
		if !isValidUUID(sessionID) {
			http.Error(w, "invalid session id", http.StatusBadRequest)
			return
		}
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
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{Messages: toMessageDTOs(messages), HasMore: len(messages) == limit})
		return
	}

	session, err := db.GetSession(hash)
	if err != nil {
		log.Printf("history: GetSession error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if session == nil {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{Messages: []MessageDTO{}, HasMore: false})
		return
	}

	messages, err := db.GetMessagesDesc(session.ID, limit, offset)
	if err != nil {
		log.Printf("history: GetMessagesDesc error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messagesResponse{Messages: toMessageDTOs(messages), HasMore: len(messages) == limit})
}
