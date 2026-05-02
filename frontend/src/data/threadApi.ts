import type { HistoryPage } from "@/domain/entities/chat"
import { MESSAGE_PAGE_SIZE } from "@/domain/constants"
import { API_URL, JSON_HEADERS, requestJson, requestStream } from "@/data/http/client"

export async function fetchThreadMessages(
  conversationId: string,
  limit = MESSAGE_PAGE_SIZE,
  offset = 0,
): Promise<HistoryPage> {
  return requestJson(
    `${API_URL}/api/thread?conversation_id=${encodeURIComponent(conversationId)}&limit=${limit}&offset=${offset}`,
    {
      credentials: "include",
    },
  )
}

export async function sendThreadMessage(
  conversationId: string,
  userMessage: string,
  onChunk: (chunk: string) => void,
  signal?: AbortSignal,
): Promise<void> {
  await requestStream(
    `${API_URL}/api/thread`,
    {
      method: "POST",
      headers: JSON_HEADERS,
      credentials: "include",
      signal,
      body: JSON.stringify({
        conversation_id: conversationId,
        user_message: userMessage,
      }),
    },
    onChunk,
    signal,
  )
}
