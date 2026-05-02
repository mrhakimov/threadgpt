package service

import (
	"context"
	"testing"
	"threadgpt/domain"
)

func TestSessionServiceUpdate_OnlyUpdatesSessionPrompt(t *testing.T) {
	sessionRepo := &stubSessionRepository{
		session: &domain.Session{
			ID:           "session-1",
			APIKeyHash:   "hash-1",
			SystemPrompt: stringPtr("Old prompt"),
		},
	}
	conversationRepo := &stubConversationRepository{}
	assistant := &stubAssistantClient{}

	service := NewSessionService(sessionRepo, conversationRepo, assistant)

	err := service.Update(context.Background(), "sk-test", "hash-1", "session-1", "", "New prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sessionRepo.systemPromptUpdates != 1 {
		t.Fatalf("expected session prompt to be updated once, got %d", sessionRepo.systemPromptUpdates)
	}
	if sessionRepo.session.SystemPrompt == nil || *sessionRepo.session.SystemPrompt != "New prompt" {
		t.Fatalf("expected updated prompt, got %+v", sessionRepo.session.SystemPrompt)
	}
	if len(assistant.deletedConversationIDs) != 0 {
		t.Fatalf("expected no conversation deletions during update")
	}
}

func TestSessionServiceDelete_RemovesRemoteConversationsBeforeDeletingSession(t *testing.T) {
	sessionRepo := &stubSessionRepository{
		session: &domain.Session{
			ID:         "session-1",
			APIKeyHash: "hash-1",
		},
	}
	conversationRepo := &stubConversationRepository{
		listBySessionAsc: []domain.ConversationRef{
			{ConversationID: "conv-1", SessionID: "session-1"},
			{ConversationID: "conv-2", SessionID: "session-1"},
		},
	}
	assistant := &stubAssistantClient{}

	service := NewSessionService(sessionRepo, conversationRepo, assistant)

	if err := service.Delete(context.Background(), "sk-test", "hash-1", "session-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(assistant.deletedConversationIDs) != 2 {
		t.Fatalf("expected 2 deleted conversations, got %d", len(assistant.deletedConversationIDs))
	}
}
