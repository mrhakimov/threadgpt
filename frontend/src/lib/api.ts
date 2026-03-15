const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8000"

async function handleError(res: Response): Promise<never> {
  if (res.status >= 500) throw new Error("Server error, please try again.")
  if (res.status === 401 || res.status === 403) throw new Error("Unauthorized. Please re-enter your API key.")
  if (res.status === 404) throw new Error("Not found.")
  if (res.status === 429) throw new Error("Too many requests. Please wait a moment.")
  throw new Error("Something went wrong.")
}

const jsonHeaders = { "Content-Type": "application/json" }

export async function auth(apiKey: string): Promise<void> {
  const res = await fetch(`${API_URL}/api/auth`, {
    method: "POST",
    headers: jsonHeaders,
    credentials: "include",
    body: JSON.stringify({ api_key: apiKey }),
  })
  if (!res.ok) return handleError(res)
}

export async function checkAuth(): Promise<boolean> {
  try {
    const res = await fetch(`${API_URL}/api/auth/check`, {
      credentials: "include",
    })
    return res.ok
  } catch {
    return false
  }
}

export async function logout(): Promise<void> {
  await fetch(`${API_URL}/api/auth/logout`, {
    method: "DELETE",
    credentials: "include",
  })
}

export async function initSession() {
  const res = await fetch(`${API_URL}/api/session`, {
    method: "POST",
    headers: jsonHeaders,
    credentials: "include",
    body: JSON.stringify({}),
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function fetchSession(sessionId: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`, {
    credentials: "include",
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function fetchSessions(limit = 20, offset = 0): Promise<{ sessions: import("@/types").Session[], has_more: boolean }> {
  const res = await fetch(`${API_URL}/api/sessions?limit=${limit}&offset=${offset}`, {
    credentials: "include",
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function createSession(name: string) {
  const res = await fetch(`${API_URL}/api/sessions`, {
    method: "POST",
    headers: jsonHeaders,
    credentials: "include",
    body: JSON.stringify({ name }),
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function renameSession(sessionId: string, name: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`, {
    method: "PATCH",
    headers: jsonHeaders,
    credentials: "include",
    body: JSON.stringify({ name }),
  })
  if (!res.ok) return handleError(res)
}

export async function updateSystemPrompt(sessionId: string, systemPrompt: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`, {
    method: "PATCH",
    headers: jsonHeaders,
    credentials: "include",
    body: JSON.stringify({ system_prompt: systemPrompt }),
  })
  if (!res.ok) return handleError(res)
}

export async function deleteSession(sessionId: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`, {
    method: "DELETE",
    credentials: "include",
  })
  if (!res.ok) return handleError(res)
}

export async function fetchHistory(sessionId?: string, limit = 50, offset = 0): Promise<{ messages: import("@/types").Message[], has_more: boolean }> {
  const headers: Record<string, string> = {}
  if (sessionId) headers["X-Session-ID"] = sessionId
  const res = await fetch(`${API_URL}/api/history?limit=${limit}&offset=${offset}`, {
    headers,
    credentials: "include",
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function sendChatMessage(
  userMessage: string,
  onChunk: (chunk: string) => void,
  sessionId?: string,
  forceNew?: boolean
): Promise<string | undefined> {
  const res = await fetch(`${API_URL}/api/chat`, {
    method: "POST",
    headers: jsonHeaders,
    credentials: "include",
    body: JSON.stringify({ user_message: userMessage, session_id: sessionId ?? "", force_new: forceNew ?? false }),
  })
  if (!res.ok) return handleError(res)
  return consumeStream(res, onChunk)
}

export async function fetchThreadMessages(parentMessageId: string, limit = 50, offset = 0): Promise<{ messages: import("@/types").Message[], has_more: boolean }> {
  const res = await fetch(`${API_URL}/api/thread?parent_message_id=${encodeURIComponent(parentMessageId)}&limit=${limit}&offset=${offset}`, {
    credentials: "include",
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function sendThreadMessage(
  parentMessageId: string,
  userMessage: string,
  onChunk: (chunk: string) => void
): Promise<void> {
  const res = await fetch(`${API_URL}/api/thread`, {
    method: "POST",
    headers: jsonHeaders,
    credentials: "include",
    body: JSON.stringify({
      parent_message_id: parentMessageId,
      user_message: userMessage,
    }),
  })
  if (!res.ok) return handleError(res)
  await consumeStream(res, onChunk)
}

async function consumeStream(res: Response, onChunk: (chunk: string) => void): Promise<string | undefined> {
  if (!res.body) return
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ""
  let resolvedSessionId: string | undefined

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    const lines = buffer.split("\n")
    buffer = lines.pop() ?? ""

    for (const line of lines) {
      if (!line.startsWith("data: ")) continue
      const data = line.slice(6).trim()
      if (data === "[DONE]") return resolvedSessionId
      try {
        const parsed = JSON.parse(data)
        if (parsed.session_id) resolvedSessionId = parsed.session_id
        if (parsed.chunk) onChunk(parsed.chunk)
      } catch {
        // ignore parse errors
      }
    }
  }
  return resolvedSessionId
}
