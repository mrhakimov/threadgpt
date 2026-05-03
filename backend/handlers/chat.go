package handlers

import (
	"net/http"
	"threadgpt/domain"
	"threadgpt/service"
)

type ChatRequest struct {
	UserMessage string `json:"user_message"`
	SessionID   string `json:"session_id"`
	ForceNew    bool   `json:"force_new"`
	Model       string `json:"model"`
}

func HandleChat(w http.ResponseWriter, r *http.Request) {
	currentApp().HandleChat(w, r)
}

func (a *Application) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, newAPIError(http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed."))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	var req ChatRequest
	if err := decodeJSON(r, &req); err != nil || req.UserMessage == "" {
		writeAPIError(w, newAPIError(http.StatusBadRequest, "invalid_request", "The request body was invalid."))
		return
	}
	if len(req.UserMessage) > 32*1024 {
		writeAPIError(w, newAPIError(http.StatusBadRequest, "message_too_long", "Messages must be 32 KB or smaller."))
		return
	}
	if req.SessionID != "" && !isValidUUID(req.SessionID) {
		writeAPIError(w, newAPIError(http.StatusBadRequest, "invalid_session_id", "The session ID was invalid."))
		return
	}

	apiKeyHash := APIKeyHashFromContext(r.Context())
	if !a.auth.AllowChat("chat:" + apiKeyHash) {
		writeServiceError(w, domain.ErrRateLimited)
		return
	}

	err := a.chat.Handle(r.Context(), service.ChatRequest{
		APIKey:      APIKeyFromContext(r.Context()),
		APIKeyHash:  apiKeyHash,
		UserMessage: req.UserMessage,
		SessionID:   req.SessionID,
		ForceNew:    req.ForceNew,
		Model:       req.Model,
	}, newSSEStreamWriter(w))
	if err != nil {
		writeServiceError(w, err)
	}
}
