"use client"

import React from "react"
import { act, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import ThreadDrawer from "./ThreadDrawer"
import type { Message } from "@/types"

const { useThread, messageListMock } = vi.hoisted(() => ({
  useThread: vi.fn(),
  messageListMock: vi.fn((_props?: unknown) => <div data-testid="message-list" />),
}))

vi.mock("@/hooks/useThread", () => ({
  useThread,
}))

vi.mock("./MessageList", () => ({
  default: (props: unknown) => messageListMock(props),
}))

vi.mock("./ChatInput", () => ({
  default: () => <div data-testid="chat-input" />,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>{children}</button>
  ),
}))

vi.mock("@/components/shared/LoadingSpinner", () => ({
  default: () => <div data-testid="loading-spinner" />,
}))

vi.mock("@/lib/constants", () => ({
  MIN_LOADING_MS: 0,
}))

vi.mock("lucide-react", () => ({
  X: () => <span data-testid="close-icon" />,
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

describe("ThreadDrawer", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    useThread.mockReturnValue({
      messages: [],
      hasMore: false,
      loadingMore: false,
      loading: false,
      sending: false,
      streamingContent: "",
      error: null,
      sendMessage: vi.fn(),
      loadMore: vi.fn(),
      abort: vi.fn(),
    })
  })

  afterEach(() => {
    vi.runOnlyPendingTimers()
    vi.useRealTimers()
  })

  it("keeps the empty state visible without a scrollable body", () => {
    render(
      <ThreadDrawer
        parentMessage={createMessage("parent", "Parent message")}
        onClose={vi.fn()}
      />,
    )

    act(() => {
      vi.runAllTimers()
    })

    expect(screen.getByText("Ask a follow-up question below.")).toBeInTheDocument()
    expect(document.body.querySelector(".overflow-hidden")).toBeTruthy()
    expect(document.body.querySelector(".overflow-y-auto")).toBeNull()
    expect(screen.queryByTestId("message-list")).not.toBeInTheDocument()
  })

  it("renders subthread messages top-aligned and opens them at the bottom", () => {
    useThread.mockReturnValue({
      messages: [createMessage("m-1"), createMessage("m-2")],
      hasMore: false,
      loadingMore: false,
      loading: false,
      sending: false,
      streamingContent: "",
      error: null,
      sendMessage: vi.fn(),
      loadMore: vi.fn(),
      abort: vi.fn(),
    })

    render(
      <ThreadDrawer
        parentMessage={createMessage("parent", "Parent message")}
        onClose={vi.fn()}
      />,
    )

    act(() => {
      vi.runAllTimers()
    })

    expect(messageListMock).toHaveBeenCalledWith(
      expect.objectContaining({
        contentAlignment: "top",
        initialScrollPosition: "bottom",
      }),
    )
  })

  it("renders the following-up section inside its own full-width divider", () => {
    render(
      <ThreadDrawer
        parentMessage={createMessage("parent", "Parent message")}
        onClose={vi.fn()}
      />,
    )

    act(() => {
      vi.runAllTimers()
    })

    const followingUpLabel = screen.getByText("Following up on")
    const dividerSection = followingUpLabel.closest(".border-b")

    expect(dividerSection).toBeTruthy()
    expect(dividerSection).not.toHaveClass("px-4")
    expect(dividerSection?.firstElementChild).toHaveClass("px-4", "py-3")
  })
})
