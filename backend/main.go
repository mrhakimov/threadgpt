package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"threadgpt/handlers"

	"github.com/joho/godotenv"
)

func securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next(w, r)
	}
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:3000"
		}

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	go handlers.PurgeExpiredTokens()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth", securityHeaders(corsMiddleware(handlers.HandleAuth)))
	mux.HandleFunc("/api/session", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleSession))))
	mux.HandleFunc("/api/sessions", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleSessions))))
	mux.HandleFunc("/api/sessions/", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleSessionByID))))
	mux.HandleFunc("/api/history", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleHistory))))
	mux.HandleFunc("/api/chat", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleChat))))
	mux.HandleFunc("/api/thread", securityHeaders(corsMiddleware(handlers.RequireAuth(handlers.HandleThread))))

	fmt.Printf("ThreadGPT backend listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
