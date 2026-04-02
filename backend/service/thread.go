package service

import (
	"context"
	"threadgpt/domain"
	"threadgpt/repository"
)

type ThreadService struct {
	sessions  repository.SessionRepository
	messages  repository.MessageRepository
	assistant repository.AssistantClient
}

type ThreadRequest struct {
	APIKey          string
	APIKeyHash      string
	ParentMessageID string
	UserMessage     string
}

func NewThreadService(sessions repository.SessionRepository, messages repository.MessageRepository, assistant repository.AssistantClient) *ThreadService {
	return &ThreadService{
		sessions:  sessions,
		messages:  messages,
		assistant: assistant,
	}
}

func (s *ThreadService) Reply(ctx context.Context, req ThreadRequest, stream repository.StreamWriter) error {
	parentMessage, err := s.messages.GetMessageByID(ctx, req.ParentMessageID)
	if err != nil {
		return err
	}
	if parentMessage == nil {
		return domain.ErrNotFound
	}

	session, err := s.sessions.GetByID(ctx, parentMessage.SessionID)
	if err != nil {
		return err
	}
	if session == nil || session.AssistantID == nil {
		return domain.ErrNotFound
	}
	if session.APIKeyHash != req.APIKeyHash {
		return domain.ErrForbidden
	}

	existing, err := s.messages.GetThreadAsc(ctx, req.ParentMessageID, 1000, 0)
	if err != nil {
		return err
	}

	threadID, err := s.resolveThread(ctx, req, parentMessage, existing)
	if err != nil {
		return err
	}

	if err := s.assistant.AddUserMessage(ctx, req.APIKey, threadID, req.UserMessage); err != nil {
		return err
	}

	parentID := req.ParentMessageID
	if _, err := s.messages.Save(ctx, session.ID, "user", req.UserMessage, &threadID, &parentID); err != nil {
		return err
	}

	if err := stream.Start(""); err != nil {
		return err
	}

	assistantText, err := s.assistant.RunAndStream(ctx, req.APIKey, threadID, *session.AssistantID, stream)
	if assistantText != "" {
		if _, saveErr := s.messages.Save(ctx, session.ID, "assistant", assistantText, &threadID, &parentID); saveErr != nil {
			return saveErr
		}
	}
	if err != nil {
		return err
	}

	return stream.Close()
}

func (s *ThreadService) Get(ctx context.Context, apiKeyHash, parentMessageID string, limit, offset int) ([]domain.Message, error) {
	parentMessage, err := s.messages.GetMessageByID(ctx, parentMessageID)
	if err != nil {
		return nil, err
	}
	if parentMessage == nil {
		return nil, domain.ErrNotFound
	}

	session, err := s.sessions.GetByID(ctx, parentMessage.SessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, domain.ErrNotFound
	}
	if session.APIKeyHash != apiKeyHash {
		return nil, domain.ErrForbidden
	}

	return s.messages.GetThreadDesc(ctx, parentMessageID, limit, offset)
}

func (s *ThreadService) resolveThread(ctx context.Context, req ThreadRequest, parentMessage *domain.Message, existing []domain.Message) (string, error) {
	if len(existing) == 0 {
		threadID, err := s.assistant.CreateThread(ctx, req.APIKey)
		if err != nil {
			return "", err
		}
		if parentMessage.Role == "assistant" {
			if err := s.assistant.AddAssistantMessage(ctx, req.APIKey, threadID, parentMessage.Content); err != nil {
				return "", err
			}
		}
		return threadID, nil
	}

	if existing[0].OpenAIThreadID == nil {
		return "", domain.ErrInternal
	}
	return *existing[0].OpenAIThreadID, nil
}
