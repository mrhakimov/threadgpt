package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"threadgpt/handlers"

	"github.com/joho/godotenv"
)

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
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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
	mux.HandleFunc("/api/auth", corsMiddleware(handlers.HandleAuth))
	mux.HandleFunc("/api/session", corsMiddleware(handlers.RequireAuth(handlers.HandleSession)))
	mux.HandleFunc("/api/sessions", corsMiddleware(handlers.RequireAuth(handlers.HandleSessions)))
	mux.HandleFunc("/api/sessions/", corsMiddleware(handlers.RequireAuth(handlers.HandleSessionByID)))
	mux.HandleFunc("/api/history", corsMiddleware(handlers.RequireAuth(handlers.HandleHistory)))
	mux.HandleFunc("/api/chat", corsMiddleware(handlers.RequireAuth(handlers.HandleChat)))
	mux.HandleFunc("/api/thread", corsMiddleware(handlers.RequireAuth(handlers.HandleThread)))

	fmt.Printf("ThreadGPT backend listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
