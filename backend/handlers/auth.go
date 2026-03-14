package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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

type tokenEntry struct {
	rawAPIKey  string
	apiKeyHash string
	expiresAt  time.Time
}

var (
	tokenMu    sync.RWMutex
	tokenStore = map[string]tokenEntry{}
)

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func storeToken(token, rawAPIKey, apiKeyHash string) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	tokenStore[token] = tokenEntry{
		rawAPIKey:  rawAPIKey,
		apiKeyHash: apiKeyHash,
		expiresAt:  time.Now().Add(24 * time.Hour),
	}
}

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

func HandleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
	storeToken(token, req.APIKey, hash)

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
		ctx := context.WithValue(r.Context(), contextKeyAPIKey, entry.rawAPIKey)
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
