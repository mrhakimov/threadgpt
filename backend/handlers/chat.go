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
}

func HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	var req ChatRequest
	if err := decodeJSON(r, &req); err != nil || req.UserMessage == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if len(req.UserMessage) > 32*1024 {
		http.Error(w, "message too long", http.StatusBadRequest)
		return
	}
	if req.SessionID != "" && !isValidUUID(req.SessionID) {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	apiKeyHash := APIKeyHashFromContext(r.Context())
	if !app().auth.AllowChat("chat:" + apiKeyHash) {
		writeServiceError(w, domain.ErrRateLimited)
		return
	}

	err := app().chat.Handle(r.Context(), service.ChatRequest{
		APIKey:      APIKeyFromContext(r.Context()),
		APIKeyHash:  apiKeyHash,
		UserMessage: req.UserMessage,
		SessionID:   req.SessionID,
		ForceNew:    req.ForceNew,
	}, newSSEStreamWriter(w))
	if err != nil {
		writeServiceError(w, err)
	}
}
