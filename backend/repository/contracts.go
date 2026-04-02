package repository

import (
	"context"
	"threadgpt/domain"
)

type SessionRepository interface {
	GetLatestByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.Session, error)
	GetByID(ctx context.Context, sessionID string) (*domain.Session, error)
	ListByAPIKeyHash(ctx context.Context, apiKeyHash string, limit, offset int) ([]domain.Session, error)
	CreateWithPrompt(ctx context.Context, apiKeyHash, systemPrompt string) (*domain.Session, error)
	CreateNamed(ctx context.Context, apiKeyHash, name string) (*domain.Session, error)
	Rename(ctx context.Context, sessionID, name string) error
	SetSystemPrompt(ctx context.Context, sessionID, systemPrompt string) error
	UpdateAssistant(ctx context.Context, sessionID, assistantID string) error
	Delete(ctx context.Context, sessionID string) error
}

type MessageRepository interface {
	Save(ctx context.Context, sessionID, role, content string, openAIThreadID, parentMessageID *string) (*domain.Message, error)
	GetMessageByID(ctx context.Context, messageID string) (*domain.Message, error)
	GetMainAsc(ctx context.Context, sessionID string, limit, offset int) ([]domain.Message, error)
	GetMainDesc(ctx context.Context, sessionID string, limit, offset int) ([]domain.Message, error)
	GetThreadAsc(ctx context.Context, parentMessageID string, limit, offset int) ([]domain.Message, error)
	GetThreadDesc(ctx context.Context, parentMessageID string, limit, offset int) ([]domain.Message, error)
	FindFirstRootUserMessage(ctx context.Context, sessionID string) (*domain.Message, error)
	UpdateContent(ctx context.Context, messageID, content string) error
}
