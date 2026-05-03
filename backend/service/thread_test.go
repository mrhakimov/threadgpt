package service

import (
	"context"
	"testing"
	"threadgpt/domain"
	"threadgpt/repository"
)

func TestThreadServiceReply_ContinuesAfterClientDisconnect(t *testing.T) {
	sessionRepo := &stubSessionRepository{
		session: &domain.Session{
			ID:           "session-1",
			APIKeyHash:   "hash-1",
			SystemPrompt: stringPtr("Be helpful"),
		},
	}
	conversationRepo := &stubConversationRepository{
		refByConversationID: map[string]*domain.ConversationRef{
			"conv-1": {
				ConversationID: "conv-1",
				SessionID:      "session-1",
			},
		},
	}
	assistant := &stubAssistantClient{
		runAndStreamFunc: func(ctx context.Context, _ string, conversationID, userMessage, sessionID string, stream repository.StreamWriter) error {
			if err := stream.Start(sessionID); err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				t.Fatalf("expected detached context during stream, got %v", err)
			}
			if conversationID != "conv-1" {
				t.Fatalf("unexpected conversation id: %q", conversationID)
			}
			if userMessage != "Follow-up question" {
				t.Fatalf("unexpected user message: %q", userMessage)
			}
			return nil
		},
	}

	service := NewThreadService(sessionRepo, conversationRepo, assistant)

	ctx, cancel := context.WithCancel(context.Background())
	err := service.Reply(ctx, ThreadRequest{
		APIKey:         "sk-test",
		APIKeyHash:     "hash-1",
		ConversationID: "conv-1",
		UserMessage:    "Follow-up question",
	}, &cancelOnStartStreamWriter{cancel: cancel})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestThreadServiceGet_ExcludesInitialExchangeAndPaginatesNewestFirst(t *testing.T) {
	sessionRepo := &stubSessionRepository{
		session: &domain.Session{
			ID:           "session-1",
			APIKeyHash:   "hash-1",
			SystemPrompt: stringPtr("Be helpful"),
		},
	}
	conversationRepo := &stubConversationRepository{
		refByConversationID: map[string]*domain.ConversationRef{
			"conv-1": {
				ConversationID: "conv-1",
				SessionID:      "session-1",
			},
		},
	}
	assistant := &stubAssistantClient{
		listMessages: []domain.Message{
			{ID: "msg-1", Role: "user", Content: "Top-level question"},
			{ID: "msg-2", Role: "assistant", Content: "Top-level answer"},
			{ID: "msg-3", Role: "user", Content: "Follow-up one"},
			{ID: "msg-4", Role: "assistant", Content: "Answer one"},
			{ID: "msg-5", Role: "user", Content: "Follow-up two"},
			{ID: "msg-6", Role: "assistant", Content: "Answer two"},
		},
	}

	service := NewThreadService(sessionRepo, conversationRepo, assistant)

	messages, err := service.Get(context.Background(), "sk-test", "hash-1", "conv-1", 2, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].ID != "msg-5" || messages[1].ID != "msg-6" {
		t.Fatalf("unexpected paginated messages: %+v", messages)
	}
}

type stubSessionRepository struct {
	session             *domain.Session
	systemPromptUpdates int
}

func (s *stubSessionRepository) GetLatestByAPIKeyHash(context.Context, string) (*domain.Session, error) {
	return s.session, nil
}

func (s *stubSessionRepository) GetByID(context.Context, string) (*domain.Session, error) {
	return s.session, nil
}

func (s *stubSessionRepository) ListByAPIKeyHash(context.Context, string, int, int) ([]domain.Session, error) {
	return nil, nil
}

func (s *stubSessionRepository) CreateWithPrompt(_ context.Context, apiKeyHash, systemPrompt string) (*domain.Session, error) {
	if s.session == nil {
		s.session = &domain.Session{ID: "session-1", APIKeyHash: apiKeyHash}
	}
	s.session.SystemPrompt = &systemPrompt
	return s.session, nil
}

func (s *stubSessionRepository) CreateNamed(context.Context, string, string) (*domain.Session, error) {
	return s.session, nil
}

func (s *stubSessionRepository) Rename(context.Context, string, string) error {
	return nil
}

func (s *stubSessionRepository) SetSystemPrompt(_ context.Context, _ string, systemPrompt string) error {
	s.systemPromptUpdates++
	if s.session != nil {
		s.session.SystemPrompt = &systemPrompt
	}
	return nil
}

func (s *stubSessionRepository) UpdateAssistant(context.Context, string, string) error {
	return nil
}

func (s *stubSessionRepository) Delete(context.Context, string) error {
	return nil
}

type stubConversationRepository struct {
	createdConversationID string
	createdSessionID      string
	refByConversationID   map[string]*domain.ConversationRef
	listBySessionAsc      []domain.ConversationRef
}

func (r *stubConversationRepository) Create(_ context.Context, sessionID, conversationID string) (*domain.ConversationRef, error) {
	r.createdSessionID = sessionID
	r.createdConversationID = conversationID
	ref := &domain.ConversationRef{
		ConversationID: conversationID,
		SessionID:      sessionID,
	}
	if r.refByConversationID == nil {
		r.refByConversationID = map[string]*domain.ConversationRef{}
	}
	r.refByConversationID[conversationID] = ref
	return ref, nil
}

func (r *stubConversationRepository) GetByConversationID(_ context.Context, conversationID string) (*domain.ConversationRef, error) {
	if r.refByConversationID == nil {
		return nil, nil
	}
	return r.refByConversationID[conversationID], nil
}

func (r *stubConversationRepository) ListBySessionDesc(context.Context, string, int, int) ([]domain.ConversationRef, error) {
	return nil, nil
}

func (r *stubConversationRepository) ListBySessionAsc(context.Context, string) ([]domain.ConversationRef, error) {
	return append([]domain.ConversationRef(nil), r.listBySessionAsc...), nil
}

type stubAssistantClient struct {
	createConversationID   string
	createConversationFunc func(context.Context, string, string) (string, error)
	runAndStreamFunc       func(context.Context, string, string, string, string, repository.StreamWriter) error
	listMessages           []domain.Message
	deletedConversationIDs []string
}

func (a *stubAssistantClient) CreateConversation(ctx context.Context, apiKey, systemPrompt string) (string, error) {
	if a.createConversationFunc != nil {
		return a.createConversationFunc(ctx, apiKey, systemPrompt)
	}
	if a.createConversationID == "" {
		return "conv-created", nil
	}
	return a.createConversationID, nil
}

func (a *stubAssistantClient) ListMessages(context.Context, string, string) ([]domain.Message, error) {
	return append([]domain.Message(nil), a.listMessages...), nil
}

func (a *stubAssistantClient) RunAndStream(ctx context.Context, apiKey, conversationID, userMessage, sessionID string, stream repository.StreamWriter) error {
	if a.runAndStreamFunc != nil {
		return a.runAndStreamFunc(ctx, apiKey, conversationID, userMessage, sessionID, stream)
	}
	if err := stream.Start(sessionID); err != nil {
		return err
	}
	return nil
}

func (a *stubAssistantClient) DeleteConversation(_ context.Context, _ string, conversationID string) error {
	a.deletedConversationIDs = append(a.deletedConversationIDs, conversationID)
	return nil
}

type stubStreamWriter struct{}

func (s *stubStreamWriter) Start(string) error      { return nil }
func (s *stubStreamWriter) WriteChunk(string) error { return nil }
func (s *stubStreamWriter) WriteError(domain.ErrorDescriptor) error {
	return nil
}
func (s *stubStreamWriter) Close() error { return nil }

type cancelOnStartStreamWriter struct {
	cancel func()
}

func (s *cancelOnStartStreamWriter) Start(string) error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *cancelOnStartStreamWriter) WriteChunk(string) error { return nil }
func (s *cancelOnStartStreamWriter) WriteError(domain.ErrorDescriptor) error {
	return nil
}
func (s *cancelOnStartStreamWriter) Close() error { return nil }

func stringPtr(value string) *string {
	return &value
}
