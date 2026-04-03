import type { HistoryPage } from "@/domain/entities/chat"
import { MESSAGE_PAGE_SIZE } from "@/domain/constants"
import { API_URL, JSON_HEADERS, requestJson, requestStream } from "@/data/http/client"

export async function fetchThreadMessages(
  parentMessageId: string,
  limit = MESSAGE_PAGE_SIZE,
  offset = 0,
): Promise<HistoryPage> {
  return requestJson(
    `${API_URL}/api/thread?parent_message_id=${encodeURIComponent(parentMessageId)}&limit=${limit}&offset=${offset}`,
    {
      credentials: "include",
    },
  )
}

export async function sendThreadMessage(
  parentMessageId: string,
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
        parent_message_id: parentMessageId,
        user_message: userMessage,
      }),
    },
    onChunk,
    signal,
  )
}
