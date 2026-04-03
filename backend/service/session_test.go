package service

import (
	"context"
	"testing"
	"threadgpt/domain"
)

func TestSessionServiceUpdate_SyncsFirstRootUserMessageExplicitly(t *testing.T) {
	assistantID := "assistant-1"
	sessionRepo := &stubSessionRepository{
		session: &domain.Session{
			ID:           "session-1",
			APIKeyHash:   "hash-1",
			AssistantID:  &assistantID,
			SystemPrompt: stringPtr("Old prompt"),
		},
	}
	messageRepo := &stubMessageRepository{
		firstRootUserMessage: &domain.Message{
			ID:        "message-1",
			SessionID: "session-1",
			Role:      "user",
			Content:   "Old prompt",
		},
	}
	assistant := &stubAssistantClient{}

	service := NewSessionService(sessionRepo, messageRepo, assistant)

	err := service.Update(context.Background(), "sk-test", "hash-1", "session-1", "", "New prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sessionRepo.systemPromptUpdates != 1 {
		t.Fatalf("expected session prompt to be updated once, got %d", sessionRepo.systemPromptUpdates)
	}
	if assistant.updatedInstructions != "New prompt" {
		t.Fatalf("expected assistant instructions update, got %q", assistant.updatedInstructions)
	}
	if messageRepo.updatedMessageID != "message-1" || messageRepo.updatedContent != "New prompt" {
		t.Fatalf("expected first root user message to be updated, got id=%q content=%q", messageRepo.updatedMessageID, messageRepo.updatedContent)
	}
}
