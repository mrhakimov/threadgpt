import { ApiError, getApiErrorPayload, toErrorMessage } from "@/domain/errors"

export const API_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8000"

export const JSON_HEADERS = { "Content-Type": "application/json" }

export async function handleResponseError(response: Response): Promise<never> {
  throw await buildResponseError(response)
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

      let parsed: unknown
      try {
        parsed = JSON.parse(data)
      } catch {
        // Ignore malformed chunks and keep the stream alive.
        continue
      }

      const apiError = getApiErrorPayload(parsed)
      if (apiError) {
        throw new ApiError(apiError.message, {
          code: apiError.code,
          status: apiError.status,
        })
      }
      if (typeof parsed === "object" && parsed !== null && "session_id" in parsed) {
        const sessionId = parsed.session_id
        if (typeof sessionId === "string" && sessionId) {
          resolvedSessionId = sessionId
        }
      }
      if (typeof parsed === "object" && parsed !== null && "chunk" in parsed) {
        const chunk = parsed.chunk
        if (typeof chunk === "string" && chunk) {
          onChunk(chunk)
        }
      }
    }
  }

  return resolvedSessionId
}

async function buildResponseError(response: Response): Promise<ApiError> {
  const payload = await response.clone().json().catch(() => null)
  const apiError = getApiErrorPayload(payload)
  if (apiError) {
    return new ApiError(apiError.message, {
      code: apiError.code,
      status: apiError.status ?? response.status,
    })
  }

  const text = (await response.text().catch(() => "")).trim()
  if (text) {
    return new ApiError(text, { status: response.status })
  }

  return new ApiError(defaultErrorMessage(response.status), {
    status: response.status,
  })
}

function defaultErrorMessage(status: number): string {
  if (status >= 500) {
    return "Something went wrong on the server. Please try again."
  }
  if (status === 401 || status === 403) {
    return "Please sign in again."
  }
  if (status === 404) {
    return "That resource was not found."
  }
  if (status === 429) {
    return "Too many requests. Please wait a moment and try again."
  }
  return "Something went wrong."
}
