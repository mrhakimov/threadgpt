import { API_URL, JSON_HEADERS, requestStream } from "@/data/http/client"

export async function sendChatMessage(
  userMessage: string,
  onChunk: (chunk: string) => void,
  sessionId?: string,
  forceNew = false,
  signal?: AbortSignal,
): Promise<string | undefined> {
  return requestStream(
    `${API_URL}/api/chat`,
    {
      method: "POST",
      headers: JSON_HEADERS,
      credentials: "include",
      signal,
      body: JSON.stringify({
        user_message: userMessage,
        session_id: sessionId ?? "",
        force_new: forceNew,
      }),
    },
    onChunk,
    signal,
  )
}
