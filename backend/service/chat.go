package service

import (
	"context"
	"threadgpt/domain"
	"threadgpt/repository"
)

const initialChatConfirmation = "Context set! Your assistant has been configured with this as its instructions. Send your next message to start chatting."

type ChatService struct {
	sessions      repository.SessionRepository
	conversations repository.ConversationRepository
	assistant     repository.AssistantClient
}

type ChatRequest struct {
	APIKey      string
	APIKeyHash  string
	UserMessage string
	SessionID   string
	ForceNew    bool
}

func NewChatService(sessions repository.SessionRepository, conversations repository.ConversationRepository, assistant repository.AssistantClient) *ChatService {
	return &ChatService{
		sessions:      sessions,
		conversations: conversations,
		assistant:     assistant,
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

	if session == nil || session.SystemPrompt == nil {
		return s.handleInitialMessage(ctx, req, session, stream)
	}

	// Keep the remote conversation running long enough to survive an SSE disconnect.
	opCtx := context.WithoutCancel(ctx)

	conversationID, err := s.assistant.CreateConversation(opCtx, req.APIKey, *session.SystemPrompt)
	if err != nil {
		return err
	}
	if _, err := s.conversations.Create(opCtx, session.ID, conversationID); err != nil {
		return err
	}

	if err := s.assistant.RunAndStream(opCtx, req.APIKey, conversationID, req.UserMessage, session.ID, stream); err != nil {
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
	var err error
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

	if err := stream.Start(session.ID); err != nil {
		return err
	}
	if err := stream.WriteChunk(initialChatConfirmation); err != nil {
		return err
	}
	return stream.Close()
}
