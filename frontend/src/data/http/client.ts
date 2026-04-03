import { toErrorMessage } from "@/domain/errors"

export const API_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8000"

export const JSON_HEADERS = { "Content-Type": "application/json" }

export async function handleResponseError(response: Response): Promise<never> {
  if (response.status >= 500) {
    throw new Error("Server error, please try again.")
  }

  if (response.status === 401 || response.status === 403) {
    throw new Error("Unauthorized. Please re-enter your API key.")
  }

  if (response.status === 404) {
    throw new Error("Not found.")
  }

  if (response.status === 429) {
    throw new Error("Too many requests. Please wait a moment.")
  }

  throw new Error("Something went wrong.")
}

export async function requestJson<T>(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(input, init)
  if (!response.ok) {
    return handleResponseError(response)
  }

  return response.json() as Promise<T>
}

export async function requestVoid(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<void> {
  const response = await fetch(input, init)
  if (!response.ok) {
    return handleResponseError(response)
  }
}

export async function requestStream(
  input: RequestInfo | URL,
  init: RequestInit,
  onChunk: (chunk: string) => void,
  signal?: AbortSignal,
): Promise<string | undefined> {
  const response = await fetch(input, init)
  if (!response.ok) {
    return handleResponseError(response)
  }

  return consumeStream(response, onChunk, signal)
}

async function consumeStream(
  response: Response,
  onChunk: (chunk: string) => void,
  signal?: AbortSignal,
): Promise<string | undefined> {
  if (!response.body) {
    return undefined
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ""
  let resolvedSessionId: string | undefined

  signal?.addEventListener("abort", () => {
    void reader.cancel(toErrorMessage(new DOMException("Aborted", "AbortError")))
  })

  while (true) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split("\n")
    buffer = lines.pop() ?? ""

    for (const line of lines) {
      if (!line.startsWith("data: ")) {
        continue
      }

      const data = line.slice(6).trim()
      if (data === "[DONE]") {
        return resolvedSessionId
      }

      try {
        const parsed = JSON.parse(data)
        if (parsed.session_id) {
          resolvedSessionId = parsed.session_id
        }
        if (parsed.chunk) {
          onChunk(parsed.chunk)
        }
      } catch {
        // Ignore malformed chunks and keep the stream alive.
      }
    }
  }

  return resolvedSessionId
}
