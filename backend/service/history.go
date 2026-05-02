package service

import (
	"context"
	"threadgpt/domain"
	"threadgpt/repository"
)

type HistoryService struct {
	sessions      repository.SessionRepository
	conversations repository.ConversationRepository
	assistant     repository.AssistantClient
}

func NewHistoryService(sessions repository.SessionRepository, conversations repository.ConversationRepository, assistant repository.AssistantClient) *HistoryService {
	return &HistoryService{
		sessions:      sessions,
		conversations: conversations,
		assistant:     assistant,
	}
}

func (s *HistoryService) Get(ctx context.Context, apiKey, apiKeyHash, sessionID string, limit, offset int) ([]domain.ConversationPreview, error) {
	session, err := s.resolveSession(ctx, apiKeyHash, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return []domain.ConversationPreview{}, nil
	}
	if session.APIKeyHash != apiKeyHash {
		return nil, domain.ErrForbidden
	}

	refs, err := s.conversations.ListBySessionDesc(ctx, session.ID, limit, offset)
	if err != nil {
		return nil, err
	}

	previews := make([]domain.ConversationPreview, 0, len(refs))
	for _, ref := range refs {
		messages, err := s.assistant.ListMessages(ctx, apiKey, ref.ConversationID)
		if err != nil {
			return nil, err
		}
		previews = append(previews, buildConversationPreview(session.ID, ref, messages))
	}

	return previews, nil
}

func (s *HistoryService) resolveSession(ctx context.Context, apiKeyHash, sessionID string) (*domain.Session, error) {
	if sessionID != "" {
		session, err := s.sessions.GetByID(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, domain.ErrNotFound
		}
		return session, nil
	}

	return s.sessions.GetLatestByAPIKeyHash(ctx, apiKeyHash)
}

func buildConversationPreview(sessionID string, ref domain.ConversationRef, messages []domain.Message) domain.ConversationPreview {
	var userMessage string
	var assistantMessage string
	replyCount := 0
	seenAssistant := false

	for _, message := range messages {
		switch message.Role {
		case "user":
			if userMessage == "" {
				userMessage = message.Content
				continue
			}
			if seenAssistant {
				replyCount++
			}
		case "assistant":
			if assistantMessage == "" {
				assistantMessage = message.Content
			}
			seenAssistant = true
		}
	}

	return domain.ConversationPreview{
		ConversationID:   ref.ConversationID,
		SessionID:        sessionID,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		ReplyCount:       replyCount,
		CreatedAt:        ref.CreatedAt,
	}
}
