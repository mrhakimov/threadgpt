package handlers

import (
	"encoding/json"
	"net/http"
	"threadgpt/db"
	"threadgpt/openai"
)

type ThreadRequest struct {
	APIKey          string `json:"api_key"`
	ParentMessageID string `json:"parent_message_id"`
	UserMessage     string `json:"user_message"`
}

func HandleThread(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.APIKey == "" || req.ParentMessageID == "" || req.UserMessage == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	hash := hashAPIKey(req.APIKey)
	session, err := db.GetSession(hash)
	if err != nil || session == nil || session.AssistantID == nil {
		http.Error(w, "session not found", http.StatusBadRequest)
		return
	}

	// Look up existing sub-thread messages for this parent
	existing, err := db.GetThreadMessages(req.ParentMessageID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var threadID string

	if len(existing) == 0 {
		// First reply: create a new thread starting with the parent message as context
		parentMsg, err := db.GetMessageByID(req.ParentMessageID)
		if err != nil || parentMsg == nil {
			http.Error(w, "parent message not found", http.StatusBadRequest)
			return
		}

		threadID, err = openai.CreateThread(req.APIKey)
		if err != nil {
			http.Error(w, "openai error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Seed thread with parent assistant message as context
		if parentMsg.Role == "assistant" {
			if err := openai.AddAssistantMessage(req.APIKey, threadID, parentMsg.Content); err != nil {
				http.Error(w, "openai error: "+err.Error(), http.StatusInternalServerError)
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
	if err := openai.AddMessage(req.APIKey, threadID, req.UserMessage); err != nil {
		http.Error(w, "openai error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save user message with parent reference
	parentID := req.ParentMessageID
	_, err = db.SaveMessage(session.ID, "user", req.UserMessage, &threadID, &parentID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Stream response
	assistantText, err := openai.RunAndStream(req.APIKey, threadID, *session.AssistantID, w)
	if err != nil {
		return
	}

	// Save assistant response
	if assistantText != "" {
		db.SaveMessage(session.ID, "assistant", assistantText, &threadID, &parentID)
	}
}
