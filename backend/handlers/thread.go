package handlers

import (
	"net/http"
	"threadgpt/domain"
	"threadgpt/service"
)

type ThreadRequest struct {
	ConversationID string `json:"conversation_id"`
	UserMessage    string `json:"user_message"`
}

func HandleThread(w http.ResponseWriter, r *http.Request) {
	currentApp().HandleThread(w, r)
}

func (a *Application) HandleThread(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.handleGetThread(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	var req ThreadRequest
	if err := decodeJSON(r, &req); err != nil || req.ConversationID == "" || req.UserMessage == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if len(req.UserMessage) > 32*1024 {
		http.Error(w, "message too long", http.StatusBadRequest)
		return
	}

	apiKeyHash := APIKeyHashFromContext(r.Context())
	if !a.auth.AllowChat("chat:" + apiKeyHash) {
		writeServiceError(w, domain.ErrRateLimited)
		return
	}

	err := a.threads.Reply(r.Context(), service.ThreadRequest{
		APIKey:         APIKeyFromContext(r.Context()),
		APIKeyHash:     apiKeyHash,
		ConversationID: req.ConversationID,
		UserMessage:    req.UserMessage,
	}, newSSEStreamWriter(w))
	if err != nil {
		writeServiceError(w, err)
	}
}

func (a *Application) handleGetThread(w http.ResponseWriter, r *http.Request) {
	conversationID := r.URL.Query().Get("conversation_id")
	if conversationID == "" {
		http.Error(w, "missing conversation_id", http.StatusBadRequest)
		return
	}

	limit, offset := parsePaginationParams(r, defaultMessagesLimit, maxPaginationOffset)
	messages, err := a.threads.Get(r.Context(), APIKeyFromContext(r.Context()), APIKeyHashFromContext(r.Context()), conversationID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, messagesResponse{
		Messages: toMessageDTOs(messages),
		HasMore:  len(messages) == limit,
	})
}
