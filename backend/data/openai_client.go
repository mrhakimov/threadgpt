package data

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"threadgpt/domain"
	"threadgpt/repository"
	"time"
)

const openAIBaseURL = "https://api.openai.com/v1"

type OpenAIClient struct {
	client *http.Client
}

func NewOpenAIClient() *OpenAIClient {
	return &OpenAIClient{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *OpenAIClient) CreateAssistant(ctx context.Context, apiKey, instructions string) (string, error) {
	var result struct {
		ID string `json:"id"`
	}

	err := c.doRequest(ctx, apiKey, http.MethodPost, "/assistants", map[string]any{
		"model":        "gpt-4o",
		"instructions": instructions,
		"name":         "ThreadGPT Assistant",
	}, &result)
	return result.ID, err
}

func (c *OpenAIClient) CreateThread(ctx context.Context, apiKey string) (string, error) {
	var result struct {
		ID string `json:"id"`
	}

	err := c.doRequest(ctx, apiKey, http.MethodPost, "/threads", map[string]any{}, &result)
	return result.ID, err
}

func (c *OpenAIClient) AddUserMessage(ctx context.Context, apiKey, threadID, content string) error {
	return c.doRequest(ctx, apiKey, http.MethodPost, "/threads/"+threadID+"/messages", map[string]any{
		"role":    "user",
		"content": content,
	}, nil)
}

func (c *OpenAIClient) AddAssistantMessage(ctx context.Context, apiKey, threadID, content string) error {
	return c.doRequest(ctx, apiKey, http.MethodPost, "/threads/"+threadID+"/messages", map[string]any{
		"role":    "assistant",
		"content": content,
	}, nil)
}

func (c *OpenAIClient) RunAndStream(ctx context.Context, apiKey, threadID, assistantID string, stream repository.StreamWriter) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"assistant_id": assistantID,
		"stream":       true,
	})
	if err != nil {
		return "", err
	}

	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(runCtx, http.MethodPost, openAIBaseURL+"/threads/"+threadID+"/runs", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "assistants=v2")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("openai stream error %d on POST /threads/.../runs", resp.StatusCode)
		return "", domain.ErrInternal
	}

	var fullText strings.Builder
	scanner := bufio.NewScanner(resp.Body)
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

		objectType, _ := event["object"].(string)
		if objectType != "thread.message.delta" {
			continue
		}

		delta, ok := event["delta"].(map[string]any)
		if !ok {
			continue
		}
		contentItems, ok := delta["content"].([]any)
		if !ok {
			continue
		}
		for _, item := range contentItems {
			contentMap, ok := item.(map[string]any)
			if !ok || contentMap["type"] != "text" {
				continue
			}
			textMap, ok := contentMap["text"].(map[string]any)
			if !ok {
				continue
			}
			chunk, _ := textMap["value"].(string)
			if chunk == "" {
				continue
			}
			fullText.WriteString(chunk)
			if err := stream.WriteChunk(chunk); err != nil {
				return fullText.String(), err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fullText.String(), err
	}

	return fullText.String(), nil
}

func (c *OpenAIClient) UpdateAssistantInstructions(ctx context.Context, apiKey, assistantID, instructions string) error {
	return c.doRequest(ctx, apiKey, http.MethodPost, "/assistants/"+assistantID, map[string]any{
		"instructions": instructions,
	}, nil)
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
	req.Header.Set("OpenAI-Beta", "assistants=v2")

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
		log.Printf("openai error %d on %s %s", resp.StatusCode, method, path)
		return domain.ErrInternal
	}
	if result != nil {
		return json.Unmarshal(respBody, result)
	}
	return nil
}
