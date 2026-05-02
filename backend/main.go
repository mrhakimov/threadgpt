package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"threadgpt/data"
	"threadgpt/handlers"
	"threadgpt/service"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/hkdf"
)

const (
	defaultAllowedOrigin = "http://localhost:3000"
	defaultPort          = "8000"
)

func securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if os.Getenv("COOKIE_SECURE") == "true" {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		next(w, r)
	}
}

func isAllowedOrigin(origin, configured string) bool {
	if origin == configured {
		return true
	}
	if configured == "" || configured == defaultAllowedOrigin || configured == "dev" {
		if strings.HasPrefix(origin, "http://localhost:") ||
			strings.HasPrefix(origin, "http://127.0.0.1:") ||
			strings.HasSuffix(origin, ".ngrok-free.app") ||
			strings.HasSuffix(origin, ".ngrok.io") {
			return true
		}
	}
	return false
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = defaultAllowedOrigin
		}

		w.Header().Set("Vary", "Origin")
		origin := r.Header.Get("Origin")
		if isAllowedOrigin(origin, allowedOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Session-ID")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func main() {
	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	if os.Getenv("SUPABASE_URL") == "" || (os.Getenv("SUPABASE_SERVICE_KEY") == "" && os.Getenv("SUPABASE_SECRET_KEY") == "") {
		log.Fatal("SUPABASE_URL and either SUPABASE_SERVICE_KEY or SUPABASE_SECRET_KEY must be set")
	}

	if os.Getenv("ALLOWED_ORIGIN") == "" {
		log.Printf("WARNING: ALLOWED_ORIGIN not set; defaulting to %s; set this in production", defaultAllowedOrigin)
	}

	// Load or generate TOKEN_ENCRYPTION_KEY (32 bytes / 64 hex chars).
	var encKey []byte
	if keyHex := os.Getenv("TOKEN_ENCRYPTION_KEY"); keyHex != "" {
		k, err := hex.DecodeString(keyHex)
		if err != nil || len(k) != 32 {
			log.Fatal("TOKEN_ENCRYPTION_KEY must be exactly 64 hex characters (32 bytes)")
		}
		encKey = k
	} else {
		log.Println("WARNING: no TOKEN_ENCRYPTION_KEY set; generating ephemeral key — tokens will not survive restart")
		encKey = make([]byte, 32)
		if _, err := rand.Read(encKey); err != nil {
			log.Fatal("failed to generate encryption key:", err)
		}
	}

	store := data.NewSupabaseStore()
	assistant := data.NewOpenAIClient()
	auth := service.NewAuthService()
	app := handlers.NewApplication(handlers.Dependencies{
		Auth:         auth,
		Chat:         service.NewChatService(store, store, assistant),
		History:      service.NewHistoryService(store, store, assistant),
		Sessions:     service.NewSessionService(store, store, assistant),
		Threads:      service.NewThreadService(store, store, assistant),
		KeyValidator: assistant,
	})

	app.SetEncryptionKey(encKey)

	// Persist tokens to disk so they survive restarts.
	// Only useful when TOKEN_ENCRYPTION_KEY is stable (set in .env).
	if os.Getenv("TOKEN_ENCRYPTION_KEY") != "" {
		app.SetTokenStorePath(".token_store.json")
	}

	// Derive a separate HMAC key for API key hashing via HKDF-SHA256.
	hashKey := make([]byte, 32)
	hkdfReader := hkdf.New(sha256.New, encKey, nil, []byte("threadgpt-api-key-hash"))
	if _, err := io.ReadFull(hkdfReader, hashKey); err != nil {
		log.Fatal("failed to derive hash key:", err)
	}
	app.SetHashKey(hashKey)

	// Configure token store size (default 1000, override via MAX_TOKEN_STORE_SIZE).
	if v := os.Getenv("MAX_TOKEN_STORE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			app.SetMaxTokenStoreSize(n)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	go app.PurgeExpiredTokens()

	mux := http.NewServeMux()
	handle := func(pattern string, handler http.HandlerFunc) {
		mux.HandleFunc(pattern, securityHeaders(corsMiddleware(handler)))
	}

	handle("/api/auth", app.HandleAuth)
	handle("/api/auth/check", app.HandleAuthCheck)
	handle("/api/auth/info", app.HandleAuthInfo)
	handle("/api/auth/logout", app.HandleLogout)
	handle("/api/session", app.RequireAuth(app.HandleSession))
	handle("/api/sessions", app.RequireAuth(app.HandleSessions))
	handle("/api/sessions/", app.RequireAuth(app.HandleSessionByID))
	handle("/api/history", app.RequireAuth(app.HandleHistory))
	handle("/api/chat", app.RequireAuth(app.HandleChat))
	handle("/api/thread", app.RequireAuth(app.HandleThread))

	fmt.Printf("ThreadGPT backend listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
