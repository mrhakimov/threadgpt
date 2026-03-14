package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"threadgpt/db"
	"threadgpt/openai"
)

type ThreadRequest struct {
	ParentMessageID string `json:"parent_message_id"`
	UserMessage     string `json:"user_message"`
}

func HandleThread(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		handleGetThread(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	var req ThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ParentMessageID == "" || req.UserMessage == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if len(req.UserMessage) > 32*1024 {
		http.Error(w, "message too long", http.StatusBadRequest)
		return
	}

	apiKey := APIKeyFromContext(r.Context())
	hash := APIKeyHashFromContext(r.Context())

	// Look up the parent message to get its session
	parentMsg, err := db.GetMessageByID(req.ParentMessageID)
	if err != nil || parentMsg == nil {
		http.Error(w, "parent message not found", http.StatusBadRequest)
		return
	}

	// Load session from the parent message's session_id
	session, err := db.GetSessionByID(parentMsg.SessionID)
	if err != nil || session == nil || session.AssistantID == nil {
		http.Error(w, "session not found", http.StatusBadRequest)
		return
	}

	// Verify session belongs to this user
	if session.APIKeyHash != hash {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}

	// Look up existing sub-thread messages for this parent (fetch all to determine thread continuity)
	existing, err := db.GetThreadMessages(req.ParentMessageID, 1000, 0)
	if err != nil {
		log.Printf("thread: GetThreadMessages error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var threadID string

	if len(existing) == 0 {
		// First reply: create a new thread starting with the parent message as context
		threadID, err = openai.CreateThread(apiKey)
		if err != nil {
			log.Printf("thread: CreateThread error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Seed thread with parent assistant message as context
		if parentMsg.Role == "assistant" {
			if err := openai.AddAssistantMessage(apiKey, threadID, parentMsg.Content); err != nil {
				log.Printf("thread: AddAssistantMessage error: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
	} else {
		// Reuse existing thread
		if existing[0].OpenAIThreadID == nil {
			http.Error(w, "existing thread has no openai_thread_id", http.StatusInternalServerError)
			return
		}
		threadID = *existing[0].OpenAIThreadID
	}

	// Add user message
	if err := openai.AddMessage(apiKey, threadID, req.UserMessage); err != nil {
		log.Printf("thread: AddMessage error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Save user message with parent reference
	parentID := req.ParentMessageID
	_, err = db.SaveMessage(session.ID, "user", req.UserMessage, &threadID, &parentID)
	if err != nil {
		log.Printf("thread: SaveMessage error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Stream response
	assistantText, err := openai.RunAndStream(apiKey, threadID, *session.AssistantID, w)
	if err != nil {
		log.Printf("thread: RunAndStream error: %v", err)
		return
	}

	// Save assistant response before signaling done so fetchHistory returns it
	if assistantText != "" {
		db.SaveMessage(session.ID, "assistant", assistantText, &threadID, &parentID)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func handleGetThread(w http.ResponseWriter, r *http.Request) {
	hash := APIKeyHashFromContext(r.Context())
	parentMessageID := r.URL.Query().Get("parent_message_id")
	if parentMessageID == "" {
		http.Error(w, "missing parent_message_id", http.StatusBadRequest)
		return
	}

	parentMsg, err := db.GetMessageByID(parentMessageID)
	if err != nil || parentMsg == nil {
		http.Error(w, "parent message not found", http.StatusBadRequest)
		return
	}

	session, err := db.GetSessionByID(parentMsg.SessionID)
	if err != nil || session == nil || session.APIKeyHash != hash {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}

	limit, offset := parsePaginationParams(r, defaultMessagesLimit)
	messages, err := db.GetThreadMessagesDesc(parentMessageID, limit, offset)
	if err != nil {
		log.Printf("thread: GetThreadMessagesDesc error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messagesResponse{Messages: messages, HasMore: len(messages) == limit})
}
