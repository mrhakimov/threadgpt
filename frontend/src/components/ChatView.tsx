"use client"

import { useState, useRef, useCallback } from "react"
import { Message, Session } from "@/types"
import { useChat } from "@/hooks/useChat"
import MessageList from "./MessageList"
import ChatInput from "./ChatInput"
import ThreadDrawer from "./ThreadDrawer"
import ConversationMenu from "./ConversationMenu"
import { Button } from "@/components/ui/button"
import { ChevronDown, Settings } from "lucide-react"
import SettingsPage from "./SettingsPage"

interface Props {
  apiKey: string
  sessionId: string | null
  onSelectSession: (sessionId: string | null) => void
  onSessionCreated: (session: Session) => void
}

export default function ChatView({ apiKey, sessionId, onSelectSession, onSessionCreated }: Props) {
  const { messages, session, loading, sending, streamingContent, error, sendMessage } =
    useChat(apiKey, sessionId ?? undefined, (resolvedId) => {
      if (!sessionId) onSelectSession(resolvedId)
    })
  const [threadParent, setThreadParent] = useState<Message | null>(null)
  const [showScrollBtn, setShowScrollBtn] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    setShowScrollBtn(distFromBottom > 100)
  }, [])

  const scrollToBottom = useCallback(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" })
  }, [])

  if (loading) {
    return (
      <div className="h-screen flex items-center justify-center">
        <p className="text-muted-foreground text-sm">Loading...</p>
      </div>
    )
  }

  const isEmpty = messages.length === 0 && !streamingContent
  const isFirstMessage = !session?.assistant_id

  return (
    <div className="relative h-screen flex flex-col bg-background overflow-hidden">
      {/* Header */}
      <header className="shrink-0 border-b px-4 py-3 flex items-center gap-3">
        <ConversationMenu
          apiKey={apiKey}
          activeSessionId={sessionId}
          isCurrentEmpty={isEmpty}
          onSelectSession={onSelectSession}
          onSessionCreated={onSessionCreated}
        />
        <h1 className="font-semibold">ThreadGPT</h1>
        {session?.system_prompt && (
          <span className="text-xs text-muted-foreground truncate max-w-xs">
            {session.system_prompt}
          </span>
        )}
        <div className="ml-auto">
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
              onReply={setThreadParent}
              scrollRef={scrollRef}
            />
          )}
        </div>
      </div>

      {/* Scroll-to-bottom button */}
      {showScrollBtn && (
        <div className="absolute bottom-24 left-1/2 -translate-x-1/2 z-10">
          <Button
            size="sm"
            variant="secondary"
            className="rounded-full shadow-md h-8 px-3 gap-1 text-xs"
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
          apiKey={apiKey}
          parentMessage={threadParent}
          onClose={() => setThreadParent(null)}
        />
      )}

      {/* Settings */}
      {showSettings && (
        <SettingsPage apiKey={apiKey} onClose={() => setShowSettings(false)} />
      )}
    </div>
  )
}
