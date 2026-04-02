package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"threadgpt/domain"
	"time"
)

type SupabaseStore struct {
	client *http.Client
}

func NewSupabaseStore() *SupabaseStore {
	return &SupabaseStore{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *SupabaseStore) GetLatestByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.Session, error) {
	var sessions []domain.Session
	err := s.doRequest(ctx, http.MethodGet, filterPath("sessions", url.Values{
		"api_key_hash": {"eq." + apiKeyHash},
		"order":        {"created_at.desc"},
		"limit":        {"1"},
	}), nil, &sessions)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	return &sessions[0], nil
}

func (s *SupabaseStore) GetByID(ctx context.Context, sessionID string) (*domain.Session, error) {
	var sessions []domain.Session
	err := s.doRequest(ctx, http.MethodGet, filterPath("sessions", url.Values{
		"id":    {"eq." + sessionID},
		"limit": {"1"},
	}), nil, &sessions)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	return &sessions[0], nil
}

func (s *SupabaseStore) ListByAPIKeyHash(ctx context.Context, apiKeyHash string, limit, offset int) ([]domain.Session, error) {
	var sessions []domain.Session
	err := s.doRequest(ctx, http.MethodGet, filterPath("sessions", url.Values{
		"api_key_hash": {"eq." + apiKeyHash},
		"order":        {"created_at.desc"},
		"limit":        {fmt.Sprintf("%d", limit)},
		"offset":       {fmt.Sprintf("%d", offset)},
	}), nil, &sessions)
	return sessions, err
}

func (s *SupabaseStore) CreateWithPrompt(ctx context.Context, apiKeyHash, systemPrompt string) (*domain.Session, error) {
	var sessions []domain.Session
	err := s.doRequest(ctx, http.MethodPost, "sessions", map[string]string{
		"api_key_hash":  apiKeyHash,
		"system_prompt": systemPrompt,
	}, &sessions)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, domain.ErrInternal
	}
	return &sessions[0], nil
}

func (s *SupabaseStore) CreateNamed(ctx context.Context, apiKeyHash, name string) (*domain.Session, error) {
	var sessions []domain.Session
	err := s.doRequest(ctx, http.MethodPost, "sessions", map[string]string{
		"api_key_hash": apiKeyHash,
		"name":         name,
	}, &sessions)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, domain.ErrInternal
	}
	return &sessions[0], nil
}

func (s *SupabaseStore) Rename(ctx context.Context, sessionID, name string) error {
	return s.doRequest(ctx, http.MethodPatch, filterPath("sessions", url.Values{
		"id": {"eq." + sessionID},
	}), map[string]string{"name": name}, nil)
}

func (s *SupabaseStore) SetSystemPrompt(ctx context.Context, sessionID, systemPrompt string) error {
	if err := s.doRequest(ctx, http.MethodPatch, filterPath("sessions", url.Values{
		"id": {"eq." + sessionID},
	}), map[string]string{"system_prompt": systemPrompt}, nil); err != nil {
		return err
	}

	firstMessage, err := s.FindFirstRootUserMessage(ctx, sessionID)
	if err != nil {
		return err
	}
	if firstMessage == nil {
		return nil
	}

	return s.UpdateContent(ctx, firstMessage.ID, systemPrompt)
}

func (s *SupabaseStore) UpdateAssistant(ctx context.Context, sessionID, assistantID string) error {
	return s.doRequest(ctx, http.MethodPatch, filterPath("sessions", url.Values{
		"id": {"eq." + sessionID},
	}), map[string]string{"assistant_id": assistantID}, nil)
}

func (s *SupabaseStore) Delete(ctx context.Context, sessionID string) error {
	if err := s.doRequest(ctx, http.MethodDelete, filterPath("messages", url.Values{
		"session_id": {"eq." + sessionID},
	}), nil, nil); err != nil {
		return err
	}
	return s.doRequest(ctx, http.MethodDelete, filterPath("sessions", url.Values{
		"id": {"eq." + sessionID},
	}), nil, nil)
}

func (s *SupabaseStore) Save(ctx context.Context, sessionID, role, content string, openAIThreadID, parentMessageID *string) (*domain.Message, error) {
	payload := map[string]any{
		"session_id": sessionID,
		"role":       role,
		"content":    content,
	}
	if openAIThreadID != nil {
		payload["openai_thread_id"] = *openAIThreadID
	}
	if parentMessageID != nil {
		payload["parent_message_id"] = *parentMessageID
	}

	var messages []domain.Message
	err := s.doRequest(ctx, http.MethodPost, "messages", payload, &messages)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, domain.ErrInternal
	}
	return &messages[0], nil
}

func (s *SupabaseStore) GetMessageByID(ctx context.Context, messageID string) (*domain.Message, error) {
	var messages []domain.Message
	err := s.doRequest(ctx, http.MethodGet, filterPath("messages", url.Values{
		"id":    {"eq." + messageID},
		"limit": {"1"},
	}), nil, &messages)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return &messages[0], nil
}

func (s *SupabaseStore) GetMainAsc(ctx context.Context, sessionID string, limit, offset int) ([]domain.Message, error) {
	var messages []domain.Message
	err := s.doRequest(ctx, http.MethodGet, filterPath("messages", url.Values{
		"session_id":        {"eq." + sessionID},
		"parent_message_id": {"is.null"},
		"order":             {"created_at.asc"},
		"limit":             {fmt.Sprintf("%d", limit)},
		"offset":            {fmt.Sprintf("%d", offset)},
	}), nil, &messages)
	if err != nil {
		return nil, err
	}
	return s.enrichReplyCounts(ctx, sessionID, messages)
}

func (s *SupabaseStore) GetMainDesc(ctx context.Context, sessionID string, limit, offset int) ([]domain.Message, error) {
	var messages []domain.Message
	err := s.doRequest(ctx, http.MethodGet, filterPath("messages", url.Values{
		"session_id":        {"eq." + sessionID},
		"parent_message_id": {"is.null"},
		"order":             {"created_at.desc"},
		"limit":             {fmt.Sprintf("%d", limit)},
		"offset":            {fmt.Sprintf("%d", offset)},
	}), nil, &messages)
	if err != nil {
		return nil, err
	}

	reverse(messages)
	return s.enrichReplyCounts(ctx, sessionID, messages)
}

func (s *SupabaseStore) GetThreadAsc(ctx context.Context, parentMessageID string, limit, offset int) ([]domain.Message, error) {
	var messages []domain.Message
	err := s.doRequest(ctx, http.MethodGet, filterPath("messages", url.Values{
		"parent_message_id": {"eq." + parentMessageID},
		"order":             {"created_at.asc"},
		"limit":             {fmt.Sprintf("%d", limit)},
		"offset":            {fmt.Sprintf("%d", offset)},
	}), nil, &messages)
	return messages, err
}

func (s *SupabaseStore) GetThreadDesc(ctx context.Context, parentMessageID string, limit, offset int) ([]domain.Message, error) {
	var messages []domain.Message
	err := s.doRequest(ctx, http.MethodGet, filterPath("messages", url.Values{
		"parent_message_id": {"eq." + parentMessageID},
		"order":             {"created_at.desc"},
		"limit":             {fmt.Sprintf("%d", limit)},
		"offset":            {fmt.Sprintf("%d", offset)},
	}), nil, &messages)
	if err != nil {
		return nil, err
	}
	reverse(messages)
	return messages, nil
}

func (s *SupabaseStore) FindFirstRootUserMessage(ctx context.Context, sessionID string) (*domain.Message, error) {
	var messages []domain.Message
	err := s.doRequest(ctx, http.MethodGet, filterPath("messages", url.Values{
		"session_id":        {"eq." + sessionID},
		"role":              {"eq.user"},
		"parent_message_id": {"is.null"},
		"order":             {"created_at.asc"},
		"limit":             {"1"},
	}), nil, &messages)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return &messages[0], nil
}

func (s *SupabaseStore) UpdateContent(ctx context.Context, messageID, content string) error {
	return s.doRequest(ctx, http.MethodPatch, filterPath("messages", url.Values{
		"id": {"eq." + messageID},
	}), map[string]string{"content": content}, nil)
}

func (s *SupabaseStore) enrichReplyCounts(ctx context.Context, sessionID string, messages []domain.Message) ([]domain.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	var threadMessages []domain.Message
	if err := s.doRequest(ctx, http.MethodGet, filterPath("messages", url.Values{
		"session_id":        {"eq." + sessionID},
		"parent_message_id": {"not.is.null"},
		"role":              {"eq.user"},
	}), nil, &threadMessages); err != nil {
		return messages, nil
	}

	counts := make(map[string]int)
	for _, message := range threadMessages {
		if message.ParentMessageID != nil {
			counts[*message.ParentMessageID]++
		}
	}

	for i := range messages {
		messages[i].ReplyCount = counts[messages[i].ID]
	}

	return messages, nil
}

func (s *SupabaseStore) doRequest(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, supabaseURL()+"/rest/v1/"+path, bodyReader)
	if err != nil {
		return err
	}

	key := supabaseKey()
	req.Header.Set("apikey", key)
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	if result != nil {
		req.Header.Set("Accept", "application/json")
	}
	if method == http.MethodPost {
		req.Header.Set("Prefer", "return=representation")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		log.Printf("supabase error %d: %s", resp.StatusCode, truncateForLog(respBody))
		return domain.ErrInternal
	}
	if result != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, result)
	}
	return nil
}

func supabaseURL() string {
	return os.Getenv("SUPABASE_URL")
}

func supabaseKey() string {
	if key := os.Getenv("SUPABASE_SERVICE_KEY"); key != "" {
		return key
	}
	return os.Getenv("SUPABASE_SECRET_KEY")
}

func filterPath(table string, params url.Values) string {
	return table + "?" + params.Encode()
}

func truncateForLog(b []byte) string {
	s := string(b)
	if len(s) > 200 {
		s = s[:200] + "...[truncated]"
	}
	return s
}

func reverse[T any](items []T) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}
