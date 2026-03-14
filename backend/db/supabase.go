package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// ErrInternal is returned when a Supabase operation fails, hiding internal details from callers.
var ErrInternal = errors.New("internal error")

type Session struct {
	ID           string  `json:"id"`
	APIKeyHash   string  `json:"api_key_hash"`
	AssistantID  *string `json:"assistant_id"`
	SystemPrompt *string `json:"system_prompt"`
	Name         *string `json:"name"`
	CreatedAt    string  `json:"created_at"`
}

type Message struct {
	ID              string  `json:"id"`
	SessionID       string  `json:"session_id"`
	Role            string  `json:"role"`
	Content         string  `json:"content"`
	OpenAIThreadID  *string `json:"openai_thread_id"`
	ParentMessageID *string `json:"parent_message_id"`
	ReplyCount      int     `json:"reply_count"`
	CreatedAt       string  `json:"created_at"`
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
		log.Printf("supabase error %d: %s", resp.StatusCode, string(respBody))
		return ErrInternal
	}

	if result != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, result)
	}
	return nil
}

func filterPath(table string, params url.Values) string {
	return table + "?" + params.Encode()
}

func GetSession(apiKeyHash string) (*Session, error) {
	var sessions []Session
	err := doRequest("GET", filterPath("sessions", url.Values{"api_key_hash": {"eq." + apiKeyHash}, "order": {"created_at.desc"}, "limit": {"1"}}), nil, &sessions)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	return &sessions[0], nil
}

func GetSessionByID(sessionID string) (*Session, error) {
	var sessions []Session
	err := doRequest("GET", filterPath("sessions", url.Values{"id": {"eq." + sessionID}, "limit": {"1"}}), nil, &sessions)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	return &sessions[0], nil
}

func GetSessions(apiKeyHash string) ([]Session, error) {
	var sessions []Session
	err := doRequest("GET", filterPath("sessions", url.Values{"api_key_hash": {"eq." + apiKeyHash}, "order": {"created_at.desc"}}), nil, &sessions)
	return sessions, err
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

func CreateNamedSession(apiKeyHash, name string) (*Session, error) {
	payload := map[string]string{
		"api_key_hash": apiKeyHash,
		"name":         name,
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

func RenameSession(sessionID, name string) error {
	payload := map[string]string{"name": name}
	return doRequest("PATCH", filterPath("sessions", url.Values{"id": {"eq." + sessionID}}), payload, nil)
}

func SetSystemPrompt(sessionID, systemPrompt string) error {
	payload := map[string]string{"system_prompt": systemPrompt}
	if err := doRequest("PATCH", filterPath("sessions", url.Values{"id": {"eq." + sessionID}}), payload, nil); err != nil {
		return err
	}
	// Update the first user message so history reflects the change on reload
	var msgs []Message
	if err := doRequest("GET", filterPath("messages", url.Values{"session_id": {"eq." + sessionID}, "role": {"eq.user"}, "parent_message_id": {"is.null"}, "order": {"created_at.asc"}, "limit": {"1"}}), nil, &msgs); err != nil {
		return fmt.Errorf("SetSystemPrompt: get first message: %w", err)
	}
	if len(msgs) == 0 {
		fmt.Printf("SetSystemPrompt: no user message found for session %s\n", sessionID)
		return nil
	}
	fmt.Printf("SetSystemPrompt: updating message %s for session %s\n", msgs[0].ID, sessionID)
	if err := doRequest("PATCH", filterPath("messages", url.Values{"id": {"eq." + msgs[0].ID}}), map[string]string{"content": systemPrompt}, nil); err != nil {
		return fmt.Errorf("SetSystemPrompt: update message content: %w", err)
	}
	return nil
}

func UpdateSessionAssistant(sessionID, assistantID string) error {
	payload := map[string]string{"assistant_id": assistantID}
	return doRequest("PATCH", filterPath("sessions", url.Values{"id": {"eq." + sessionID}}), payload, nil)
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
	if err := doRequest("GET", filterPath("messages", url.Values{"session_id": {"eq." + sessionID}, "parent_message_id": {"is.null"}, "order": {"created_at.asc"}}), nil, &messages); err != nil {
		return nil, err
	}

	// Fetch user sub-thread messages for this session to compute follow-up counts.
	var threadMsgs []Message
	if err := doRequest("GET", filterPath("messages", url.Values{"session_id": {"eq." + sessionID}, "parent_message_id": {"not.is.null"}, "role": {"eq.user"}}), nil, &threadMsgs); err == nil {
		log.Printf("follow-up count query returned %d messages", len(threadMsgs))
		counts := make(map[string]int)
		for _, m := range threadMsgs {
			if m.ParentMessageID != nil {
				counts[*m.ParentMessageID]++
			}
		}
		for i := range messages {
			messages[i].ReplyCount = counts[messages[i].ID]
		}
	}

	return messages, nil
}

func GetThreadMessages(parentMessageID string) ([]Message, error) {
	var messages []Message
	err := doRequest("GET", filterPath("messages", url.Values{"parent_message_id": {"eq." + parentMessageID}, "order": {"created_at.asc"}}), nil, &messages)
	return messages, err
}

func DeleteSession(sessionID string) error {
	// Delete messages first (cascade)
	if err := doRequest("DELETE", filterPath("messages", url.Values{"session_id": {"eq." + sessionID}}), nil, nil); err != nil {
		return err
	}
	return doRequest("DELETE", filterPath("sessions", url.Values{"id": {"eq." + sessionID}}), nil, nil)
}

func GetMessageByID(messageID string) (*Message, error) {
	var messages []Message
	err := doRequest("GET", filterPath("messages", url.Values{"id": {"eq." + messageID}, "limit": {"1"}}), nil, &messages)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return &messages[0], nil
}
