# Plan: Secure API Key — Server-Side Session Tokens

## Context

The OpenAI API key is currently sent as a raw plaintext field in every JSON request body (`/api/session`, `/api/sessions`, `/api/chat`, `/api/thread`). Anyone inspecting browser DevTools network tab or intercepting HTTP traffic can read it on every request. The key should only cross the network **once** — during the initial session handshake — and all subsequent requests should use an opaque session token.

Secondary issue: `/api/history` accepts a raw `api_key` query parameter as a fallback, which can appear in server logs and browser history.

---

## Architecture

```
[App load / key entry]
  raw apiKey → POST /api/session { api_key }   ← only network transmission
       ↓
  Backend: generateToken(), store token→apiKey in memory, return { session_token, ... }
       ↓
  Frontend: _sessionToken (module-level var in api.ts, never localStorage)
       ↓
  All subsequent requests: Authorization: Bearer <token>
       ↓
  Backend: resolveToken(token) → apiKey → used for OpenAI calls
```

The raw key stays in React state **only** (never localStorage) for page-refresh re-exchange. On every page load, `initSession(apiKey)` is called once to exchange the key for a fresh token. If the page is refreshed and the key is gone from memory, the user re-enters it (ApiKeyGate handles this already).

---

## Backend Changes

### New file: `backend/handlers/store.go`
In-memory token store shared across all handlers. Tokens have a **1-hour TTL** to limit blast radius if one is intercepted. A background goroutine sweeps expired entries every 10 minutes.
```go
package handlers

import (
    "crypto/rand"
    "encoding/hex"
    "net/http"
    "sync"
    "time"
)

type tokenEntry struct {
    apiKey    string
    expiresAt time.Time
}

var (
    tokenStoreMu sync.RWMutex
    tokenStore   = make(map[string]tokenEntry) // token → entry
)

const tokenTTL = time.Hour

func init() {
    go func() {
        for range time.Tick(10 * time.Minute) {
            now := time.Now()
            tokenStoreMu.Lock()
            for t, e := range tokenStore {
                if now.After(e.expiresAt) {
                    delete(tokenStore, t)
                }
            }
            tokenStoreMu.Unlock()
        }
    }()
}

func storeToken(apiKey string) (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    token := hex.EncodeToString(b)
    tokenStoreMu.Lock()
    tokenStore[token] = tokenEntry{apiKey: apiKey, expiresAt: time.Now().Add(tokenTTL)}
    tokenStoreMu.Unlock()
    return token, nil
}

func resolveToken(token string) (string, bool) {
    tokenStoreMu.RLock()
    entry, ok := tokenStore[token]
    tokenStoreMu.RUnlock()
    if !ok || time.Now().After(entry.expiresAt) {
        return "", false
    }
    return entry.apiKey, true
}

func extractBearerToken(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    if len(auth) > 7 && auth[:7] == "Bearer " {
        return auth[7:]
    }
    return ""
}
```

### `backend/handlers/session.go`
- Add `SessionToken string` to `SessionResponse`
- `HandleSession`: call `storeToken(req.APIKey)`, include token in response
- `handleCreateNamedSession`: remove `APIKey` from `CreateSessionRequest`, resolve key via `extractBearerToken` + `resolveToken`

### `backend/handlers/chat.go`
- Remove `APIKey` from `ChatRequest`
- `HandleChat`: resolve key via token at the top, use `apiKey` variable throughout (all OpenAI calls remain unchanged)

### `backend/handlers/thread.go`
- Remove `APIKey` from `ThreadRequest`
- `HandleThread`: same token-resolution pattern as chat.go

### `backend/handlers/history.go`
- Remove the `api_key` raw query param fallback (lines ~30-35); return 400 if `api_key_hash` is also missing

---

## Frontend Changes

### `frontend/src/lib/api.ts`
- Add module-level `let _sessionToken: string | null = null` — memory-only, never written to localStorage/sessionStorage
- Add `authHeader()` helper returning `{ "Content-Type": "application/json", "Authorization": "Bearer <token>" }`
- Add `authedFetch()` wrapper: calls `fetch`, and if response is **401**, clears `_sessionToken` and throws `"Session expired. Please re-enter your API key."` — this handles token expiry and server restarts gracefully
- `initSession`: still sends raw key once, stores returned `session_token` into `_sessionToken`
- `createSession(name)`: drop `apiKey` param, use `authHeader()`
- `sendChatMessage(userMessage, onChunk, sessionId)`: drop `apiKey` param, use `authedFetch()`
- `sendThreadMessage(parentMessageId, userMessage, onChunk)`: drop `apiKey` param, use `authedFetch()`
- `fetchSessions(apiKey)` and `fetchHistory(apiKey, sessionId?)`: unchanged (already send only hash in query params)

### `frontend/src/hooks/useChat.ts`
- In `init()`: always call `initSession(apiKey)` first (even when `sessionId` is known) so `_sessionToken` is populated before any authed call
- Remove `apiKey` as argument from `sendChatMessage(...)` calls

### `frontend/src/hooks/useThread.ts`
- Remove `apiKey` parameter from hook signature
- Remove `apiKey` argument from `sendThreadMessage(...)` call

### `frontend/src/components/ThreadDrawer.tsx`
- Remove `apiKey` from `Props` interface and destructuring
- Update `useThread(parentMessage.id)` call (drop `apiKey`)

### `frontend/src/components/ChatView.tsx`
- Remove `apiKey={apiKey}` prop from `<ThreadDrawer>` usage

### `frontend/src/components/ConversationMenu.tsx`
- Change `createSession(apiKey, name)` → `createSession(name)`

---

## Frontend Storage Change

### `frontend/src/app/page.tsx`
- **Remove** `localStorage.setItem(STORAGE_KEY, key)` and `localStorage.getItem(STORAGE_KEY)` — the raw key must not persist to localStorage (XSS-readable)
- Keep `apiKey` in React state only (`useState`)
- On page refresh the key is gone; `ApiKeyGate` re-renders and the user re-enters it — this is the correct UX for a security-sensitive credential

---

## Files NOT Changed
- `backend/main.go` — CORS already allows `Authorization` header; no changes needed
- `backend/openai/client.go` — receives raw key from handlers; unchanged
- `backend/db/supabase.go` — stores only hash; unchanged
- `frontend/src/components/SettingsPage.tsx` — receives `apiKey` for masked display only; unchanged

---

## Implementation Order

1. Create `backend/handlers/store.go`
2. Update `backend/handlers/session.go`
3. Update `backend/handlers/chat.go`
4. Update `backend/handlers/thread.go`
5. Update `backend/handlers/history.go`
6. Update `frontend/src/lib/api.ts`
7. Update `frontend/src/hooks/useChat.ts`
8. Update `frontend/src/hooks/useThread.ts`
9. Update `frontend/src/components/ThreadDrawer.tsx`
10. Update `frontend/src/components/ChatView.tsx`
11. Update `frontend/src/components/ConversationMenu.tsx`
12. Update `frontend/src/app/page.tsx` — remove localStorage persistence

## Verification

1. `cd backend && go build ./...` — must compile with no errors
2. `cd frontend && npm run build` — TypeScript must have no type errors
3. Manual test: open DevTools → Network tab → enter API key → confirm only `/api/session` request body contains `api_key`; all subsequent requests show `Authorization: Bearer ...` header with no `api_key` in body
4. Open DevTools → Application → Local Storage → confirm `threadgpt_api_key` key no longer exists
5. Test page refresh: confirm `ApiKeyGate` re-renders (key is not remembered), user re-enters key, chat works
6. Test token expiry simulation: manually clear `_sessionToken` in browser console → next request should surface "Session expired" error
