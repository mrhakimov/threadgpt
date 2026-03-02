package handlers

import (
	"encoding/json"
	"net/http"
	"threadgpt/db"
	"threadgpt/openai"
)

type ChatRequest struct {
	APIKey      string `json:"api_key"`
	UserMessage string `json:"user_message"`
}

func HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.APIKey == "" || req.UserMessage == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	hash := hashAPIKey(req.APIKey)

	// Look up or create session
	session, err := db.GetSession(hash)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if session == nil {
		// First message: create Assistant + session
		assistantID, err := openai.CreateAssistant(req.APIKey, req.UserMessage)
		if err != nil {
			http.Error(w, "openai error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		session, err = db.CreateSession(hash, req.UserMessage)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = db.UpdateSessionAssistant(session.ID, assistantID)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		session.AssistantID = &assistantID

		// Save the first user message
		_, err = db.SaveMessage(session.ID, "user", req.UserMessage, nil, nil)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Respond with a system message indicating the context was set
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		confirmMsg := "Context set! Your assistant has been configured with this as its instructions. Send your next message to start chatting."
		chunkJSON, _ := json.Marshal(map[string]string{"chunk": confirmMsg})
		w.Write([]byte("data: " + string(chunkJSON) + "\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		db.SaveMessage(session.ID, "assistant", confirmMsg, nil, nil)
		return
	}

	// Session exists — create a new OpenAI Thread for this message
	if session.AssistantID == nil {
		http.Error(w, "session has no assistant", http.StatusInternalServerError)
		return
	}

	threadID, err := openai.CreateThread(req.APIKey)
	if err != nil {
		http.Error(w, "openai error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := openai.AddMessage(req.APIKey, threadID, req.UserMessage); err != nil {
		http.Error(w, "openai error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save user message
	_, err = db.SaveMessage(session.ID, "user", req.UserMessage, &threadID, nil)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Stream the response
	assistantText, err := openai.RunAndStream(req.APIKey, threadID, *session.AssistantID, w)
	if err != nil {
		// Headers already sent if streaming started, just log
		return
	}

	// Save assistant message
	if assistantText != "" {
		db.SaveMessage(session.ID, "assistant", assistantText, &threadID, nil)
	}
}
