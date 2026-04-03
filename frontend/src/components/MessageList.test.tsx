"use client"

import React from "react"
import { render, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import MessageList from "./MessageList"
import type { Message } from "@/types"

vi.mock("./MessageBubble", () => ({
  default: ({ message }: { message: Message }) => <div data-testid={`message-${message.id}`}>{message.content}</div>,
}))

vi.mock("@/components/shared/LoadingSpinner", () => ({
  default: () => <div data-testid="loading-spinner" />,
}))

function createMessage(id: string, content = id): Message {
  return {
    id,
    session_id: "session-1",
    role: "assistant",
    content,
    created_at: new Date().toISOString(),
  }
}

describe("MessageList", () => {
  beforeEach(() => {
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
      callback(0)
      return 1
    })
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("does not auto-scroll to bottom when the same history is refreshed", async () => {
    const scrollTo = vi.fn()
    const scrollEl = document.createElement("div")
    scrollEl.scrollTo = scrollTo
    Object.defineProperty(scrollEl, "scrollHeight", { value: 600, configurable: true })
    Object.defineProperty(scrollEl, "scrollTop", { value: 0, writable: true, configurable: true })

    const scrollRef = { current: scrollEl }
    const messages = [createMessage("m-1"), createMessage("m-2")]

    const { rerender } = render(<MessageList messages={messages} scrollRef={scrollRef} />)

    await waitFor(() => {
      expect(scrollEl.scrollTop).toBe(600)
    })

    scrollTo.mockClear()

    rerender(
      <MessageList
        messages={messages.map((message) => ({ ...message }))}
        scrollRef={scrollRef}
      />,
    )

    await waitFor(() => {
      expect(scrollTo).not.toHaveBeenCalled()
    })
  })

  it("keeps following the bottom when a new tail message arrives", async () => {
    const scrollTo = vi.fn()
    const scrollEl = document.createElement("div")
    scrollEl.scrollTo = scrollTo
    Object.defineProperty(scrollEl, "scrollHeight", { value: 900, configurable: true })
    Object.defineProperty(scrollEl, "scrollTop", { value: 0, writable: true, configurable: true })

    const scrollRef = { current: scrollEl }
    const messages = [createMessage("m-1"), createMessage("m-2")]

    const { rerender } = render(<MessageList messages={messages} scrollRef={scrollRef} />)

    await waitFor(() => {
      expect(scrollEl.scrollTop).toBe(900)
    })

    scrollTo.mockClear()

    rerender(
      <MessageList
        messages={[...messages, createMessage("m-3")]}
        scrollRef={scrollRef}
      />,
    )

    await waitFor(() => {
      expect(scrollTo).toHaveBeenCalledWith({ top: 900, behavior: "smooth" })
    })
  })

  it("re-snaps to the bottom when the scroll context changes", async () => {
    const scrollEl = document.createElement("div")
    scrollEl.scrollTo = vi.fn()
    Object.defineProperty(scrollEl, "scrollHeight", { value: 600, writable: true, configurable: true })
    Object.defineProperty(scrollEl, "scrollTop", { value: 0, writable: true, configurable: true })

    const scrollRef = { current: scrollEl }
    const firstMessages = [createMessage("m-1"), createMessage("m-2")]
    const nextMessages = [createMessage("m-10"), createMessage("m-11")]

    const { rerender } = render(
      <MessageList messages={firstMessages} scrollRef={scrollRef} scrollContextKey="session-1" />,
    )

    await waitFor(() => {
      expect(scrollEl.scrollTop).toBe(600)
    })

    scrollEl.scrollTop = 120
    Object.defineProperty(scrollEl, "scrollHeight", { value: 1200, writable: true, configurable: true })

    rerender(
      <MessageList messages={nextMessages} scrollRef={scrollRef} scrollContextKey="session-2" />,
    )

    await waitFor(() => {
      expect(scrollEl.scrollTop).toBe(1200)
    })
  })

  it("does not show the pagination spinner unless older messages are actively loading", () => {
    const scrollEl = document.createElement("div")
    scrollEl.scrollTo = vi.fn()
    Object.defineProperty(scrollEl, "scrollHeight", { value: 600, configurable: true })

    const scrollRef = { current: scrollEl }
    const messages = [createMessage("m-1"), createMessage("m-2")]

    const { rerender, queryByTestId } = render(
      <MessageList messages={messages} scrollRef={scrollRef} hasMore loadingMore={false} />,
    )

    expect(queryByTestId("loading-spinner")).not.toBeInTheDocument()

    rerender(
      <MessageList messages={messages} scrollRef={scrollRef} hasMore loadingMore />,
    )

    expect(queryByTestId("loading-spinner")).toBeInTheDocument()
  })
})
