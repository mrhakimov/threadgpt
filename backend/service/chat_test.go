package service

import (
	"context"
	"testing"
	"threadgpt/domain"
	"threadgpt/repository"
)

func TestChatServiceHandle_ContinuesAfterClientDisconnect(t *testing.T) {
	assistantID := "assistant-1"
	sessionRepo := &stubSessionRepository{
		session: &domain.Session{
			ID:          "session-1",
			APIKeyHash:  "hash-1",
			AssistantID: &assistantID,
		},
	}
	messageRepo := &stubMessageRepository{}
	assistant := &stubAssistantClient{
		createThreadID: "thread-1",
		runAndStreamFunc: func(ctx context.Context, _ string, threadID, assistantID string, _ repository.StreamWriter) (string, error) {
			if err := ctx.Err(); err != nil {
				t.Fatalf("expected detached context during stream, got %v", err)
			}
			if threadID != "thread-1" {
				t.Fatalf("unexpected thread id: %q", threadID)
			}
			if assistantID != "assistant-1" {
				t.Fatalf("unexpected assistant id: %q", assistantID)
			}
			return "Saved after disconnect", nil
		},
	}

	service := NewChatService(sessionRepo, messageRepo, assistant)

	ctx, cancel := context.WithCancel(context.Background())
	err := service.Handle(ctx, ChatRequest{
		APIKey:      "sk-test",
		APIKeyHash:  "hash-1",
		UserMessage: "Hello",
		SessionID:   "session-1",
	}, &cancelOnStartStreamWriter{cancel: cancel})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messageRepo.savedMessages) != 2 {
		t.Fatalf("expected user and assistant messages to be saved, got %d", len(messageRepo.savedMessages))
	}
	if messageRepo.savedMessages[0].Role != "user" || messageRepo.savedMessages[0].Content != "Hello" {
		t.Fatalf("unexpected saved user message: %+v", messageRepo.savedMessages[0])
	}
	if messageRepo.savedMessages[1].Role != "assistant" || messageRepo.savedMessages[1].Content != "Saved after disconnect" {
		t.Fatalf("unexpected saved assistant message: %+v", messageRepo.savedMessages[1])
	}
}
