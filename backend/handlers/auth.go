package handlers

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"threadgpt/domain"
)

type contextKey int

const (
	contextKeyAPIKey contextKey = iota
	contextKeyAPIKeyHash
)

func SetEncryptionKey(key []byte) {
	currentApp().SetEncryptionKey(key)
}

func SetHashKey(key []byte) {
	currentApp().SetHashKey(key)
}

func SetTokenStorePath(path string) {
	currentApp().SetTokenStorePath(path)
}

func SetMaxTokenStoreSize(n int) {
	currentApp().SetMaxTokenStoreSize(n)
}

func PurgeExpiredTokens() {
	currentApp().PurgeExpiredTokens()
}

func (a *Application) SetEncryptionKey(key []byte) {
	a.auth.SetEncryptionKey(key)
}

func (a *Application) SetHashKey(key []byte) {
	a.auth.SetHashKey(key)
}

func (a *Application) SetTokenStorePath(path string) {
	a.auth.SetTokenStorePath(path)
}

func (a *Application) SetMaxTokenStoreSize(n int) {
	a.auth.SetMaxTokenStoreSize(n)
}

func (a *Application) PurgeExpiredTokens() {
	a.auth.PurgeExpiredTokens()
}

func HandleAuth(w http.ResponseWriter, r *http.Request) {
	currentApp().HandleAuth(w, r)
}

func (a *Application) HandleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, newAPIError(http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed."))
		return
	}

	ip := remoteIP(r)
	if !a.auth.AllowAuth(ip) {
		writeServiceError(w, domain.ErrRateLimited)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	var req struct {
		APIKey string `json:"api_key"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, newAPIError(http.StatusBadRequest, "invalid_request", "The request body was invalid."))
		return
	}
	if len(req.APIKey) < 20 || !strings.HasPrefix(req.APIKey, "sk-") {
		writeAPIError(w, newAPIError(http.StatusBadRequest, "invalid_api_key", "That API key doesn't look valid. It should start with sk-."))
		return
	}

	if a.keyValidator != nil {
		if err := a.keyValidator.ValidateAPIKey(r.Context(), req.APIKey); err != nil {
			writeServiceError(w, err)
			return
		}
	}

	token, err := a.auth.Login(req.APIKey)
	if err != nil {
		writeServiceError(w, domain.ErrInternal)
		return
	}

	secure := r.TLS != nil || os.Getenv("COOKIE_SECURE") == "true" ||
		(strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") && os.Getenv("TRUSTED_PROXY") == "true")

	sameSite := http.SameSiteStrictMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "threadgpt_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: sameSite,
		Secure:   secure,
	})

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "token": token})
}

func HandleAuthCheck(w http.ResponseWriter, r *http.Request) {
	currentApp().HandleAuthCheck(w, r)
}

func (a *Application) HandleAuthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, newAPIError(http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed."))
		return
	}

	ip := remoteIP(r)
	if !a.auth.AllowChat(ip) {
		writeServiceError(w, domain.ErrRateLimited)
		return
	}

	token := tokenFromRequest(r)
	if token == "" {
		writeServiceError(w, domain.ErrUnauthorized)
		return
	}
	if err := a.auth.Check(token); err != nil {
		writeServiceError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
}

func HandleAuthInfo(w http.ResponseWriter, r *http.Request) {
	currentApp().HandleAuthInfo(w, r)
}

func (a *Application) HandleAuthInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, newAPIError(http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed."))
		return
	}

	token := tokenFromRequest(r)
	if token == "" {
		writeServiceError(w, domain.ErrUnauthorized)
		return
	}

	expiresAt, err := a.auth.GetExpiresAt(token)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"expires_at": expiresAt.UTC().Format("2006-01-02T15:04:05Z")})
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	currentApp().HandleLogout(w, r)
}

func (a *Application) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAPIError(w, newAPIError(http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed."))
		return
	}

	ip := remoteIP(r)
	if !a.auth.AllowChat(ip) {
		writeServiceError(w, domain.ErrRateLimited)
		return
	}

	if token := tokenFromRequest(r); token != "" {
		a.auth.Logout(token)
	}

	secure := r.TLS != nil || os.Getenv("COOKIE_SECURE") == "true" ||
		(strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") && os.Getenv("TRUSTED_PROXY") == "true")
	sameSite := http.SameSiteStrictMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{Name: "threadgpt_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: sameSite, Secure: secure})
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return currentApp().RequireAuth(next)
}

func (a *Application) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := tokenFromRequest(r)
		if token == "" {
			writeServiceError(w, domain.ErrUnauthorized)
			return
		}

		authContext, err := a.auth.Authorize(token)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyAPIKey, authContext.APIKey)
		ctx = context.WithValue(ctx, contextKeyAPIKeyHash, authContext.APIKeyHash)
		next(w, r.WithContext(ctx))
	}
}

// tokenFromRequest extracts the auth token from either the Authorization
// header (Bearer scheme) or the threadgpt_token cookie.
func tokenFromRequest(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	if cookie, err := r.Cookie("threadgpt_token"); err == nil {
		return cookie.Value
	}
	return ""
}

func APIKeyFromContext(ctx context.Context) string {
	value, _ := ctx.Value(contextKeyAPIKey).(string)
	return value
}

func APIKeyHashFromContext(ctx context.Context) string {
	value, _ := ctx.Value(contextKeyAPIKeyHash).(string)
	return value
}

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
