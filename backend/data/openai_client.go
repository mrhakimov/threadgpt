package data

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"threadgpt/domain"
	"threadgpt/repository"
	"time"
)

const openAIBaseURL = "https://api.openai.com/v1"

type OpenAIClient struct {
	client       *http.Client
	streamClient *http.Client
}

func NewOpenAIClient() *OpenAIClient {
	return &OpenAIClient{
		client:       &http.Client{Timeout: 30 * time.Second},
		streamClient: &http.Client{},
	}
}

func (c *OpenAIClient) ValidateAPIKey(ctx context.Context, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openAIBaseURL+"/models?limit=1", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode == 401 {
		return domain.ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		return domain.ErrInternal
	}
	return nil
}

func (c *OpenAIClient) CreateConversation(ctx context.Context, apiKey, systemPrompt string) (string, error) {
	payload := map[string]any{}
	if strings.TrimSpace(systemPrompt) != "" {
		payload["items"] = []map[string]any{
			{
				"type":    "message",
				"role":    "developer",
				"content": systemPrompt,
			},
		}
	}

	var result struct {
		ID string `json:"id"`
	}
	err := c.doRequest(ctx, apiKey, http.MethodPost, "/conversations", payload, &result)
	return result.ID, err
}

func (c *OpenAIClient) ListMessages(ctx context.Context, apiKey, conversationID string) ([]domain.Message, error) {
	var messages []domain.Message
	var after string

	for {
		params := url.Values{
			"order": {"asc"},
			"limit": {"100"},
		}
		if after != "" {
			params.Set("after", after)
		}

		var result struct {
			Data    []conversationItem `json:"data"`
			HasMore bool               `json:"has_more"`
			LastID  string             `json:"last_id"`
		}
		if err := c.doRequest(ctx, apiKey, http.MethodGet, "/conversations/"+conversationID+"/items?"+params.Encode(), nil, &result); err != nil {
			return nil, err
		}

		for _, item := range result.Data {
			if item.Type != "message" {
				continue
			}
			if item.Role != "user" && item.Role != "assistant" {
				continue
			}

			messages = append(messages, domain.Message{
				ID:        item.ID,
				Role:      item.Role,
				Content:   item.textContent(),
				CreatedAt: formatUnixTimestamp(item.CreatedAt),
			})
		}

		if !result.HasMore || result.LastID == "" {
			break
		}
		after = result.LastID
	}

	return messages, nil
}

func (c *OpenAIClient) RunAndStream(ctx context.Context, apiKey, conversationID, userMessage string, stream repository.StreamWriter) error {
	payload, err := json.Marshal(map[string]any{
		"model":        "gpt-4o",
		"conversation": conversationID,
		"input": []map[string]any{
			{
				"role":    "user",
				"content": userMessage,
			},
		},
		"stream": true,
	})
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(runCtx, http.MethodPost, openAIBaseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("openai stream error %d on POST /responses", resp.StatusCode)
		return domain.ErrInternal
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case "response.output_text.delta":
			chunk, _ := event["delta"].(string)
			if chunk == "" {
				continue
			}
			if err := stream.WriteChunk(chunk); err != nil {
				return err
			}
		case "error":
			log.Printf("openai response stream returned error event")
			return domain.ErrInternal
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (c *OpenAIClient) DeleteConversation(ctx context.Context, apiKey, conversationID string) error {
	return c.doRequest(ctx, apiKey, http.MethodDelete, "/conversations/"+conversationID, nil, nil)
}

type conversationItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Role      string `json:"role"`
	CreatedAt int64  `json:"created_at"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func (i conversationItem) textContent() string {
	var b strings.Builder
	for _, part := range i.Content {
		switch part.Type {
		case "input_text", "output_text":
			b.WriteString(part.Text)
		}
	}
	return b.String()
}

func formatUnixTimestamp(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func (c *OpenAIClient) doRequest(ctx context.Context, apiKey, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, openAIBaseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		log.Printf("openai error %d on %s %s: %s", resp.StatusCode, method, path, truncateForLog(respBody))
		return domain.ErrInternal
	}
	if result != nil {
		if len(respBody) == 0 {
			return nil
		}
		return json.Unmarshal(respBody, result)
	}
	return nil
}
