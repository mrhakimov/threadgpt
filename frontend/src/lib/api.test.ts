import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  API_URL,
  checkAuth,
  sendChatMessage,
  sendThreadMessage,
} from "./api"

function createStreamResponse(chunks: string[]) {
  return new Response(
    new ReadableStream({
      start(controller) {
        for (const chunk of chunks) {
          controller.enqueue(new TextEncoder().encode(chunk))
        }
        controller.close()
      },
    }),
  )
}

describe("api", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it("falls back to persisted auth when auth check fails with a server error", async () => {
    localStorage.setItem("threadgpt_authed", "1")

    vi.spyOn(global, "fetch").mockResolvedValue(
      new Response(null, { status: 503 }),
    )

    await expect(checkAuth()).resolves.toBe(true)
    expect(fetch).toHaveBeenCalledWith(`${API_URL}/api/auth/check`, {
      credentials: "include",
    })
  })

  it("clears persisted auth when the server reports unauthorized", async () => {
    localStorage.setItem("threadgpt_authed", "1")

    vi.spyOn(global, "fetch").mockResolvedValue(
      new Response(null, { status: 401 }),
    )

    await expect(checkAuth()).resolves.toBe(false)
    expect(localStorage.getItem("threadgpt_authed")).toBeNull()
  })

  it("streams chat chunks and returns the resolved session id", async () => {
    vi.spyOn(global, "fetch").mockResolvedValue(
      createStreamResponse([
        'data: {"session_id":"session-123"}\n',
        'data: {"chunk":"Hello"}\n',
        'data: {"chunk":" world"}\n',
        "data: [DONE]\n",
      ]),
    )

    const onChunk = vi.fn()

    await expect(
      sendChatMessage("Hi", onChunk, "existing-session"),
    ).resolves.toBe("session-123")

    expect(onChunk).toHaveBeenNthCalledWith(1, "Hello")
    expect(onChunk).toHaveBeenNthCalledWith(2, " world")
    expect(fetch).toHaveBeenCalledWith(`${API_URL}/api/chat`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      signal: undefined,
      body: JSON.stringify({
        user_message: "Hi",
        session_id: "existing-session",
        force_new: false,
      }),
    })
  })

  it("streams thread replies until the stream completes", async () => {
    vi.spyOn(global, "fetch").mockResolvedValue(
      createStreamResponse([
        'data: {"chunk":"First"}\n',
        'data: {"chunk":" reply"}\n',
        "data: [DONE]\n",
      ]),
    )

    const onChunk = vi.fn()

    await expect(
      sendThreadMessage("parent-1", "Follow up", onChunk),
    ).resolves.toBeUndefined()

    expect(onChunk).toHaveBeenNthCalledWith(1, "First")
    expect(onChunk).toHaveBeenNthCalledWith(2, " reply")
  })
})
