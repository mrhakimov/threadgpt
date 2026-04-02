package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"sync"
	"threadgpt/domain"
	"time"
)

const (
	MaxAuthPerMinute = 10
	MaxChatPerMinute = 60
)

type AuthContext struct {
	APIKey     string
	APIKeyHash string
}

type AuthService struct {
	tokenMu           sync.RWMutex
	tokenStore        map[string]tokenEntry
	encryptionKey     []byte
	hashKey           []byte
	tokenStorePath    string
	maxTokenStoreSize int
	authLimiter       *RateLimiter
	chatLimiter       *RateLimiter
}

type tokenEntry struct {
	EncryptedAPIKey string    `json:"e"`
	APIKeyHash      string    `json:"h"`
	ExpiresAt       time.Time `json:"x"`
}

func NewAuthService() *AuthService {
	return &AuthService{
		tokenStore:        map[string]tokenEntry{},
		maxTokenStoreSize: 1000,
		authLimiter:       NewRateLimiter(50000),
		chatLimiter:       NewRateLimiter(50000),
	}
}

func (s *AuthService) SetEncryptionKey(key []byte) {
	s.encryptionKey = key
}

func (s *AuthService) SetHashKey(key []byte) {
	s.hashKey = key
}

func (s *AuthService) SetTokenStorePath(path string) {
	s.tokenStorePath = path
	s.loadTokenStore()
}

func (s *AuthService) SetMaxTokenStoreSize(n int) {
	s.maxTokenStoreSize = n
}

func (s *AuthService) AllowAuth(key string) bool {
	return s.authLimiter.Allow(key, MaxAuthPerMinute)
}

func (s *AuthService) AllowChat(key string) bool {
	return s.chatLimiter.Allow(key, MaxChatPerMinute)
}

func (s *AuthService) Login(apiKey string) (string, error) {
	token, err := s.generateToken()
	if err != nil {
		return "", err
	}
	if err := s.storeToken(token, apiKey, s.HashAPIKey(apiKey)); err != nil {
		return "", err
	}
	return token, nil
}

func (s *AuthService) Check(token string) error {
	if _, ok := s.lookupToken(token); !ok {
		return domain.ErrUnauthorized
	}
	return nil
}

func (s *AuthService) Logout(token string) {
	s.tokenMu.Lock()
	delete(s.tokenStore, token)
	s.tokenMu.Unlock()
	go s.saveTokenStore()
}

func (s *AuthService) Authorize(token string) (*AuthContext, error) {
	entry, ok := s.lookupToken(token)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	apiKey, err := s.decryptAPIKey(entry.EncryptedAPIKey)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	return &AuthContext{
		APIKey:     apiKey,
		APIKeyHash: entry.APIKeyHash,
	}, nil
}

func (s *AuthService) HashAPIKey(apiKey string) string {
	mac := hmac.New(sha256.New, s.hashKey)
	mac.Write([]byte(apiKey))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *AuthService) PurgeExpiredTokens() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.tokenMu.Lock()
		for token, entry := range s.tokenStore {
			if time.Now().After(entry.ExpiresAt) {
				delete(s.tokenStore, token)
			}
		}
		s.tokenMu.Unlock()
		s.saveTokenStore()
		s.authLimiter.PurgeExpired()
		s.chatLimiter.PurgeExpired()
	}
}

func (s *AuthService) generateToken() (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return hex.EncodeToString(random), nil
}

func (s *AuthService) storeToken(token, rawAPIKey, apiKeyHash string) error {
	encrypted, err := s.encryptAPIKey(rawAPIKey)
	if err != nil {
		return err
	}

	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	if len(s.tokenStore) >= s.maxTokenStoreSize {
		var evictKey string
		var earliest time.Time
		for key, entry := range s.tokenStore {
			if evictKey == "" || entry.ExpiresAt.Before(earliest) {
				evictKey = key
				earliest = entry.ExpiresAt
			}
		}
		if evictKey != "" {
			delete(s.tokenStore, evictKey)
		}
	}

	s.tokenStore[token] = tokenEntry{
		EncryptedAPIKey: encrypted,
		APIKeyHash:      apiKeyHash,
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}
	go s.saveTokenStore()
	return nil
}

func (s *AuthService) lookupToken(token string) (tokenEntry, bool) {
	s.tokenMu.RLock()
	defer s.tokenMu.RUnlock()

	entry, ok := s.tokenStore[token]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return tokenEntry{}, false
	}
	return entry, true
}

func (s *AuthService) encryptAPIKey(raw string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(raw), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *AuthService) decryptAPIKey(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *AuthService) loadTokenStore() {
	if s.tokenStorePath == "" {
		return
	}
	data, err := os.ReadFile(s.tokenStorePath)
	if err != nil {
		return
	}

	var persisted map[string]tokenEntry
	if err := json.Unmarshal(data, &persisted); err != nil {
		return
	}

	now := time.Now()
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	for token, entry := range persisted {
		if now.Before(entry.ExpiresAt) {
			s.tokenStore[token] = entry
		}
	}
}

func (s *AuthService) saveTokenStore() {
	if s.tokenStorePath == "" {
		return
	}

	s.tokenMu.RLock()
	persisted := make(map[string]tokenEntry, len(s.tokenStore))
	for token, entry := range s.tokenStore {
		persisted[token] = entry
	}
	s.tokenMu.RUnlock()

	data, err := json.Marshal(persisted)
	if err != nil {
		return
	}
	_ = os.WriteFile(s.tokenStorePath, data, 0600)
}
