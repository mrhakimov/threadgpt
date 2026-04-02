package service

import (
	"context"
	"threadgpt/domain"
	"threadgpt/repository"
)

type HistoryService struct {
	sessions repository.SessionRepository
	messages repository.MessageRepository
}

func NewHistoryService(sessions repository.SessionRepository, messages repository.MessageRepository) *HistoryService {
	return &HistoryService{
		sessions: sessions,
		messages: messages,
	}
}

func (s *HistoryService) Get(ctx context.Context, apiKeyHash, sessionID string, limit, offset int) ([]domain.Message, error) {
	if sessionID != "" {
		session, err := s.sessions.GetByID(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if session == nil || session.APIKeyHash != apiKeyHash {
			return nil, domain.ErrForbidden
		}
		return s.messages.GetMainDesc(ctx, sessionID, limit, offset)
	}

	session, err := s.sessions.GetLatestByAPIKeyHash(ctx, apiKeyHash)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return []domain.Message{}, nil
	}

	return s.messages.GetMainDesc(ctx, session.ID, limit, offset)
}
