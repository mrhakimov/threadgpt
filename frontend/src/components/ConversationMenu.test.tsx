"use client"

import React from "react"
import { act, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import ConversationMenu from "./ConversationMenu"

const {
  listSessions,
  spinnerMock,
} = vi.hoisted(() => ({
  listSessions: vi.fn(),
  spinnerMock: vi.fn(({ className }: { className?: string }) => <div data-testid="loading-spinner" data-size={className} />),
}))

vi.mock("@/lib/constants", () => ({
  MIN_LOADING_MS: 1000,
}))

vi.mock("@/services/sessionService", () => ({
  deleteConversation: vi.fn(),
  getConversationLabel: (session: { name?: string | null }) => session.name ?? "Untitled",
  listSessions,
  renameConversation: vi.fn(),
}))

vi.mock("@/components/conversations/ConversationMenuHeader", () => ({
  default: () => <div data-testid="conversation-menu-header" />,
}))

vi.mock("@/components/conversations/NewConversationButton", () => ({
  default: () => <button type="button">New</button>,
}))

vi.mock("@/components/conversations/ConversationListItem", () => ({
  default: ({ label }: { label: string }) => <div>{label}</div>,
}))

vi.mock("@/components/shared/LoadingSpinner", () => ({
  default: (props: { className?: string }) => spinnerMock(props),
}))

describe("ConversationMenu", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.runOnlyPendingTimers()
    vi.useRealTimers()
  })

  it("shows only the main loading spinner during the initial session load", async () => {
    listSessions.mockResolvedValue({
      sessions: [{ session_id: "session-1", name: "Thread" }],
      has_more: true,
    })

    render(
      <ConversationMenu
        activeSessionId={null}
        collapsed={false}
        onToggle={vi.fn()}
        onSelectSession={vi.fn()}
      />,
    )

    await act(async () => {
      vi.advanceTimersByTime(200)
      await Promise.resolve()
    })

    expect(listSessions).toHaveBeenCalledTimes(1)
    expect(screen.getAllByTestId("loading-spinner")).toHaveLength(1)
  })

})
