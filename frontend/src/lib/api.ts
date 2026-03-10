const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8000"

export async function initSession(apiKey: string) {
  const res = await fetch(`${API_URL}/api/session`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ api_key: apiKey }),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function fetchSession(sessionId: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function fetchSessions(apiKey: string) {
  const hash = await sha256(apiKey)
  const res = await fetch(`${API_URL}/api/sessions?api_key_hash=${hash}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function createSession(apiKey: string, name: string) {
  const res = await fetch(`${API_URL}/api/sessions`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ api_key: apiKey, name }),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function renameSession(sessionId: string, name: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  })
  if (!res.ok) throw new Error(await res.text())
}

export async function deleteSession(sessionId: string) {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}`, {
    method: "DELETE",
  })
  if (!res.ok) throw new Error(await res.text())
}

export async function fetchHistory(apiKey: string, sessionId?: string) {
  if (sessionId) {
    const res = await fetch(`${API_URL}/api/history?session_id=${sessionId}`)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  }
  const hash = await sha256(apiKey)
  const res = await fetch(`${API_URL}/api/history?api_key_hash=${hash}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function sendChatMessage(
  apiKey: string,
  userMessage: string,
  onChunk: (chunk: string) => void,
  sessionId?: string,
  forceNew?: boolean
): Promise<string | undefined> {
  const res = await fetch(`${API_URL}/api/chat`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ api_key: apiKey, user_message: userMessage, session_id: sessionId ?? "", force_new: forceNew ?? false }),
  })
  if (!res.ok) throw new Error(await res.text())
  return consumeStream(res, onChunk)
}

export async function sendThreadMessage(
  apiKey: string,
  parentMessageId: string,
  userMessage: string,
  onChunk: (chunk: string) => void
): Promise<void> {
  const res = await fetch(`${API_URL}/api/thread`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      api_key: apiKey,
      parent_message_id: parentMessageId,
      user_message: userMessage,
    }),
  })
  if (!res.ok) throw new Error(await res.text())
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

export async function sha256(text: string): Promise<string> {
  const msgBuffer = new TextEncoder().encode(text)
  const hashBuffer = await crypto.subtle.digest("SHA-256", msgBuffer)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("")
}
