package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// ErrInternal is returned when an OpenAI API call fails, hiding internal details from callers.
var ErrInternal = errors.New("internal error")

const baseURL = "https://api.openai.com/v1"

func doRequest(apiKey, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "assistants=v2")

	client := &http.Client{Timeout: 30 * time.Second}
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
		log.Printf("openai error %d on %s %s", resp.StatusCode, method, path)
		return ErrInternal
	}

	if result != nil {
		return json.Unmarshal(respBody, result)
	}
	return nil
}

func CreateAssistant(apiKey, instructions string) (string, error) {
	payload := map[string]any{
		"model":        "gpt-4o",
		"instructions": instructions,
		"name":         "ThreadGPT Assistant",
	}
	var result struct {
		ID string `json:"id"`
	}
	err := doRequest(apiKey, "POST", "/assistants", payload, &result)
	return result.ID, err
}

func CreateThread(apiKey string) (string, error) {
	var result struct {
		ID string `json:"id"`
	}
	err := doRequest(apiKey, "POST", "/threads", map[string]any{}, &result)
	return result.ID, err
}

func AddMessage(apiKey, threadID, content string) error {
	payload := map[string]any{
		"role":    "user",
		"content": content,
	}
	return doRequest(apiKey, "POST", "/threads/"+threadID+"/messages", payload, nil)
}

// RunAndStream creates a run with streaming and writes SSE chunks to the ResponseWriter.
// It returns the full assistant response text.
func RunAndStream(apiKey, threadID, assistantID string, w http.ResponseWriter) (string, error) {
	payload := map[string]any{
		"assistant_id": assistantID,
		"stream":       true,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/threads/"+threadID+"/runs", bytes.NewReader(b))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "assistants=v2")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("openai stream error %d on POST /threads/.../runs", resp.StatusCode)
		return "", ErrInternal
	}

	// Set SSE headers on the response writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, canFlush := w.(http.Flusher)

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

		// Extract text delta from thread.message.delta events
		eventType, _ := event["object"].(string)
		if eventType == "thread.message.delta" {
			delta, ok := event["delta"].(map[string]any)
			if !ok {
				continue
			}
			contentArr, ok := delta["content"].([]any)
			if !ok {
				continue
			}
			for _, c := range contentArr {
				cMap, ok := c.(map[string]any)
				if !ok {
					continue
				}
				if cMap["type"] == "text" {
					textObj, ok := cMap["text"].(map[string]any)
					if !ok {
						continue
					}
					chunk, _ := textObj["value"].(string)
					if chunk != "" {
						fullText.WriteString(chunk)
						chunkJSON, _ := json.Marshal(map[string]string{"chunk": chunk})
						fmt.Fprintf(w, "data: %s\n\n", chunkJSON)
						if canFlush {
							flusher.Flush()
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fullText.String(), err
	}

	return fullText.String(), nil
}

func UpdateAssistantInstructions(apiKey, assistantID, instructions string) error {
	payload := map[string]any{
		"instructions": instructions,
	}
	return doRequest(apiKey, "POST", "/assistants/"+assistantID, payload, nil)
}

// AddMessageToThread adds a message to an existing thread (used for sub-threads with assistant role)
func AddAssistantMessage(apiKey, threadID, content string) error {
	payload := map[string]any{
		"role":    "assistant",
		"content": content,
	}
	return doRequest(apiKey, "POST", "/threads/"+threadID+"/messages", payload, nil)
}
