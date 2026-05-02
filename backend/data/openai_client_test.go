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
		if req.Method != http.MethodPost || req.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}

		chunk := strings.Repeat("a", 70*1024)
		payload, err := json.Marshal(map[string]any{
			"type":  "response.output_text.delta",
			"delta": chunk,
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

	if err := client.RunAndStream(context.Background(), "sk-test", "conv-1", "Hello", stream); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream.output.String() != strings.Repeat("a", 70*1024) {
		t.Fatalf("expected full chunk to be streamed, got %d bytes", len(stream.output.String()))
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
