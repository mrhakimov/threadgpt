package service

import (
	"context"
	"threadgpt/domain"
	"threadgpt/repository"
)

type SessionService struct {
	sessions  repository.SessionRepository
	assistant repository.AssistantClient
}

func NewSessionService(sessions repository.SessionRepository, assistant repository.AssistantClient) *SessionService {
	return &SessionService{
		sessions:  sessions,
		assistant: assistant,
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

func (s *SessionService) Update(ctx context.Context, apiKey, apiKeyHash, sessionID, name, systemPrompt string) error {
	session, err := s.GetByID(ctx, apiKeyHash, sessionID)
	if err != nil {
		return err
	}

	if name != "" {
		if err := s.sessions.Rename(ctx, sessionID, name); err != nil {
			return err
		}
	}

	if systemPrompt != "" {
		if session.AssistantID != nil {
			if err := s.assistant.UpdateAssistantInstructions(ctx, apiKey, *session.AssistantID, systemPrompt); err != nil {
				return err
			}
		}
		if err := s.sessions.SetSystemPrompt(ctx, sessionID, systemPrompt); err != nil {
			return err
		}
	}

	return nil
}

func (s *SessionService) Delete(ctx context.Context, apiKeyHash, sessionID string) error {
	if _, err := s.GetByID(ctx, apiKeyHash, sessionID); err != nil {
		return err
	}
	return s.sessions.Delete(ctx, sessionID)
}
