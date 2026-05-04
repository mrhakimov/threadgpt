package data

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"threadgpt/domain"
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

	if err := client.RunAndStream(context.Background(), "sk-test", "conv-1", "Hello", "session-1", "", stream); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stream.started {
		t.Fatalf("expected stream to start")
	}
	if stream.output.String() != strings.Repeat("a", 70*1024) {
		t.Fatalf("expected full chunk to be streamed, got %d bytes", len(stream.output.String()))
	}
}

func TestOpenAIClient_ListModelsReturnsPreferredTextModels(t *testing.T) {
	restore := mockOpenAITransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/models" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}

		return jsonHTTPResponse(http.StatusOK, map[string]any{
			"data": []map[string]string{
				{"id": "tts-1"},
				{"id": "gpt-4o"},
				{"id": "gpt-5.4-mini"},
				{"id": "gpt-image-2"},
				{"id": "gpt-5.5-2026-04-20"},
				{"id": "gpt-5.5"},
				{"id": "text-embedding-3-small"},
				{"id": "gpt-5.4-nano"},
				{"id": "omni-moderation-latest"},
			},
		}), nil
	})
	defer restore()

	client := NewOpenAIClient()
	models, err := client.ListModels(context.Background(), "sk-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"gpt-5.5", "gpt-5.4-mini", "gpt-5.4-nano"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("expected %v, got %v", want, models)
	}
}

func TestOpenAIClient_RunAndStreamUsesCurrentDefaultModel(t *testing.T) {
	restore := mockOpenAITransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}

		var payload struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request payload: %v", err)
		}
		if payload.Model != "gpt-5.5" {
			t.Fatalf("expected default model gpt-5.5, got %q", payload.Model)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader("data: [DONE]\n\n")),
		}, nil
	})
	defer restore()

	client := NewOpenAIClient()
	stream := &recordingStreamWriter{}

	if err := client.RunAndStream(context.Background(), "sk-test", "conv-1", "Hello", "session-1", "", stream); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIClient_ValidateAPIKeyMapsProviderErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    error
	}{
		{
			name:       "invalid api key",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"code":"invalid_api_key","message":"Incorrect API key provided"}}`,
			wantErr:    domain.ErrInvalidAPIKey,
		},
		{
			name:       "quota exceeded",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":{"code":"insufficient_quota","message":"You exceeded your current quota"}}`,
			wantErr:    domain.ErrQuotaExceeded,
		},
		{
			name:       "provider unavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       `{"error":{"message":"server busy"}}`,
			wantErr:    domain.ErrProviderUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := mockOpenAITransport(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.Path != "/v1/models" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			})
			defer restore()

			client := NewOpenAIClient()
			err := client.ValidateAPIKey(context.Background(), "sk-test")
			if err != tt.wantErr {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestOpenAIClient_RunAndStreamDoesNotStartStreamOnProviderError(t *testing.T) {
	restore := mockOpenAITransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"error":{"code":"insufficient_quota","message":"You exceeded your current quota"}}`,
			)),
		}, nil
	})
	defer restore()

	client := NewOpenAIClient()
	stream := &recordingStreamWriter{}

	err := client.RunAndStream(context.Background(), "sk-test", "conv-1", "Hello", "session-1", "", stream)
	if err != domain.ErrQuotaExceeded {
		t.Fatalf("expected quota exceeded, got %v", err)
	}
	if stream.started {
		t.Fatalf("expected stream to stay unopened")
	}
}

type recordingStreamWriter struct {
	output  strings.Builder
	started bool
	errors  []domain.ErrorDescriptor
}

func (w *recordingStreamWriter) Start(string) error {
	w.started = true
	return nil
}

func (w *recordingStreamWriter) WriteChunk(chunk string) error {
	w.output.WriteString(chunk)
	return nil
}

func (w *recordingStreamWriter) WriteError(detail domain.ErrorDescriptor) error {
	w.errors = append(w.errors, detail)
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

func jsonHTTPResponse(status int, body any) *http.Response {
	payload, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(payload))),
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
