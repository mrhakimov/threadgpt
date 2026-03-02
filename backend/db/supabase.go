package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Session struct {
	ID           string  `json:"id"`
	APIKeyHash   string  `json:"api_key_hash"`
	AssistantID  *string `json:"assistant_id"`
	SystemPrompt *string `json:"system_prompt"`
	CreatedAt    string  `json:"created_at"`
}

type Message struct {
	ID             string  `json:"id"`
	SessionID      string  `json:"session_id"`
	Role           string  `json:"role"`
	Content        string  `json:"content"`
	OpenAIThreadID *string `json:"openai_thread_id"`
	ParentMessageID *string `json:"parent_message_id"`
	CreatedAt      string  `json:"created_at"`
}

func supabaseURL() string {
	return os.Getenv("SUPABASE_URL")
}

func supabaseKey() string {
	if v := os.Getenv("SUPABASE_SERVICE_KEY"); v != "" {
		return v
	}
	return os.Getenv("SUPABASE_SECRET_KEY")
}

func doRequest(method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	url := supabaseURL() + "/rest/v1/" + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("apikey", supabaseKey())
	req.Header.Set("Authorization", "Bearer "+supabaseKey())
	req.Header.Set("Content-Type", "application/json")
	if result != nil {
		req.Header.Set("Accept", "application/json")
	}
	if method == "POST" {
		req.Header.Set("Prefer", "return=representation")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("supabase error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, result)
	}
	return nil
}

func GetSession(apiKeyHash string) (*Session, error) {
	var sessions []Session
	err := doRequest("GET", "sessions?api_key_hash=eq."+apiKeyHash+"&limit=1", nil, &sessions)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	return &sessions[0], nil
}

func CreateSession(apiKeyHash, systemPrompt string) (*Session, error) {
	payload := map[string]string{
		"api_key_hash":  apiKeyHash,
		"system_prompt": systemPrompt,
	}
	var sessions []Session
	err := doRequest("POST", "sessions", payload, &sessions)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no session returned")
	}
	return &sessions[0], nil
}

func UpdateSessionAssistant(sessionID, assistantID string) error {
	payload := map[string]string{"assistant_id": assistantID}
	return doRequest("PATCH", "sessions?id=eq."+sessionID, payload, nil)
}

func SaveMessage(sessionID, role, content string, openaiThreadID, parentMessageID *string) (*Message, error) {
	payload := map[string]any{
		"session_id": sessionID,
		"role":       role,
		"content":    content,
	}
	if openaiThreadID != nil {
		payload["openai_thread_id"] = *openaiThreadID
	}
	if parentMessageID != nil {
		payload["parent_message_id"] = *parentMessageID
	}

	var messages []Message
	err := doRequest("POST", "messages", payload, &messages)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("no message returned")
	}
	return &messages[0], nil
}

func GetMessages(sessionID string) ([]Message, error) {
	var messages []Message
	err := doRequest("GET", "messages?session_id=eq."+sessionID+"&parent_message_id=is.null&order=created_at.asc", nil, &messages)
	return messages, err
}

func GetThreadMessages(parentMessageID string) ([]Message, error) {
	var messages []Message
	err := doRequest("GET", "messages?parent_message_id=eq."+parentMessageID+"&order=created_at.asc", nil, &messages)
	return messages, err
}

func GetMessageByID(messageID string) (*Message, error) {
	var messages []Message
	err := doRequest("GET", "messages?id=eq."+messageID+"&limit=1", nil, &messages)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return &messages[0], nil
}
