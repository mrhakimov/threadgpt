"use client"

import React from "react"
import { act, renderHook, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { useChat } from "./useChat"
import type { Message, Session } from "@/types"

const {
  loadChatSession,
  loadOlderChatMessages,
  loadCompleteChatHistory,
  sendChatTurn,
  buildOptimisticChatMessage,
  applySystemPromptLocally,
  incrementLocalReplyCount,
} = vi.hoisted(() => ({
  loadChatSession: vi.fn(),
  loadOlderChatMessages: vi.fn(),
  loadCompleteChatHistory: vi.fn(),
  sendChatTurn: vi.fn(),
  buildOptimisticChatMessage: vi.fn(),
  applySystemPromptLocally: vi.fn(),
  incrementLocalReplyCount: vi.fn(),
}))

vi.mock("@/services/chatService", () => ({
  loadChatSession,
  loadOlderChatMessages,
  loadCompleteChatHistory,
  sendChatTurn,
  buildOptimisticChatMessage,
  applySystemPromptLocally,
  incrementLocalReplyCount,
}))

function createMessage(
  overrides: Partial<Message> = {},
): Message {
  return {
    id: crypto.randomUUID(),
    session_id: "session-1",
    role: "assistant",
    content: "message",
    created_at: new Date().toISOString(),
    ...overrides,
  }
}

function createSession(
  overrides: Partial<Session> = {},
): Session {
  return {
    session_id: "session-1",
    is_new: false,
    ...overrides,
  }
}

describe("useChat", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    buildOptimisticChatMessage.mockImplementation((content: string, sessionId?: string) =>
      createMessage({
        session_id: sessionId ?? "",
        role: "user",
        content,
      }),
    )
    applySystemPromptLocally.mockImplementation((messages: Message[], content: string) =>
      messages.length > 0
        ? [{ ...messages[0], content }, ...messages.slice(1)]
        : messages,
    )
    incrementLocalReplyCount.mockImplementation((messages: Message[], messageId: string, by: number) =>
      messages.map((message) =>
        message.id === messageId
          ? { ...message, reply_count: (message.reply_count ?? 0) + by }
          : message,
      ),
    )
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("keeps a blank conversation empty when sessionId is null", async () => {
    loadChatSession.mockResolvedValue({
      messages: [],
      hasMoreMessages: false,
      session: null,
      loadedConversationCount: 0,
    })

    const { result } = renderHook(() => useChat(null))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(loadChatSession).toHaveBeenCalledWith(null)
    expect(result.current.messages).toEqual([])
    expect(result.current.session).toBeNull()
  })

  it("loads the latest session and resolves the detected session id", async () => {
    loadChatSession.mockResolvedValue({
      messages: [createMessage({ session_id: "latest-1", content: "Existing answer" })],
      hasMoreMessages: true,
      session: createSession({ session_id: "latest-1" }),
      resolvedSessionId: "latest-1",
      loadedConversationCount: 1,
    })

    const onSessionResolved = vi.fn()
    const { result } = renderHook(() => useChat(undefined, onSessionResolved))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(loadChatSession).toHaveBeenCalledWith(undefined)
    expect(result.current.messages).toHaveLength(1)
    expect(result.current.hasMoreMessages).toBe(true)
    expect(result.current.session?.session_id).toBe("latest-1")
    expect(onSessionResolved).toHaveBeenCalledWith("latest-1")
  })

  it("resolves a new session after the first streamed message and refreshes history", async () => {
    const messages = [
      createMessage({
        session_id: "new-session",
        role: "user",
        content: "Set the rules",
      }),
      createMessage({
        session_id: "new-session",
        role: "assistant",
        content: "Rules set",
      }),
    ]

    loadChatSession.mockResolvedValue({
      messages: [],
      hasMoreMessages: false,
      session: null,
      loadedConversationCount: 0,
    })
    sendChatTurn.mockImplementation(
      async ({
        onChunk,
      }: {
        onChunk: (chunk: string) => void
      }) => {
        onChunk("Rules")
        onChunk(" set")

        return {
          messages,
          hasMoreMessages: false,
          loadedConversationCount: 1,
          session: createSession({ session_id: "new-session" }),
          resolvedSessionId: "new-session",
        }
      },
    )

    const onSessionResolved = vi.fn()
    const { result } = renderHook(() => useChat(null, onSessionResolved))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    await act(async () => {
      await result.current.sendMessage("Set the rules")
    })

    expect(buildOptimisticChatMessage).not.toHaveBeenCalled()
    expect(sendChatTurn).toHaveBeenCalledWith({
      content: "Set the rules",
      requestedSessionId: null,
      currentSession: null,
      onChunk: expect.any(Function),
      signal: expect.any(AbortSignal),
    })
    expect(result.current.messages).toHaveLength(2)
    expect(result.current.streamingContent).toBe("")
    expect(result.current.session?.session_id).toBe("new-session")
    expect(onSessionResolved).toHaveBeenCalledWith("new-session")
  })

  it("scrolls to top without refetching when the full history is already loaded", async () => {
    loadChatSession.mockResolvedValue({
      messages: [createMessage({ session_id: "session-1", content: "Existing answer" })],
      hasMoreMessages: false,
      session: createSession({ session_id: "session-1" }),
      loadedConversationCount: 1,
    })

    const rafSpy = vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
      callback(0)
      return 1
    })

    const { result } = renderHook(() => useChat("session-1"))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    const scrollEl = document.createElement("div")
    scrollEl.scrollTop = 240

    await act(async () => {
      await result.current.loadAllMessages(scrollEl)
    })

    expect(loadCompleteChatHistory).not.toHaveBeenCalled()
    expect(scrollEl.scrollTop).toBe(0)

    rafSpy.mockRestore()
  })

})
