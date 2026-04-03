"use client"

import React from "react"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import ChatView from "./ChatView"
import type { Message } from "@/types"

const {
  useChat,
  useTheme,
  setStoredSidebarState,
  updateConversationSystemPrompt,
} = vi.hoisted(() => ({
  useChat: vi.fn(),
  useTheme: vi.fn(),
  setStoredSidebarState: vi.fn(),
  updateConversationSystemPrompt: vi.fn(),
}))

vi.mock("@/hooks/useChat", () => ({
  useChat,
}))

vi.mock("@/hooks/useTheme", () => ({
  useTheme,
}))

vi.mock("@/domain/constants", () => ({
  MIN_LOADING_MS: 0,
  SETTINGS_ANIMATION_MS: 0,
}))

vi.mock("@/services/sessionService", () => ({
  getStoredSidebarState: () => false,
  setStoredSidebarState,
  updateConversationSystemPrompt,
}))

vi.mock("@/services/chatService", () => ({
  getChatInputPlaceholder: () => "Message ThreadGPT",
  isFirstMessageSession: () => false,
}))

vi.mock("./ConversationMenu", () => ({
  default: () => <div data-testid="conversation-menu" />,
}))

vi.mock("./SettingsPage", () => ({
  default: () => <div data-testid="settings-page" />,
}))

vi.mock("@/components/chat/ChatComposer", () => ({
  default: () => <div data-testid="chat-composer" />,
}))

vi.mock("@/components/chat/ChatEmptyState", () => ({
  default: () => <div data-testid="chat-empty-state" />,
}))

vi.mock("@/components/chat/ChatHeader", () => ({
  default: ({ subtitle }: { subtitle?: string | null }) => <div data-testid="chat-header">{subtitle}</div>,
}))

vi.mock("@/components/chat/ScrollToBottomButton", () => ({
  default: () => null,
}))

vi.mock("@/components/shared/LoadingSpinner", () => ({
  default: () => <div data-testid="loading-spinner" />,
}))

vi.mock("./ThreadDrawer", () => ({
  default: () => <div data-testid="thread-drawer" />,
}))

vi.mock("./MessageList", () => ({
  default: ({ onInitialScrollComplete }: { onInitialScrollComplete?: () => void }) => (
    <div data-testid="message-list">
      <button type="button" data-testid="initial-scroll-complete" onClick={onInitialScrollComplete}>
        complete
      </button>
    </div>
  ),
}))

function createMessage(id: string): Message {
  return {
    id,
    session_id: "session-1",
    role: "assistant",
    content: id,
    created_at: new Date().toISOString(),
  }
}

describe("ChatView", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useTheme.mockReturnValue({ theme: "light", setTheme: vi.fn() })
    useChat.mockReturnValue({
      messages: [createMessage("m-1")],
      hasMoreMessages: true,
      loadingMore: false,
      session: { session_id: "session-1", name: "Thread" },
      loading: false,
      sending: false,
      streamingContent: "",
      error: null,
      sendMessage: vi.fn(),
      loadMoreMessages: vi.fn(),
      loadAllMessages: vi.fn(),
      updateLocalSystemPrompt: vi.fn(),
      incrementReplyCount: vi.fn(),
    })
  })

  it("does not load older messages before the initial bottom scroll completes", async () => {
    const loadMoreMessages = vi.fn()
    useChat.mockReturnValue({
      messages: [createMessage("m-1")],
      hasMoreMessages: true,
      loadingMore: false,
      session: { session_id: "session-1", name: "Thread" },
      loading: false,
      sending: false,
      streamingContent: "",
      error: null,
      sendMessage: vi.fn(),
      loadMoreMessages,
      loadAllMessages: vi.fn(),
      updateLocalSystemPrompt: vi.fn(),
      incrementReplyCount: vi.fn(),
    })

    const { container } = render(
      <ChatView sessionId="session-1" onSelectSession={vi.fn()} onUnauthorized={vi.fn()} />,
    )

    await waitFor(() => {
      expect(screen.getByTestId("message-list")).toBeInTheDocument()
    })

    const scrollEl = container.querySelector(".overflow-y-auto") as HTMLDivElement
    Object.defineProperty(scrollEl, "scrollHeight", { value: 1000, configurable: true })
    Object.defineProperty(scrollEl, "clientHeight", { value: 400, configurable: true })
    Object.defineProperty(scrollEl, "scrollTop", { value: 0, writable: true, configurable: true })

    fireEvent.scroll(scrollEl)

    expect(loadMoreMessages).not.toHaveBeenCalled()
  })

  it("loads older messages after the initial bottom scroll completes and the user reaches the top", async () => {
    const loadMoreMessages = vi.fn()
    useChat.mockReturnValue({
      messages: [createMessage("m-1")],
      hasMoreMessages: true,
      loadingMore: false,
      session: { session_id: "session-1", name: "Thread" },
      loading: false,
      sending: false,
      streamingContent: "",
      error: null,
      sendMessage: vi.fn(),
      loadMoreMessages,
      loadAllMessages: vi.fn(),
      updateLocalSystemPrompt: vi.fn(),
      incrementReplyCount: vi.fn(),
    })

    const { container } = render(
      <ChatView sessionId="session-1" onSelectSession={vi.fn()} onUnauthorized={vi.fn()} />,
    )

    await waitFor(() => {
      expect(screen.getByTestId("message-list")).toBeInTheDocument()
    })

    fireEvent.click(screen.getByTestId("initial-scroll-complete"))

    const scrollEl = container.querySelector(".overflow-y-auto") as HTMLDivElement
    Object.defineProperty(scrollEl, "scrollHeight", { value: 1000, configurable: true })
    Object.defineProperty(scrollEl, "clientHeight", { value: 400, configurable: true })
    Object.defineProperty(scrollEl, "scrollTop", { value: 0, writable: true, configurable: true })

    fireEvent.scroll(scrollEl)

    expect(loadMoreMessages).toHaveBeenCalledWith(scrollEl)
  })

  it("loads older messages when the user scrolls near the top", async () => {
    const loadMoreMessages = vi.fn()
    useChat.mockReturnValue({
      messages: [createMessage("m-1")],
      hasMoreMessages: true,
      loadingMore: false,
      session: { session_id: "session-1", name: "Thread" },
      loading: false,
      sending: false,
      streamingContent: "",
      error: null,
      sendMessage: vi.fn(),
      loadMoreMessages,
      loadAllMessages: vi.fn(),
      updateLocalSystemPrompt: vi.fn(),
      incrementReplyCount: vi.fn(),
    })

    const { container } = render(
      <ChatView sessionId="session-1" onSelectSession={vi.fn()} onUnauthorized={vi.fn()} />,
    )

    await waitFor(() => {
      expect(screen.getByTestId("message-list")).toBeInTheDocument()
    })

    fireEvent.click(screen.getByTestId("initial-scroll-complete"))

    const scrollEl = container.querySelector(".overflow-y-auto") as HTMLDivElement
    Object.defineProperty(scrollEl, "scrollHeight", { value: 1000, configurable: true })
    Object.defineProperty(scrollEl, "clientHeight", { value: 400, configurable: true })
    Object.defineProperty(scrollEl, "scrollTop", { value: 24, writable: true, configurable: true })

    fireEvent.scroll(scrollEl)

    expect(loadMoreMessages).toHaveBeenCalledWith(scrollEl)
  })

  it("does not auto-load older messages just because the viewport is taller than the latest page", async () => {
    const loadMoreMessages = vi.fn()
    useChat.mockReturnValue({
      messages: [createMessage("m-1")],
      hasMoreMessages: true,
      loadingMore: false,
      session: { session_id: "session-1", name: "Thread" },
      loading: false,
      sending: false,
      streamingContent: "",
      error: null,
      sendMessage: vi.fn(),
      loadMoreMessages,
      loadAllMessages: vi.fn(),
      updateLocalSystemPrompt: vi.fn(),
      incrementReplyCount: vi.fn(),
    })

    const { container } = render(
      <ChatView sessionId="session-1" onSelectSession={vi.fn()} onUnauthorized={vi.fn()} />,
    )

    await waitFor(() => {
      expect(screen.getByTestId("message-list")).toBeInTheDocument()
    })

    const scrollEl = container.querySelector(".overflow-y-auto") as HTMLDivElement
    Object.defineProperty(scrollEl, "scrollHeight", { value: 300, configurable: true })
    Object.defineProperty(scrollEl, "clientHeight", { value: 400, configurable: true })
    Object.defineProperty(scrollEl, "scrollTop", { value: 0, writable: true, configurable: true })

    fireEvent.click(screen.getByTestId("initial-scroll-complete"))

    await waitFor(() => {
      expect(loadMoreMessages).not.toHaveBeenCalled()
    })
  })
})
