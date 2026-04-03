package data

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOpenAIClient_RunAndStreamHandlesLargeChunks(t *testing.T) {
	restore := mockOpenAITransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/threads/thread-1/runs" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}

		chunk := strings.Repeat("a", 70*1024)
		payload, err := json.Marshal(map[string]any{
			"object": "thread.message.delta",
			"delta": map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": map[string]any{"value": chunk},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("failed to build payload: %v", err)
		}

		body := "data: " + string(payload) + "\n\n" + "data: [DONE]\n\n"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
	defer restore()

	client := NewOpenAIClient()
	stream := &recordingStreamWriter{}

	text, err := client.RunAndStream(context.Background(), "sk-test", "thread-1", "assistant-1", stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != strings.Repeat("a", 70*1024) {
		t.Fatalf("expected full chunk to be returned, got %d bytes", len(text))
	}
	if stream.output.String() != text {
		t.Fatalf("expected stream output to match returned text")
	}
}

type recordingStreamWriter struct {
	output strings.Builder
}

func (w *recordingStreamWriter) Start(string) error {
	return nil
}

func (w *recordingStreamWriter) WriteChunk(chunk string) error {
	w.output.WriteString(chunk)
	return nil
}

func (w *recordingStreamWriter) Close() error {
	return nil
}

func mockOpenAITransport(fn func(*http.Request) (*http.Response, error)) func() {
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "api.openai.com" {
			return fn(req)
		}
		return original.RoundTrip(req)
	})
	return func() {
		http.DefaultTransport = original
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
