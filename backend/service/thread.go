package service

import (
	"context"
	"threadgpt/domain"
	"threadgpt/repository"
)

type ThreadService struct {
	sessions      repository.SessionRepository
	conversations repository.ConversationRepository
	assistant     repository.AssistantClient
}

type ThreadRequest struct {
	APIKey         string
	APIKeyHash     string
	ConversationID string
	UserMessage    string
	Model          string
}

func NewThreadService(sessions repository.SessionRepository, conversations repository.ConversationRepository, assistant repository.AssistantClient) *ThreadService {
	return &ThreadService{
		sessions:      sessions,
		conversations: conversations,
		assistant:     assistant,
	}
}

func (s *ThreadService) Reply(ctx context.Context, req ThreadRequest, stream repository.StreamWriter) error {
	ref, session, err := s.resolveConversation(ctx, req.APIKeyHash, req.ConversationID)
	if err != nil {
		return err
	}
	if ref == nil || session == nil {
		return domain.ErrNotFound
	}

	opCtx := context.WithoutCancel(ctx)
	if err := s.assistant.RunAndStream(opCtx, req.APIKey, ref.ConversationID, req.UserMessage, "", req.Model, stream); err != nil {
		return err
	}
	return stream.Close()
}

func (s *ThreadService) Get(ctx context.Context, apiKey, apiKeyHash, conversationID string, limit, offset int) ([]domain.Message, error) {
	ref, _, err := s.resolveConversation(ctx, apiKeyHash, conversationID)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, domain.ErrNotFound
	}

	messages, err := s.assistant.ListMessages(ctx, apiKey, ref.ConversationID)
	if err != nil {
		return nil, err
	}
	paginated := paginateNewestAscending(messages, limit, offset)
	for i := range paginated {
		paginated[i].SessionID = ref.SessionID
	}

	return paginated, nil
}

func (s *ThreadService) resolveConversation(ctx context.Context, apiKeyHash, conversationID string) (*domain.ConversationRef, *domain.Session, error) {
	ref, err := s.conversations.GetByConversationID(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}
	if ref == nil {
		return nil, nil, domain.ErrNotFound
	}

	session, err := s.sessions.GetByID(ctx, ref.SessionID)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, domain.ErrNotFound
	}
	if session.APIKeyHash != apiKeyHash {
		return nil, nil, domain.ErrForbidden
	}

	return ref, session, nil
}

func paginateNewestAscending(messages []domain.Message, limit, offset int) []domain.Message {
	if len(messages) == 0 || limit <= 0 || offset >= len(messages) {
		return []domain.Message{}
	}

	end := len(messages) - offset
	if end < 0 {
		return []domain.Message{}
	}

	start := end - limit
	if start < 0 {
		start = 0
	}

	items := append([]domain.Message(nil), messages[start:end]...)
	return items
}
