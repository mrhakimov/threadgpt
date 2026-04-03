"use client"

import { useState, useRef, useCallback, useEffect } from "react"
import { Message } from "@/types"
import { useChat } from "@/hooks/useChat"
import { useTheme } from "@/hooks/useTheme"
import MessageList from "./MessageList"
import ThreadDrawer from "./ThreadDrawer"
import ConversationMenu from "./ConversationMenu"
import SettingsPage from "./SettingsPage"
import ChatComposer from "@/components/chat/ChatComposer"
import ChatEmptyState from "@/components/chat/ChatEmptyState"
import ChatHeader from "@/components/chat/ChatHeader"
import ScrollToBottomButton from "@/components/chat/ScrollToBottomButton"
import LoadingSpinner from "@/components/shared/LoadingSpinner"
import { MIN_LOADING_MS, SETTINGS_ANIMATION_MS } from "@/domain/constants"
import {
  getStoredSidebarState,
  setStoredSidebarState,
  updateConversationSystemPrompt,
} from "@/services/sessionService"
import {
  getChatInputPlaceholder,
  isFirstMessageSession,
} from "@/services/chatService"

interface Props {
  sessionId: string | null | undefined
  onSelectSession: (sessionId: string | null) => void
  onUnauthorized: () => void
}

const LOAD_MORE_TOP_THRESHOLD = 200

export default function ChatView({ sessionId, onSelectSession, onUnauthorized }: Props) {
  const { theme, setTheme } = useTheme()
  const [sidebarRefreshTrigger, setSidebarRefreshTrigger] = useState(0)
  const { messages, hasMoreMessages, loadingMore, session, loading, sending, streamingContent, error, sendMessage, loadMoreMessages, loadAllMessages, updateLocalSystemPrompt, incrementReplyCount } =
    useChat(sessionId, (resolvedId) => {
      if (!sessionId) onSelectSession(resolvedId)
      setSidebarRefreshTrigger((n) => n + 1)
    }, onUnauthorized)
  const [threadParent, setThreadParent] = useState<Message | null>(null)
  const [showLoading, setShowLoading] = useState(true)
  const loadStartRef = useRef(Date.now())
  const [showScrollBtn, setShowScrollBtn] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const [closingSettings, setClosingSettings] = useState(false)
  const [focusTrigger, setFocusTrigger] = useState(0)
  const [overrideName, setOverrideName] = useState<string | null>(null)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(getStoredSidebarState)
  const [canLoadMoreOnScroll, setCanLoadMoreOnScroll] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const threadAbortRef = useRef<(() => void) | null>(null)
  const prevSessionIdRef = useRef<string | null | undefined>(sessionId)
  const settingsCloseTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const handleSelectSession = useCallback((id: string | null) => {
    threadAbortRef.current?.()
    if (id === null) setThreadParent(null)
    onSelectSession(id)
  }, [onSelectSession])

  useEffect(() => {
    const prev = prevSessionIdRef.current
    prevSessionIdRef.current = sessionId

    // When a new session resolves (null → ID), don't reset the loading state —
    // messages are already displayed from sendMessage.
    const isSessionResolution = prev === null && typeof sessionId === "string"
    if (isSessionResolution) return

    threadAbortRef.current?.()
    setThreadParent(null)
    setFocusTrigger((n) => n + 1)
    setOverrideName(null)
    setCanLoadMoreOnScroll(false)
    setShowLoading(true)
    loadStartRef.current = Date.now()
  }, [sessionId])

  useEffect(() => {
    if (!loading) {
      const elapsed = Date.now() - loadStartRef.current
      const remaining = MIN_LOADING_MS - elapsed
      if (remaining > 0) {
        const t = setTimeout(() => setShowLoading(false), remaining)
        return () => clearTimeout(t)
      }
      setShowLoading(false)
    }
  }, [loading, sessionId])

  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    setShowScrollBtn(distFromBottom > 100)
    if (canLoadMoreOnScroll && el.scrollTop <= LOAD_MORE_TOP_THRESHOLD && hasMoreMessages && !loadingMore) {
      loadMoreMessages(el)
    }
  }, [canLoadMoreOnScroll, hasMoreMessages, loadingMore, loadMoreMessages])

  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    setShowScrollBtn(distFromBottom > 100)
  }, [messages, streamingContent])

  const scrollToBottom = useCallback(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" })
    setFocusTrigger((n) => n + 1)
  }, [])

  const handleOpenSettings = useCallback(() => {
    if (settingsCloseTimerRef.current) {
      clearTimeout(settingsCloseTimerRef.current)
      settingsCloseTimerRef.current = null
    }
    setClosingSettings(false)
    setShowSettings(true)
  }, [])

  const handleCloseSettings = useCallback(() => {
    if (closingSettings) return
    setClosingSettings(true)
    settingsCloseTimerRef.current = setTimeout(() => {
      setShowSettings(false)
      setClosingSettings(false)
      settingsCloseTimerRef.current = null
    }, SETTINGS_ANIMATION_MS)
  }, [closingSettings])

  useEffect(() => {
    return () => {
      if (settingsCloseTimerRef.current) clearTimeout(settingsCloseTimerRef.current)
    }
  }, [])

  const isEmpty = messages.length === 0 && !streamingContent
  const isFirstMessage = isFirstMessageSession(session)
  const subtitle =
    overrideName ??
    (session?.name && session.name !== "New conversation"
      ? session.name
      : session?.system_prompt ?? null)

  const handleToggleSidebar = useCallback(() => {
    setSidebarCollapsed((current) => {
      const next = !current
      setStoredSidebarState(next)
      return next
    })
  }, [])

  return (
    <div className="relative h-screen flex bg-background overflow-hidden">
      <ConversationMenu
        activeSessionId={sessionId ?? null}
        isCurrentEmpty={isEmpty}
        collapsed={sidebarCollapsed}
        onToggle={handleToggleSidebar}
        onSelectSession={handleSelectSession}
        onRequestFocusCurrentInput={() => setFocusTrigger((n) => n + 1)}
        onRenameActive={(name) => setOverrideName(name)}
        refreshTrigger={sidebarRefreshTrigger}
      />

      <div className="flex-1 flex flex-col min-w-0 relative overflow-hidden">
        <ChatHeader
          title="ThreadGPT"
          subtitle={subtitle}
          onTitleClick={() => {
            loadAllMessages(scrollRef.current)
            setFocusTrigger((n) => n + 1)
          }}
          onOpenSettings={handleOpenSettings}
          settingsOpen={showSettings}
          settingsClosing={closingSettings}
        />

        <div ref={scrollRef} onScroll={handleScroll} className="flex-1 overflow-y-auto px-4 relative">
          {showLoading ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <LoadingSpinner className="h-6 w-6" />
            </div>
          ) : (
            <div className={isEmpty ? "max-w-3xl mx-auto w-full h-full" : "max-w-3xl mx-auto w-full min-h-full flex flex-col"}>
              {isEmpty ? (
                <ChatEmptyState isFirstMessage={isFirstMessage} />
              ) : (
                <MessageList
                  messages={messages}
                  streamingContent={streamingContent}
                  sending={sending}
                  onReply={setThreadParent}
                  onEditSystemPrompt={session?.session_id ? async (content) => {
                    await updateConversationSystemPrompt(session.session_id!, content)
                    updateLocalSystemPrompt(content)
                  } : undefined}
                  scrollRef={scrollRef}
                  scrollContextKey={session?.session_id ?? sessionId ?? "new-conversation"}
                  onInitialScrollComplete={() => setCanLoadMoreOnScroll(true)}
                  showSystemPrompt
                  hasMore={hasMoreMessages}
                  loadingMore={loadingMore}
                  onLoadMore={loadMoreMessages}
                />
              )}
            </div>
          )}
        </div>

        <ScrollToBottomButton visible={showScrollBtn} onClick={scrollToBottom} />

        <ChatComposer
          error={error}
          onSend={sendMessage}
          disabled={sending}
          focusTrigger={focusTrigger}
          placeholder={getChatInputPlaceholder(isFirstMessage)}
        />

        {threadParent && (
          <ThreadDrawer
            parentMessage={threadParent}
            onClose={() => { threadAbortRef.current?.(); setThreadParent(null); setFocusTrigger((n) => n + 1) }}
            onReply={(parentId) => incrementReplyCount(parentId, 1)}
            onAbortRef={(fn) => { threadAbortRef.current = fn }}
          />
        )}

        {showSettings && (
          <SettingsPage closing={closingSettings} onClose={handleCloseSettings} onLogout={onUnauthorized} theme={theme} setTheme={setTheme} />
        )}
      </div>
    </div>
  )
}
