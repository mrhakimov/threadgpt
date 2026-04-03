import type { HistoryPage, Session } from "@/domain/entities/chat"
import { MESSAGE_PAGE_SIZE, SESSION_PAGE_SIZE } from "@/domain/constants"
import { API_URL, JSON_HEADERS, requestJson, requestVoid } from "@/data/http/client"

export async function initSession(): Promise<Session> {
  return requestJson(`${API_URL}/api/session`, {
    method: "POST",
    headers: JSON_HEADERS,
    credentials: "include",
    body: JSON.stringify({}),
  })
}

export async function fetchSession(sessionId: string): Promise<Session> {
  return requestJson(`${API_URL}/api/sessions/${sessionId}`, {
    credentials: "include",
  })
}

export async function fetchSessions(
  limit = SESSION_PAGE_SIZE,
  offset = 0,
): Promise<{ sessions: Session[]; has_more: boolean }> {
  return requestJson(`${API_URL}/api/sessions?limit=${limit}&offset=${offset}`, {
    credentials: "include",
  })
}

export async function createSession(name: string): Promise<Session> {
  return requestJson(`${API_URL}/api/sessions`, {
    method: "POST",
    headers: JSON_HEADERS,
    credentials: "include",
    body: JSON.stringify({ name }),
  })
}

export async function renameSession(
  sessionId: string,
  name: string,
): Promise<void> {
  await requestVoid(`${API_URL}/api/sessions/${sessionId}`, {
    method: "PATCH",
    headers: JSON_HEADERS,
    credentials: "include",
    body: JSON.stringify({ name }),
  })
}

export async function updateSystemPrompt(
  sessionId: string,
  systemPrompt: string,
): Promise<void> {
  await requestVoid(`${API_URL}/api/sessions/${sessionId}`, {
    method: "PATCH",
    headers: JSON_HEADERS,
    credentials: "include",
    body: JSON.stringify({ system_prompt: systemPrompt }),
  })
}

export async function deleteSession(sessionId: string): Promise<void> {
  await requestVoid(`${API_URL}/api/sessions/${sessionId}`, {
    method: "DELETE",
    credentials: "include",
  })
}

export async function fetchHistory(
  sessionId?: string,
  limit = MESSAGE_PAGE_SIZE,
  offset = 0,
): Promise<HistoryPage> {
  const headers: Record<string, string> = {}
  if (sessionId) {
    headers["X-Session-ID"] = sessionId
  }

  return requestJson(`${API_URL}/api/history?limit=${limit}&offset=${offset}`, {
    headers,
    credentials: "include",
  })
}
