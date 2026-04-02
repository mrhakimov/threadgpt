package handlers

import (
	"net/http"
	"strconv"
	"threadgpt/domain"
)

const defaultMessagesLimit = 10

type MessageDTO struct {
	ID              string  `json:"id"`
	SessionID       string  `json:"session_id"`
	Role            string  `json:"role"`
	Content         string  `json:"content"`
	ParentMessageID *string `json:"parent_message_id"`
	ReplyCount      int     `json:"reply_count"`
	CreatedAt       string  `json:"created_at"`
}

type messagesResponse struct {
	Messages []MessageDTO `json:"messages"`
	HasMore  bool         `json:"has_more"`
}

func HandleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.Header.Get("X-Session-ID")
	if sessionID != "" && !isValidUUID(sessionID) {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	limit, offset := parsePaginationParams(r, defaultMessagesLimit)
	messages, err := app().history.Get(r.Context(), APIKeyHashFromContext(r.Context()), sessionID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, messagesResponse{
		Messages: toMessageDTOs(messages),
		HasMore:  len(messages) == limit,
	})
}

func parsePaginationParams(r *http.Request, defaultLimit int) (limit, offset int) {
	limit = defaultLimit
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if value := r.URL.Query().Get("offset"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 && parsed <= 10000 {
			offset = parsed
		}
	}
	return limit, offset
}

func toMessageDTO(message domain.Message) MessageDTO {
	return MessageDTO{
		ID:              message.ID,
		SessionID:       message.SessionID,
		Role:            message.Role,
		Content:         message.Content,
		ParentMessageID: message.ParentMessageID,
		ReplyCount:      message.ReplyCount,
		CreatedAt:       message.CreatedAt,
	}
}

func toMessageDTOs(messages []domain.Message) []MessageDTO {
	items := make([]MessageDTO, len(messages))
	for i, message := range messages {
		items[i] = toMessageDTO(message)
	}
	return items
}
