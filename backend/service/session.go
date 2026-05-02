package service

import (
	"context"
	"threadgpt/domain"
	"threadgpt/repository"
)

type SessionService struct {
	sessions      repository.SessionRepository
	conversations repository.ConversationRepository
	assistant     repository.AssistantClient
}

func NewSessionService(sessions repository.SessionRepository, conversations repository.ConversationRepository, assistant repository.AssistantClient) *SessionService {
	return &SessionService{
		sessions:      sessions,
		conversations: conversations,
		assistant:     assistant,
	}
}

func (s *SessionService) GetCurrent(ctx context.Context, apiKeyHash string) (*domain.Session, error) {
	return s.sessions.GetLatestByAPIKeyHash(ctx, apiKeyHash)
}

func (s *SessionService) List(ctx context.Context, apiKeyHash string, limit, offset int) ([]domain.Session, error) {
	return s.sessions.ListByAPIKeyHash(ctx, apiKeyHash, limit, offset)
}

func (s *SessionService) CreateNamed(ctx context.Context, apiKeyHash, name string) (*domain.Session, error) {
	return s.sessions.CreateNamed(ctx, apiKeyHash, name)
}

func (s *SessionService) GetByID(ctx context.Context, apiKeyHash, sessionID string) (*domain.Session, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, domain.ErrNotFound
	}
	if session.APIKeyHash != apiKeyHash {
		return nil, domain.ErrForbidden
	}
	return session, nil
}

func (s *SessionService) Update(ctx context.Context, _ string, apiKeyHash, sessionID, name, systemPrompt string) error {
	if _, err := s.GetByID(ctx, apiKeyHash, sessionID); err != nil {
		return err
	}

	if name != "" {
		if err := s.sessions.Rename(ctx, sessionID, name); err != nil {
			return err
		}
	}

	if systemPrompt != "" {
		if err := s.sessions.SetSystemPrompt(ctx, sessionID, systemPrompt); err != nil {
			return err
		}
	}

	return nil
}

func (s *SessionService) Delete(ctx context.Context, apiKey, apiKeyHash, sessionID string) error {
	if _, err := s.GetByID(ctx, apiKeyHash, sessionID); err != nil {
		return err
	}

	refs, err := s.conversations.ListBySessionAsc(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		if err := s.assistant.DeleteConversation(ctx, apiKey, ref.ConversationID); err != nil {
			return err
		}
	}

	return s.sessions.Delete(ctx, sessionID)
}
