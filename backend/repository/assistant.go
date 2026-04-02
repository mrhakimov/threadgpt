package repository

import "context"

type StreamWriter interface {
	Start(sessionID string) error
	WriteChunk(chunk string) error
	Close() error
}

type AssistantClient interface {
	CreateAssistant(ctx context.Context, apiKey, instructions string) (string, error)
	CreateThread(ctx context.Context, apiKey string) (string, error)
	AddUserMessage(ctx context.Context, apiKey, threadID, content string) error
	AddAssistantMessage(ctx context.Context, apiKey, threadID, content string) error
	RunAndStream(ctx context.Context, apiKey, threadID, assistantID string, stream StreamWriter) (string, error)
	UpdateAssistantInstructions(ctx context.Context, apiKey, assistantID, instructions string) error
}
