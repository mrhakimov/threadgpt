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
	"threadgpt/handlers"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/hkdf"
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

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:3000"
		}

		w.Header().Set("Vary", "Origin")
		origin := r.Header.Get("Origin")
		if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
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
		log.Fatal("SUPABASE_URL and SUPABASE_SERVICE_KEY must be set")
	}

	if os.Getenv("ALLOWED_ORIGIN") == "" {
		log.Println("WARNING: ALLOWED_ORIGIN not set; defaulting to http://localhost:3000 — set this in production")
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
	handlers.SetEncryptionKey(encKey)

	// Persist tokens to disk so they survive restarts.
	// Only useful when TOKEN_ENCRYPTION_KEY is stable (set in .env).
	if os.Getenv("TOKEN_ENCRYPTION_KEY") != "" {
		handlers.SetTokenStorePath(".token_store.json")
	}

	// Derive a separate HMAC key for API key hashing via HKDF-SHA256.
	hashKey := make([]byte, 32)
	hkdfReader := hkdf.New(sha256.New, encKey, nil, []byte("threadgpt-api-key-hash"))
	if _, err := io.ReadFull(hkdfReader, hashKey); err != nil {
		log.Fatal("failed to derive hash key:", err)
	}
	handlers.SetHashKey(hashKey)

	// Configure token store size (default 1000, override via MAX_TOKEN_STORE_SIZE).
	if v := os.Getenv("MAX_TOKEN_STORE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			handlers.SetMaxTokenStoreSize(n)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	go handlers.PurgeExpiredTokens()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth", securityHeaders(corsMiddleware(handlers.HandleAuth)))
	mux.HandleFunc("/api/auth/check", securityHeaders(corsMiddleware(handlers.HandleAuthCheck)))
	mux.HandleFunc("/api/auth/logout", securityHeaders(corsMiddleware(handlers.HandleLogout)))
	mux.HandleFunc("/api/session", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleSession))))
	mux.HandleFunc("/api/sessions", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleSessions))))
	mux.HandleFunc("/api/sessions/", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleSessionByID))))
	mux.HandleFunc("/api/history", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleHistory))))
	mux.HandleFunc("/api/chat", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleChat))))
	mux.HandleFunc("/api/thread", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleThread))))

	fmt.Printf("ThreadGPT backend listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
