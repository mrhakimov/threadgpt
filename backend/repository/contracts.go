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

type ConversationRepository interface {
	Create(ctx context.Context, sessionID, conversationID string) (*domain.ConversationRef, error)
	GetByConversationID(ctx context.Context, conversationID string) (*domain.ConversationRef, error)
	ListBySessionDesc(ctx context.Context, sessionID string, limit, offset int) ([]domain.ConversationRef, error)
	ListBySessionAsc(ctx context.Context, sessionID string) ([]domain.ConversationRef, error)
}
