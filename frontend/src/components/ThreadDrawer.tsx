"use client"

import { useEffect, useRef } from "react"
import { Message } from "@/types"
import { useThread } from "@/hooks/useThread"
import MessageList from "./MessageList"
import ChatInput from "./ChatInput"
import { Button } from "@/components/ui/button"
import { X } from "lucide-react"

interface Props {
  token: string
  parentMessage: Message
  onClose: () => void
}

export default function ThreadDrawer({ token, parentMessage, onClose }: Props) {
  const { messages, sending, streamingContent, error, sendMessage } = useThread(
    token,
    parentMessage.id
  )
  const drawerRef = useRef<HTMLDivElement>(null)

  // Close on Escape
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose()
    }
    document.addEventListener("keydown", onKey)
    return () => document.removeEventListener("keydown", onKey)
  }, [onClose])

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/20 z-40"
        onClick={onClose}
      />

      {/* Drawer */}
      <div
        ref={drawerRef}
        className="fixed right-0 top-0 h-full w-full max-w-md bg-background border-l shadow-xl z-50 flex flex-col"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b">
          <span className="font-semibold text-sm">Thread</span>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Parent message */}
        <div className="px-4 py-3 border-b bg-muted/50">
          <p className="text-xs text-muted-foreground mb-1">Replying to</p>
          <p className="text-sm text-foreground whitespace-pre-wrap line-clamp-5">
            {parentMessage.content}
          </p>
        </div>

        {/* Thread messages */}
        <div className="flex-1 overflow-y-auto px-4">
          {messages.length === 0 && !streamingContent && (
            <p className="text-sm text-muted-foreground text-center mt-8">
              Start a sub-thread by replying below.
            </p>
          )}
          <MessageList messages={messages} streamingContent={streamingContent} />
        </div>

        {error && (
          <p className="text-xs text-destructive px-4 pb-2">{error}</p>
        )}

        {/* Input */}
        <div className="px-4 py-3 border-t">
          <ChatInput
            onSend={sendMessage}
            disabled={sending}
            placeholder="Reply in thread..."
          />
        </div>
      </div>
    </>
  )
}
