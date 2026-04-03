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

	// Closing a subthread drawer aborts the client request, but we still want
	// the branch reply to finish and be saved so later refreshes stay consistent.
	opCtx := context.WithoutCancel(ctx)

	threadID, err := s.messages.GetBranchThreadID(opCtx, req.ParentMessageID)
	if err != nil {
		return err
	}

	resolvedThreadID, err := s.resolveThread(opCtx, req.APIKey, parentMessage, threadID)
	if err != nil {
		return err
	}

	if err := s.assistant.AddUserMessage(opCtx, req.APIKey, resolvedThreadID, req.UserMessage); err != nil {
		return err
	}

	parentID := req.ParentMessageID
	if _, err := s.messages.Save(opCtx, session.ID, "user", req.UserMessage, &resolvedThreadID, &parentID); err != nil {
		return err
	}

	if err := stream.Start(""); err != nil {
		return err
	}

	assistantText, err := s.assistant.RunAndStream(opCtx, req.APIKey, resolvedThreadID, *session.AssistantID, stream)
	if assistantText != "" {
		if _, saveErr := s.messages.Save(opCtx, session.ID, "assistant", assistantText, &resolvedThreadID, &parentID); saveErr != nil {
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

func (s *ThreadService) resolveThread(ctx context.Context, apiKey string, parentMessage *domain.Message, existingThreadID *string) (string, error) {
	if existingThreadID != nil {
		return *existingThreadID, nil
	}

	threadID, err := s.assistant.CreateThread(ctx, apiKey)
	if err != nil {
		return "", err
	}

	messages, err := s.loadBranchContext(ctx, parentMessage)
	if err != nil {
		return "", err
	}
	for _, message := range messages {
		if err := s.replayMessage(ctx, apiKey, threadID, message); err != nil {
			return "", err
		}
	}

	return threadID, nil
}

func (s *ThreadService) loadBranchContext(ctx context.Context, message *domain.Message) ([]domain.Message, error) {
	var messages []domain.Message
	current := message

	for current != nil {
		messages = append(messages, *current)
		if current.ParentMessageID == nil {
			break
		}

		next, err := s.messages.GetMessageByID(ctx, *current.ParentMessageID)
		if err != nil {
			return nil, err
		}
		if next == nil {
			return nil, domain.ErrInternal
		}
		current = next
	}

	reverseMessages(messages)
	return messages, nil
}

func (s *ThreadService) replayMessage(ctx context.Context, apiKey, threadID string, message domain.Message) error {
	switch message.Role {
	case "assistant":
		return s.assistant.AddAssistantMessage(ctx, apiKey, threadID, message.Content)
	case "user":
		return s.assistant.AddUserMessage(ctx, apiKey, threadID, message.Content)
	default:
		return domain.ErrInternal
	}
}

func reverseMessages(messages []domain.Message) {
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
}
