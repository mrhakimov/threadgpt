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
	"os"
	"regexp"
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

// hashKey is set once at startup via SetHashKey (derived separately from encryptionKey).
var hashKey []byte

// SetEncryptionKey stores the 32-byte AES key used for encrypting API keys at rest.
func SetEncryptionKey(key []byte) {
	encryptionKey = key
}

// SetHashKey stores the 32-byte HMAC key used for hashing API keys for DB lookup.
func SetHashKey(key []byte) {
	hashKey = key
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

var maxTokenStoreSize = 1000

// SetMaxTokenStoreSize overrides the default token store size limit.
func SetMaxTokenStoreSize(n int) {
	maxTokenStoreSize = n
}

var (
	authRateMu  sync.Mutex
	authRateMap = map[string][]time.Time{}
)

const maxAuthPerMinute = 10

var (
	chatRateMu  sync.Mutex
	chatRateMap = map[string][]time.Time{}
)

const maxChatPerMinute = 60

const maxRateLimitMapSize = 50000

// uuidRe matches standard UUID v4 format (case-insensitive).
var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func isValidUUID(s string) bool { return uuidRe.MatchString(s) }

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
		// Evict the soonest-expiring entry instead of rejecting
		var evictKey string
		var earliest time.Time
		for k, v := range tokenStore {
			if evictKey == "" || v.expiresAt.Before(earliest) {
				evictKey = k
				earliest = v.expiresAt
			}
		}
		if evictKey != "" {
			delete(tokenStore, evictKey)
		}
	}
	tokenStore[token] = tokenEntry{
		encryptedAPIKey: enc,
		apiKeyHash:      apiKeyHash,
		expiresAt:       time.Now().Add(24 * time.Hour),
	}
	return nil
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

		// Purge stale auth rate-limit entries
		cutoff := time.Now().Add(-time.Minute)
		authRateMu.Lock()
		for ip, times := range authRateMap {
			var keep []time.Time
			for _, t := range times {
				if t.After(cutoff) {
					keep = append(keep, t)
				}
			}
			if len(keep) == 0 {
				delete(authRateMap, ip)
			} else {
				authRateMap[ip] = keep
			}
		}
		authRateMu.Unlock()

		// Purge stale chat rate-limit entries
		chatRateMu.Lock()
		for key, times := range chatRateMap {
			var keep []time.Time
			for _, t := range times {
				if t.After(cutoff) {
					keep = append(keep, t)
				}
			}
			if len(keep) == 0 {
				delete(chatRateMap, key)
			} else {
				chatRateMap[key] = keep
			}
		}
		chatRateMu.Unlock()
	}
}

// remoteIP extracts the client IP. X-Real-IP is only trusted when TRUSTED_PROXY=true.
func remoteIP(r *http.Request) string {
	if os.Getenv("TRUSTED_PROXY") == "true" {
		if ip := r.Header.Get("X-Real-IP"); ip != "" && net.ParseIP(ip) != nil {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// checkRateLimit returns true if the caller is within limit, false if exceeded.
func checkRateLimit(key string, limit int, mu *sync.Mutex, m map[string][]time.Time) bool {
	now := time.Now()
	mu.Lock()
	defer mu.Unlock()
	timestamps := m[key]
	cutoff := now.Add(-time.Minute)
	filtered := timestamps[:0]
	for _, t := range timestamps {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= limit {
		return false
	}
	if _, exists := m[key]; !exists && len(m) >= maxRateLimitMapSize {
		// Evict one arbitrary entry to cap memory usage
		for k := range m {
			delete(m, k)
			break
		}
	}
	m[key] = append(filtered, now)
	return true
}

func HandleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Per-IP rate limiting
	ip := remoteIP(r)
	if !checkRateLimit(ip, maxAuthPerMinute, &authRateMu, authRateMap) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
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
	if err := storeToken(token, req.APIKey, hash); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	secure := r.TLS != nil || os.Getenv("COOKIE_SECURE") == "true" ||
		strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") && os.Getenv("TRUSTED_PROXY") == "true"
	http.SetCookie(w, &http.Cookie{
		Name:     "threadgpt_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func HandleAuthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := remoteIP(r)
	if !checkRateLimit(ip, maxChatPerMinute, &chatRateMu, chatRateMap) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	cookie, err := r.Cookie("threadgpt_token")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, ok := lookupToken(cookie.Value); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := remoteIP(r)
	if !checkRateLimit(ip, maxChatPerMinute, &chatRateMu, chatRateMap) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	if cookie, err := r.Cookie("threadgpt_token"); err == nil {
		tokenMu.Lock()
		delete(tokenStore, cookie.Value)
		tokenMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "threadgpt_token", Value: "", Path: "/", MaxAge: -1})
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("threadgpt_token")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := cookie.Value
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
