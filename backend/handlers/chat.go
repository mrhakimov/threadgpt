package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"threadgpt/db"
	"threadgpt/openai"
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserMessage == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if len(req.UserMessage) > 32*1024 {
		http.Error(w, "message too long", http.StatusBadRequest)
		return
	}

	apiKey := APIKeyFromContext(r.Context())
	hash := APIKeyHashFromContext(r.Context())

	// Look up session: by explicit ID, or fall back to most recent (unless force_new)
	var session *db.Session
	var err error
	if req.SessionID != "" {
		session, err = db.GetSessionByID(req.SessionID)
		if err != nil {
			log.Printf("chat: GetSessionByID error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		// Verify session belongs to this user
		if session != nil && session.APIKeyHash != hash {
			http.Error(w, "unauthorized", http.StatusForbidden)
			return
		}
	} else if !req.ForceNew {
		session, err = db.GetSession(hash)
		if err != nil {
			log.Printf("chat: GetSession error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	if session == nil || session.AssistantID == nil {
		// First message: create Assistant and bind to session
		assistantID, err := openai.CreateAssistant(apiKey, req.UserMessage)
		if err != nil {
			log.Printf("chat: CreateAssistant error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if session == nil {
			session, err = db.CreateSession(hash, req.UserMessage)
			if err != nil {
				log.Printf("chat: CreateSession error: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		} else {
			// Named session exists but has no assistant yet — set system prompt
			if err := db.SetSystemPrompt(session.ID, req.UserMessage); err != nil {
				log.Printf("chat: SetSystemPrompt error: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}

		err = db.UpdateSessionAssistant(session.ID, assistantID)
		if err != nil {
			log.Printf("chat: UpdateSessionAssistant error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		session.AssistantID = &assistantID

		// Save the first user message
		_, err = db.SaveMessage(session.ID, "user", req.UserMessage, nil, nil)
		if err != nil {
			log.Printf("chat: SaveMessage error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Respond with a system message indicating the context was set
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		sessionJSON, _ := json.Marshal(map[string]string{"session_id": session.ID})
		w.Write([]byte("data: " + string(sessionJSON) + "\n\n"))

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

	threadID, err := openai.CreateThread(apiKey)
	if err != nil {
		log.Printf("chat: CreateThread error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := openai.AddMessage(apiKey, threadID, req.UserMessage); err != nil {
		log.Printf("chat: AddMessage error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Save user message
	_, err = db.SaveMessage(session.ID, "user", req.UserMessage, &threadID, nil)
	if err != nil {
		log.Printf("chat: SaveMessage error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Stream the response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	sessionJSON, _ := json.Marshal(map[string]string{"session_id": session.ID})
	w.Write([]byte("data: " + string(sessionJSON) + "\n\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	assistantText, err := openai.RunAndStream(apiKey, threadID, *session.AssistantID, w)
	if err != nil {
		// Headers already sent if streaming started, just log
		log.Printf("chat: RunAndStream error: %v", err)
		return
	}

	// Save assistant message before signaling done so fetchHistory returns it
	if assistantText != "" {
		db.SaveMessage(session.ID, "assistant", assistantText, &threadID, nil)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
