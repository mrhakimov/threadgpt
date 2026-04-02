package service

import (
	"context"
	"threadgpt/domain"
	"threadgpt/repository"
)

const initialChatConfirmation = "Context set! Your assistant has been configured with this as its instructions. Send your next message to start chatting."

type ChatService struct {
	sessions  repository.SessionRepository
	messages  repository.MessageRepository
	assistant repository.AssistantClient
}

type ChatRequest struct {
	APIKey      string
	APIKeyHash  string
	UserMessage string
	SessionID   string
	ForceNew    bool
}

func NewChatService(sessions repository.SessionRepository, messages repository.MessageRepository, assistant repository.AssistantClient) *ChatService {
	return &ChatService{
		sessions:  sessions,
		messages:  messages,
		assistant: assistant,
	}
}

func (s *ChatService) Handle(ctx context.Context, req ChatRequest, stream repository.StreamWriter) error {
	session, err := s.resolveSession(ctx, req.APIKeyHash, req.SessionID, req.ForceNew)
	if err != nil {
		return err
	}

	if req.SessionID != "" {
		if session == nil {
			return domain.ErrNotFound
		}
		if session.APIKeyHash != req.APIKeyHash {
			return domain.ErrForbidden
		}
	}

	if session == nil || session.AssistantID == nil {
		return s.handleInitialMessage(ctx, req, session, stream)
	}

	threadID, err := s.assistant.CreateThread(ctx, req.APIKey)
	if err != nil {
		return err
	}
	if err := s.assistant.AddUserMessage(ctx, req.APIKey, threadID, req.UserMessage); err != nil {
		return err
	}
	if _, err := s.messages.Save(ctx, session.ID, "user", req.UserMessage, &threadID, nil); err != nil {
		return err
	}
	if err := stream.Start(session.ID); err != nil {
		return err
	}

	assistantText, err := s.assistant.RunAndStream(ctx, req.APIKey, threadID, *session.AssistantID, stream)
	if assistantText != "" {
		if _, saveErr := s.messages.Save(ctx, session.ID, "assistant", assistantText, &threadID, nil); saveErr != nil {
			return saveErr
		}
	}
	if err != nil {
		return err
	}

	return stream.Close()
}

func (s *ChatService) resolveSession(ctx context.Context, apiKeyHash, sessionID string, forceNew bool) (*domain.Session, error) {
	if sessionID != "" {
		return s.sessions.GetByID(ctx, sessionID)
	}
	if forceNew {
		return nil, nil
	}
	return s.sessions.GetLatestByAPIKeyHash(ctx, apiKeyHash)
}

func (s *ChatService) handleInitialMessage(ctx context.Context, req ChatRequest, session *domain.Session, stream repository.StreamWriter) error {
	assistantID, err := s.assistant.CreateAssistant(ctx, req.APIKey, req.UserMessage)
	if err != nil {
		return err
	}

	if session == nil {
		session, err = s.sessions.CreateWithPrompt(ctx, req.APIKeyHash, req.UserMessage)
		if err != nil {
			return err
		}
	} else {
		if err := s.sessions.SetSystemPrompt(ctx, session.ID, req.UserMessage); err != nil {
			return err
		}
	}

	if err := s.sessions.UpdateAssistant(ctx, session.ID, assistantID); err != nil {
		return err
	}
	session.AssistantID = &assistantID

	if _, err := s.messages.Save(ctx, session.ID, "user", req.UserMessage, nil, nil); err != nil {
		return err
	}

	if err := stream.Start(session.ID); err != nil {
		return err
	}
	if err := stream.WriteChunk(initialChatConfirmation); err != nil {
		return err
	}
	if _, err := s.messages.Save(ctx, session.ID, "assistant", initialChatConfirmation, nil, nil); err != nil {
		return err
	}
	return stream.Close()
}
