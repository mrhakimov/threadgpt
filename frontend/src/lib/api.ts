const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8000"

async function handleError(res: Response): Promise<never> {
  if (res.status >= 500) throw new Error("Server error, please try again.")
  if (res.status === 401 || res.status === 403) throw new Error("Unauthorized. Please re-enter your API key.")
  if (res.status === 404) throw new Error("Not found.")
  if (res.status === 429) throw new Error("Too many requests. Please wait a moment.")
  throw new Error("Something went wrong.")
}

function authHeaders(token: string) {
  return {
    "Content-Type": "application/json",
    "Authorization": `Bearer ${token}`,
  }
}

export async function auth(apiKey: string): Promise<{ token: string }> {
  const res = await fetch(`${API_URL}/api/auth`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ api_key: apiKey }),
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function initSession(token: string) {
  const res = await fetch(`${API_URL}/api/session`, {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify({}),
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function fetchSession(sessionId: string, token: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`, {
    headers: authHeaders(token),
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function fetchSessions(token: string, limit = 20, offset = 0): Promise<{ sessions: import("@/types").Session[], has_more: boolean }> {
  const res = await fetch(`${API_URL}/api/sessions?limit=${limit}&offset=${offset}`, {
    headers: authHeaders(token),
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function createSession(token: string, name: string) {
  const res = await fetch(`${API_URL}/api/sessions`, {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify({ name }),
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function renameSession(token: string, sessionId: string, name: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`, {
    method: "PATCH",
    headers: authHeaders(token),
    body: JSON.stringify({ name }),
  })
  if (!res.ok) return handleError(res)
}

export async function updateSystemPrompt(sessionId: string, systemPrompt: string, token: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`, {
    method: "PATCH",
    headers: authHeaders(token),
    body: JSON.stringify({ system_prompt: systemPrompt }),
  })
  if (!res.ok) return handleError(res)
}

export async function deleteSession(token: string, sessionId: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`, {
    method: "DELETE",
    headers: authHeaders(token),
  })
  if (!res.ok) return handleError(res)
}

export async function fetchHistory(token: string, sessionId?: string, limit = 50, offset = 0): Promise<{ messages: import("@/types").Message[], has_more: boolean }> {
  const headers: Record<string, string> = authHeaders(token)
  if (sessionId) headers["X-Session-ID"] = sessionId
  const res = await fetch(`${API_URL}/api/history?limit=${limit}&offset=${offset}`, { headers })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function sendChatMessage(
  token: string,
  userMessage: string,
  onChunk: (chunk: string) => void,
  sessionId?: string,
  forceNew?: boolean
): Promise<string | undefined> {
  const res = await fetch(`${API_URL}/api/chat`, {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify({ user_message: userMessage, session_id: sessionId ?? "", force_new: forceNew ?? false }),
  })
  if (!res.ok) return handleError(res)
  return consumeStream(res, onChunk)
}

export async function fetchThreadMessages(token: string, parentMessageId: string, limit = 50, offset = 0): Promise<{ messages: import("@/types").Message[], has_more: boolean }> {
  const res = await fetch(`${API_URL}/api/thread?parent_message_id=${encodeURIComponent(parentMessageId)}&limit=${limit}&offset=${offset}`, {
    headers: authHeaders(token),
  })
  if (!res.ok) return handleError(res)
  return res.json()
}

export async function sendThreadMessage(
  token: string,
  parentMessageId: string,
  userMessage: string,
  onChunk: (chunk: string) => void
): Promise<void> {
  const res = await fetch(`${API_URL}/api/thread`, {
    method: "POST",
    headers: authHeaders(token),
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
