package handlers

import (
	"net/http"
	"strconv"
	"threadgpt/domain"
)

const defaultMessagesLimit = 10
const maxPaginationOffset = 10000

type ConversationPreviewDTO struct {
	ConversationID   string `json:"conversation_id"`
	SessionID        string `json:"session_id"`
	UserMessage      string `json:"user_message"`
	AssistantMessage string `json:"assistant_message"`
	ReplyCount       int    `json:"reply_count"`
	CreatedAt        string `json:"created_at"`
}

type MessageDTO struct {
	ID         string `json:"id"`
	SessionID  string `json:"session_id"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	ReplyCount int    `json:"reply_count"`
	CreatedAt  string `json:"created_at"`
}

type conversationsResponse struct {
	Conversations []ConversationPreviewDTO `json:"conversations"`
	HasMore       bool                     `json:"has_more"`
}

type messagesResponse struct {
	Messages []MessageDTO `json:"messages"`
	HasMore  bool         `json:"has_more"`
}

func HandleHistory(w http.ResponseWriter, r *http.Request) {
	currentApp().HandleHistory(w, r)
}

func (a *Application) HandleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, newAPIError(http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed."))
		return
	}

	sessionID := r.Header.Get("X-Session-ID")
	if sessionID != "" && !isValidUUID(sessionID) {
		writeAPIError(w, newAPIError(http.StatusBadRequest, "invalid_session_id", "The session ID was invalid."))
		return
	}

	limit, offset := parsePaginationParams(r, defaultMessagesLimit, maxPaginationOffset)
	conversations, err := a.history.Get(r.Context(), APIKeyFromContext(r.Context()), APIKeyHashFromContext(r.Context()), sessionID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, conversationsResponse{
		Conversations: toConversationPreviewDTOs(conversations),
		HasMore:       len(conversations) == limit,
	})
}

func parsePaginationParams(r *http.Request, defaultLimit, maxOffset int) (limit, offset int) {
	limit = defaultLimit
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if value := r.URL.Query().Get("offset"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 && parsed <= maxOffset {
			offset = parsed
		}
	}
	return limit, offset
}

func toMessageDTO(message domain.Message) MessageDTO {
	return MessageDTO{
		ID:         message.ID,
		SessionID:  message.SessionID,
		Role:       message.Role,
		Content:    message.Content,
		ReplyCount: message.ReplyCount,
		CreatedAt:  message.CreatedAt,
	}
}

func toMessageDTOs(messages []domain.Message) []MessageDTO {
	items := make([]MessageDTO, len(messages))
	for i, message := range messages {
		items[i] = toMessageDTO(message)
	}
	return items
}

func toConversationPreviewDTO(preview domain.ConversationPreview) ConversationPreviewDTO {
	return ConversationPreviewDTO{
		ConversationID:   preview.ConversationID,
		SessionID:        preview.SessionID,
		UserMessage:      preview.UserMessage,
		AssistantMessage: preview.AssistantMessage,
		ReplyCount:       preview.ReplyCount,
		CreatedAt:        preview.CreatedAt,
	}
}

func toConversationPreviewDTOs(previews []domain.ConversationPreview) []ConversationPreviewDTO {
	items := make([]ConversationPreviewDTO, len(previews))
	for i, preview := range previews {
		items[i] = toConversationPreviewDTO(preview)
	}
	return items
}
