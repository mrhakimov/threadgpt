package repository

import (
	"context"
	"threadgpt/domain"
)

type StreamWriter interface {
	Start(sessionID string) error
	WriteChunk(chunk string) error
	WriteError(detail domain.ErrorDescriptor) error
	Close() error
}

type AssistantClient interface {
	CreateConversation(ctx context.Context, apiKey, systemPrompt string) (string, error)
	ListMessages(ctx context.Context, apiKey, conversationID string) ([]domain.Message, error)
	RunAndStream(ctx context.Context, apiKey, conversationID, userMessage, sessionID, model string, stream StreamWriter) error
	DeleteConversation(ctx context.Context, apiKey, conversationID string) error
}
