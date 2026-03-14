package handlers

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey int

const (
	contextKeyAPIKey     contextKey = iota
	contextKeyAPIKeyHash contextKey = iota
)

// encryptionKey is set once at startup via SetEncryptionKey.
var encryptionKey []byte

// SetEncryptionKey stores the 32-byte AES key used for encrypting API keys at rest.
func SetEncryptionKey(key []byte) {
	encryptionKey = key
}

func encryptAPIKey(raw string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
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

func decryptAPIKey(enc string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(encryptionKey)
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

type tokenEntry struct {
	encryptedAPIKey string
	apiKeyHash      string
	expiresAt       time.Time
}

var (
	tokenMu    sync.RWMutex
	tokenStore = map[string]tokenEntry{}
)

const maxTokenStoreSize = 1000

var (
	authRateMu  sync.Mutex
	authRateMap = map[string][]time.Time{}
)

const maxAuthPerMinute = 10

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func storeToken(token, rawAPIKey, apiKeyHash string) error {
	enc, err := encryptAPIKey(rawAPIKey)
	if err != nil {
		return err
	}
	tokenMu.Lock()
	defer tokenMu.Unlock()
	if len(tokenStore) >= maxTokenStoreSize {
		return errTokenStoreFull
	}
	tokenStore[token] = tokenEntry{
		encryptedAPIKey: enc,
		apiKeyHash:      apiKeyHash,
		expiresAt:       time.Now().Add(24 * time.Hour),
	}
	return nil
}

// errTokenStoreFull is a sentinel used internally.
var errTokenStoreFull = &tokenStoreFullError{}

type tokenStoreFullError struct{}

func (e *tokenStoreFullError) Error() string { return "token store full" }

func lookupToken(token string) (tokenEntry, bool) {
	tokenMu.RLock()
	defer tokenMu.RUnlock()
	entry, ok := tokenStore[token]
	if !ok || time.Now().After(entry.expiresAt) {
		return tokenEntry{}, false
	}
	return entry, true
}

func PurgeExpiredTokens() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		tokenMu.Lock()
		for token, entry := range tokenStore {
			if time.Now().After(entry.expiresAt) {
				delete(tokenStore, token)
			}
		}
		tokenMu.Unlock()
	}
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func HandleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Per-IP rate limiting
	ip := remoteIP(r)
	now := time.Now()
	authRateMu.Lock()
	timestamps := authRateMap[ip]
	cutoff := now.Add(-time.Minute)
	filtered := timestamps[:0]
	for _, t := range timestamps {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= maxAuthPerMinute {
		authRateMu.Unlock()
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	filtered = append(filtered, now)
	authRateMap[ip] = filtered
	authRateMu.Unlock()

	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	var req struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if len(req.APIKey) < 20 || !strings.HasPrefix(req.APIKey, "sk-") {
		http.Error(w, "invalid api key", http.StatusBadRequest)
		return
	}

	token, err := generateToken()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	hash := hashAPIKey(req.APIKey)
	if err := storeToken(token, req.APIKey, hash); err != nil {
		if err == errTokenStoreFull {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		entry, ok := lookupToken(token)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		rawAPIKey, err := decryptAPIKey(entry.encryptedAPIKey)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyAPIKey, rawAPIKey)
		ctx = context.WithValue(ctx, contextKeyAPIKeyHash, entry.apiKeyHash)
		next(w, r.WithContext(ctx))
	}
}

func APIKeyFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyAPIKey).(string)
	return v
}

func APIKeyHashFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyAPIKeyHash).(string)
	return v
}
