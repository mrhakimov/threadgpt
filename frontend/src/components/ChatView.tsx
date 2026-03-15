"use client"

import { useState, useRef, useCallback, useEffect } from "react"
import { Message } from "@/types"
import { useChat } from "@/hooks/useChat"
import MessageList from "./MessageList"
import ChatInput from "./ChatInput"
import ThreadDrawer from "./ThreadDrawer"
import ConversationMenu from "./ConversationMenu"
import { Button } from "@/components/ui/button"
import { ChevronDown, Loader2, Settings } from "lucide-react"
import SettingsPage from "./SettingsPage"
import { updateSystemPrompt } from "@/lib/api"

interface Props {
  sessionId: string | null | undefined
  onSelectSession: (sessionId: string | null) => void
  onUnauthorized: () => void
}

export default function ChatView({ sessionId, onSelectSession, onUnauthorized }: Props) {
  const { messages, hasMoreMessages, loadingMore, session, loading, sending, streamingContent, error, sendMessage, loadMoreMessages, loadAllMessages, updateLocalSystemPrompt, incrementReplyCount } =
    useChat(sessionId, (resolvedId) => {
      if (!sessionId) onSelectSession(resolvedId)
    }, onUnauthorized)
  const [threadParent, setThreadParent] = useState<Message | null>(null)
  const [showScrollBtn, setShowScrollBtn] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const [focusTrigger, setFocusTrigger] = useState(0)
  const [overrideName, setOverrideName] = useState<string | null>(null)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    if (typeof window === "undefined") return false
    return localStorage.getItem("sidebarCollapsed") === "true"
  })
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setFocusTrigger((n) => n + 1)
    setOverrideName(null)
  }, [sessionId])

  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    setShowScrollBtn(distFromBottom > 100)
    if (el.scrollTop === 0 && hasMoreMessages && !loadingMore) {
      loadMoreMessages(el)
    }
  }, [hasMoreMessages, loadingMore, loadMoreMessages])

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

  if (loading) {
    return (
      <div className="h-screen flex items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const isEmpty = messages.length === 0 && !streamingContent
  const isFirstMessage = !session?.assistant_id

  return (
    <div className="relative h-screen flex bg-background overflow-hidden">
      {/* Sidebar */}
      <ConversationMenu
        activeSessionId={sessionId ?? null}
        isCurrentEmpty={isEmpty}
        collapsed={sidebarCollapsed}
        onToggle={() => setSidebarCollapsed((v) => { const next = !v; localStorage.setItem("sidebarCollapsed", String(next)); return next })}
        onSelectSession={onSelectSession}
        onRenameActive={(name) => setOverrideName(name)}
      />

      {/* Main area */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Header */}
        <header className="shrink-0 border-b px-4 py-3 flex items-center gap-3">
          <h1 className="font-semibold">ThreadGPT</h1>
          {(overrideName || session?.name || session?.system_prompt) && (
            <button
              onClick={() => {
                loadAllMessages(scrollRef.current)
                setFocusTrigger((n) => n + 1)
              }}
              className="text-xs text-muted-foreground truncate max-w-xs hover:text-foreground transition-colors"
            >
              {overrideName ?? (session!.name && session!.name !== "New conversation" ? session!.name : session!.system_prompt)}
            </button>
          )}
          <div className="ml-auto flex items-center gap-1">
            <a
              href="https://x.com/omtiness"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center h-9 w-9 rounded-md hover:bg-accent hover:text-accent-foreground transition-colors"
              aria-label="X (Twitter)"
            >
              <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-4.714-6.231-5.401 6.231H2.744l7.737-8.835L2.25 2.25h6.902l4.263 5.634zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
              </svg>
            </a>
            <a
              href="https://github.com/mrhakimov/threadgpt"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center h-9 w-9 rounded-md hover:bg-accent hover:text-accent-foreground transition-colors"
              aria-label="GitHub"
            >
              <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M12 2C6.477 2 2 6.477 2 12c0 4.418 2.865 8.166 6.839 9.489.5.092.682-.217.682-.482 0-.237-.009-.868-.013-1.703-2.782.604-3.369-1.342-3.369-1.342-.454-1.155-1.11-1.463-1.11-1.463-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0 1 12 6.836a9.59 9.59 0 0 1 2.504.337c1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.741 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
              </svg>
            </a>
            <Button variant="ghost" size="icon" onClick={() => setShowSettings(true)}>
              <Settings className="h-4 w-4" />
            </Button>
          </div>
        </header>

        {/* Messages */}
        <div ref={scrollRef} onScroll={handleScroll} className="flex-1 overflow-y-auto px-4 relative">
          <div className="max-w-3xl mx-auto w-full h-full">
            {isEmpty ? (
              <div className="flex flex-col items-center justify-center h-full gap-3 text-center px-4">
                <h2 className="text-lg font-medium">
                  {isFirstMessage ? "Set your conversation context" : "Start chatting"}
                </h2>
                <p className="text-sm text-muted-foreground max-w-sm">
                  {isFirstMessage
                    ? "Your first message becomes the assistant's instructions for this entire conversation. Make it count!"
                    : "Send a message to continue your conversation."}
                </p>
              </div>
            ) : (
              <MessageList
                messages={messages}
                streamingContent={streamingContent}
                sending={sending}
                onReply={setThreadParent}
                onEditSystemPrompt={session?.session_id ? async (content) => {
                  await updateSystemPrompt(session.session_id!, content)
                  updateLocalSystemPrompt(content)
                } : undefined}
                scrollRef={scrollRef}
                showSystemPrompt
                hasMore={hasMoreMessages}
                loadingMore={loadingMore}
                onLoadMore={loadMoreMessages}
              />
            )}
          </div>
        </div>

        {/* Scroll-to-bottom button */}
        {showScrollBtn && (
          <div className="absolute bottom-24 left-1/2 -translate-x-1/2 z-10">
            <Button
              size="sm"
              className="rounded-full shadow-lg h-8 px-3 gap-1 text-xs bg-background text-foreground border border-border hover:bg-muted"
              onClick={scrollToBottom}
            >
              <ChevronDown className="h-3.5 w-3.5" />
              Scroll to bottom
            </Button>
          </div>
        )}

        {/* Input */}
        <div className="shrink-0 border-t px-4 py-3">
          <div className="max-w-3xl mx-auto w-full">
            {error && (
              <p className="text-xs text-destructive mb-2">{error}</p>
            )}
            <ChatInput
              onSend={sendMessage}
              disabled={sending}
              focusTrigger={focusTrigger}
              placeholder={
                isFirstMessage
                  ? "Set the context for your conversation..."
                  : "Send a message..."
              }
            />
          </div>
        </div>

        {/* Thread Drawer */}
        {threadParent && (
          <ThreadDrawer
            parentMessage={threadParent}
            onClose={() => setThreadParent(null)}
            onReply={(parentId) => incrementReplyCount(parentId, 1)}
          />
        )}

        {/* Settings */}
        {showSettings && (
          <SettingsPage onClose={() => setShowSettings(false)} onLogout={onUnauthorized} />
        )}
      </div>{/* end main area */}
    </div>
  )
}
