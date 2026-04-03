package service

import (
	"context"
	"testing"
	"threadgpt/domain"
	"threadgpt/repository"
)

func TestThreadServiceReply_NewBranchReplaysAncestorPath(t *testing.T) {
	assistantID := "assistant-1"
	rootID := "root-user"
	parentID := "parent-assistant"

	sessionRepo := &stubSessionRepository{
		session: &domain.Session{
			ID:          "session-1",
			APIKeyHash:  "hash-1",
			AssistantID: &assistantID,
		},
	}
	messageRepo := &stubMessageRepository{
		messagesByID: map[string]*domain.Message{
			rootID: {
				ID:        rootID,
				SessionID: "session-1",
				Role:      "user",
				Content:   "Original question",
			},
			parentID: {
				ID:              parentID,
				SessionID:       "session-1",
				Role:            "assistant",
				Content:         "Original answer",
				ParentMessageID: &rootID,
			},
		},
	}
	assistant := &stubAssistantClient{
		createThreadID:   "branch-thread-1",
		runAndStreamText: "Follow-up answer",
	}

	service := NewThreadService(sessionRepo, messageRepo, assistant)

	err := service.Reply(context.Background(), ThreadRequest{
		APIKey:          "sk-test",
		APIKeyHash:      "hash-1",
		ParentMessageID: parentID,
		UserMessage:     "Follow-up question",
	}, &stubStreamWriter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []assistantMessageCall{
		{role: "user", threadID: "branch-thread-1", content: "Original question"},
		{role: "assistant", threadID: "branch-thread-1", content: "Original answer"},
		{role: "user", threadID: "branch-thread-1", content: "Follow-up question"},
	}
	if len(assistant.addedMessages) != len(want) {
		t.Fatalf("expected %d replayed messages, got %d", len(want), len(assistant.addedMessages))
	}
	for i := range want {
		if assistant.addedMessages[i] != want[i] {
			t.Fatalf("message %d mismatch: got %+v want %+v", i, assistant.addedMessages[i], want[i])
		}
	}
}

func TestThreadServiceReply_UsesStoredBranchThreadID(t *testing.T) {
	assistantID := "assistant-1"
	parentID := "parent-1"

	sessionRepo := &stubSessionRepository{
		session: &domain.Session{
			ID:          "session-1",
			APIKeyHash:  "hash-1",
			AssistantID: &assistantID,
		},
	}
	messageRepo := &stubMessageRepository{
		messagesByID: map[string]*domain.Message{
			parentID: {
				ID:        parentID,
				SessionID: "session-1",
				Role:      "assistant",
				Content:   "Existing reply parent",
			},
		},
		branchThreadID: stringPtr("branch-thread-existing"),
	}
	assistant := &stubAssistantClient{
		runAndStreamText: "Another reply",
	}

	service := NewThreadService(sessionRepo, messageRepo, assistant)

	err := service.Reply(context.Background(), ThreadRequest{
		APIKey:          "sk-test",
		APIKeyHash:      "hash-1",
		ParentMessageID: parentID,
		UserMessage:     "Continue branch",
	}, &stubStreamWriter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if messageRepo.branchThreadLookupParentID != parentID {
		t.Fatalf("expected explicit branch thread lookup for %q, got %q", parentID, messageRepo.branchThreadLookupParentID)
	}
	if assistant.createThreadCalls != 0 {
		t.Fatalf("expected existing branch thread to be reused")
	}
	if len(assistant.addedMessages) == 0 || assistant.addedMessages[0].threadID != "branch-thread-existing" {
		t.Fatalf("expected branch thread to be reused, got %+v", assistant.addedMessages)
	}
}

func TestThreadServiceReply_ContinuesAfterClientDisconnect(t *testing.T) {
	assistantID := "assistant-1"
	rootID := "root-user"
	parentID := "parent-assistant"

	sessionRepo := &stubSessionRepository{
		session: &domain.Session{
			ID:          "session-1",
			APIKeyHash:  "hash-1",
			AssistantID: &assistantID,
		},
	}
	messageRepo := &stubMessageRepository{
		messagesByID: map[string]*domain.Message{
			rootID: {
				ID:        rootID,
				SessionID: "session-1",
				Role:      "user",
				Content:   "Original question",
			},
			parentID: {
				ID:              parentID,
				SessionID:       "session-1",
				Role:            "assistant",
				Content:         "Original answer",
				ParentMessageID: &rootID,
			},
		},
	}
	assistant := &stubAssistantClient{
		createThreadID: "branch-thread-1",
		runAndStreamFunc: func(ctx context.Context, _ string, threadID, assistantID string, _ repository.StreamWriter) (string, error) {
			if err := ctx.Err(); err != nil {
				t.Fatalf("expected detached context during stream, got %v", err)
			}
			if threadID != "branch-thread-1" {
				t.Fatalf("unexpected thread id: %q", threadID)
			}
			if assistantID != "assistant-1" {
				t.Fatalf("unexpected assistant id: %q", assistantID)
			}
			return "Saved after disconnect", nil
		},
	}

	service := NewThreadService(sessionRepo, messageRepo, assistant)

	ctx, cancel := context.WithCancel(context.Background())
	err := service.Reply(ctx, ThreadRequest{
		APIKey:          "sk-test",
		APIKeyHash:      "hash-1",
		ParentMessageID: parentID,
		UserMessage:     "Follow-up question",
	}, &cancelOnStartStreamWriter{cancel: cancel})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messageRepo.savedMessages) != 2 {
		t.Fatalf("expected user and assistant messages to be saved, got %d", len(messageRepo.savedMessages))
	}
	if messageRepo.savedMessages[0].Role != "user" || messageRepo.savedMessages[0].Content != "Follow-up question" {
		t.Fatalf("unexpected saved user message: %+v", messageRepo.savedMessages[0])
	}
	if messageRepo.savedMessages[1].Role != "assistant" || messageRepo.savedMessages[1].Content != "Saved after disconnect" {
		t.Fatalf("unexpected saved assistant message: %+v", messageRepo.savedMessages[1])
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

func (s *stubSessionRepository) CreateWithPrompt(context.Context, string, string) (*domain.Session, error) {
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

type stubMessageRepository struct {
	firstRootUserMessage       *domain.Message
	messagesByID               map[string]*domain.Message
	updatedMessageID           string
	updatedContent             string
	branchThreadID             *string
	branchThreadLookupParentID string
	savedMessages              []domain.Message
}

func (m *stubMessageRepository) Save(_ context.Context, sessionID, role, content string, openAIThreadID, parentMessageID *string) (*domain.Message, error) {
	message := domain.Message{
		ID:        "saved-message",
		SessionID: sessionID,
		Role:      role,
		Content:   content,
	}
	if openAIThreadID != nil {
		message.OpenAIThreadID = stringPtr(*openAIThreadID)
	}
	if parentMessageID != nil {
		message.ParentMessageID = stringPtr(*parentMessageID)
	}
	m.savedMessages = append(m.savedMessages, message)
	return &message, nil
}

func (m *stubMessageRepository) GetMessageByID(_ context.Context, messageID string) (*domain.Message, error) {
	if m.messagesByID == nil {
		return nil, nil
	}
	return m.messagesByID[messageID], nil
}

func (m *stubMessageRepository) GetMainAsc(context.Context, string, int, int) ([]domain.Message, error) {
	return nil, nil
}

func (m *stubMessageRepository) GetMainDesc(context.Context, string, int, int) ([]domain.Message, error) {
	return nil, nil
}

func (m *stubMessageRepository) GetThreadAsc(context.Context, string, int, int) ([]domain.Message, error) {
	return nil, nil
}

func (m *stubMessageRepository) GetThreadDesc(context.Context, string, int, int) ([]domain.Message, error) {
	return nil, nil
}

func (m *stubMessageRepository) GetBranchThreadID(_ context.Context, parentMessageID string) (*string, error) {
	m.branchThreadLookupParentID = parentMessageID
	return m.branchThreadID, nil
}

func (m *stubMessageRepository) FindFirstRootUserMessage(context.Context, string) (*domain.Message, error) {
	return m.firstRootUserMessage, nil
}

func (m *stubMessageRepository) UpdateContent(_ context.Context, messageID, content string) error {
	m.updatedMessageID = messageID
	m.updatedContent = content
	return nil
}

type stubAssistantClient struct {
	createThreadID      string
	createThreadCalls   int
	addedMessages       []assistantMessageCall
	updatedInstructions string
	runAndStreamText    string
	runAndStreamFunc    func(context.Context, string, string, string, repository.StreamWriter) (string, error)
}

type assistantMessageCall struct {
	role     string
	threadID string
	content  string
}

func (a *stubAssistantClient) CreateAssistant(context.Context, string, string) (string, error) {
	return "assistant-created", nil
}

func (a *stubAssistantClient) CreateThread(context.Context, string) (string, error) {
	a.createThreadCalls++
	return a.createThreadID, nil
}

func (a *stubAssistantClient) AddUserMessage(_ context.Context, _ string, threadID, content string) error {
	a.addedMessages = append(a.addedMessages, assistantMessageCall{role: "user", threadID: threadID, content: content})
	return nil
}

func (a *stubAssistantClient) AddAssistantMessage(_ context.Context, _ string, threadID, content string) error {
	a.addedMessages = append(a.addedMessages, assistantMessageCall{role: "assistant", threadID: threadID, content: content})
	return nil
}

func (a *stubAssistantClient) RunAndStream(ctx context.Context, apiKey, threadID, assistantID string, stream repository.StreamWriter) (string, error) {
	if a.runAndStreamFunc != nil {
		return a.runAndStreamFunc(ctx, apiKey, threadID, assistantID, stream)
	}
	return a.runAndStreamText, nil
}

func (a *stubAssistantClient) UpdateAssistantInstructions(_ context.Context, _ string, _ string, instructions string) error {
	a.updatedInstructions = instructions
	return nil
}

type stubStreamWriter struct{}

func (s *stubStreamWriter) Start(string) error      { return nil }
func (s *stubStreamWriter) WriteChunk(string) error { return nil }
func (s *stubStreamWriter) Close() error            { return nil }

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
func (s *cancelOnStartStreamWriter) Close() error            { return nil }

func stringPtr(value string) *string {
	return &value
}
