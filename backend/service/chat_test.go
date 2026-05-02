package service

import (
	"context"
	"testing"
	"threadgpt/domain"
	"threadgpt/repository"
)

func TestChatServiceHandle_ContinuesAfterClientDisconnect(t *testing.T) {
	sessionRepo := &stubSessionRepository{
		session: &domain.Session{
			ID:           "session-1",
			APIKeyHash:   "hash-1",
			SystemPrompt: stringPtr("Be helpful"),
		},
	}
	conversationRepo := &stubConversationRepository{}
	assistant := &stubAssistantClient{
		createConversationID: "conv-1",
		runAndStreamFunc: func(ctx context.Context, _ string, conversationID, userMessage string, _ repository.StreamWriter) error {
			if err := ctx.Err(); err != nil {
				t.Fatalf("expected detached context during stream, got %v", err)
			}
			if conversationID != "conv-1" {
				t.Fatalf("unexpected conversation id: %q", conversationID)
			}
			if userMessage != "Hello" {
				t.Fatalf("unexpected user message: %q", userMessage)
			}
			return nil
		},
	}

	service := NewChatService(sessionRepo, conversationRepo, assistant)

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

	if conversationRepo.createdConversationID != "conv-1" {
		t.Fatalf("expected created conversation ref, got %q", conversationRepo.createdConversationID)
	}
	if conversationRepo.createdSessionID != "session-1" {
		t.Fatalf("expected session-1, got %q", conversationRepo.createdSessionID)
	}
}

func TestChatServiceHandleInitialMessage_SetsSystemPromptWithoutCreatingConversation(t *testing.T) {
	sessionRepo := &stubSessionRepository{}
	conversationRepo := &stubConversationRepository{}
	assistant := &stubAssistantClient{}
	stream := &recordingChatStreamWriter{}

	service := NewChatService(sessionRepo, conversationRepo, assistant)

	if err := service.Handle(context.Background(), ChatRequest{
		APIKeyHash:  "hash-1",
		UserMessage: "You are concise",
	}, stream); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sessionRepo.session == nil || sessionRepo.session.SystemPrompt == nil || *sessionRepo.session.SystemPrompt != "You are concise" {
		t.Fatalf("expected session prompt to be stored, got %+v", sessionRepo.session)
	}
	if conversationRepo.createdConversationID != "" {
		t.Fatalf("expected no remote conversation to be created for system prompt setup")
	}
	if stream.chunk != initialChatConfirmation {
		t.Fatalf("expected initial confirmation chunk, got %q", stream.chunk)
	}
}

type recordingChatStreamWriter struct {
	chunk string
}

func (s *recordingChatStreamWriter) Start(string) error {
	return nil
}

func (s *recordingChatStreamWriter) WriteChunk(chunk string) error {
	s.chunk += chunk
	return nil
}

func (s *recordingChatStreamWriter) Close() error {
	return nil
}
