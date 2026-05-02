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
	return s.doRequest(ctx, http.MethodPatch, filterPath("sessions", url.Values{
		"id": {"eq." + sessionID},
	}), map[string]string{"system_prompt": systemPrompt}, nil)
}

func (s *SupabaseStore) UpdateAssistant(ctx context.Context, sessionID, assistantID string) error {
	return s.doRequest(ctx, http.MethodPatch, filterPath("sessions", url.Values{
		"id": {"eq." + sessionID},
	}), map[string]string{"assistant_id": assistantID}, nil)
}

func (s *SupabaseStore) Delete(ctx context.Context, sessionID string) error {
	if err := s.doRequest(ctx, http.MethodDelete, filterPath("session_conversations", url.Values{
		"session_id": {"eq." + sessionID},
	}), nil, nil); err != nil {
		return err
	}
	return s.doRequest(ctx, http.MethodDelete, filterPath("sessions", url.Values{
		"id": {"eq." + sessionID},
	}), nil, nil)
}

func (s *SupabaseStore) Create(ctx context.Context, sessionID, conversationID string) (*domain.ConversationRef, error) {
	var refs []domain.ConversationRef
	err := s.doRequest(ctx, http.MethodPost, "session_conversations", map[string]string{
		"session_id":       sessionID,
		"conversation_id":  conversationID,
	}, &refs)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, domain.ErrInternal
	}
	return &refs[0], nil
}

func (s *SupabaseStore) GetByConversationID(ctx context.Context, conversationID string) (*domain.ConversationRef, error) {
	var refs []domain.ConversationRef
	err := s.doRequest(ctx, http.MethodGet, filterPath("session_conversations", url.Values{
		"conversation_id": {"eq." + conversationID},
		"limit":           {"1"},
	}), nil, &refs)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, nil
	}
	return &refs[0], nil
}

func (s *SupabaseStore) ListBySessionDesc(ctx context.Context, sessionID string, limit, offset int) ([]domain.ConversationRef, error) {
	var refs []domain.ConversationRef
	err := s.doRequest(ctx, http.MethodGet, filterPath("session_conversations", url.Values{
		"session_id": {"eq." + sessionID},
		"order":      {"created_at.desc"},
		"limit":      {fmt.Sprintf("%d", limit)},
		"offset":     {fmt.Sprintf("%d", offset)},
	}), nil, &refs)
	if err != nil {
		return nil, err
	}
	reverse(refs)
	return refs, nil
}

func (s *SupabaseStore) ListBySessionAsc(ctx context.Context, sessionID string) ([]domain.ConversationRef, error) {
	var refs []domain.ConversationRef
	err := s.doRequest(ctx, http.MethodGet, filterPath("session_conversations", url.Values{
		"session_id": {"eq." + sessionID},
		"order":      {"created_at.asc"},
		"limit":      {"10000"},
	}), nil, &refs)
	return refs, err
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
