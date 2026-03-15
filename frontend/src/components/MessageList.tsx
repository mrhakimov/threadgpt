"use client"

import { useEffect, useRef, RefObject } from "react"
import { Message } from "@/types"
import MessageBubble from "./MessageBubble"
import { Loader2 } from "lucide-react"

interface Props {
  messages: Message[]
  streamingContent?: string
  sending?: boolean
  onReply?: (message: Message) => void
  onEditSystemPrompt?: (newContent: string) => Promise<void>
  scrollRef?: RefObject<HTMLDivElement | null>
  showSystemPrompt?: boolean
  hasMore?: boolean
  loadingMore?: boolean
  onLoadMore?: () => void
}

export default function MessageList({ messages, streamingContent, sending, onReply, onEditSystemPrompt, scrollRef, showSystemPrompt, hasMore, loadingMore, onLoadMore }: Props) {
  const didInitialScroll = useRef(false)
  const firstMessageIdRef = useRef<string | undefined>(undefined)

  useEffect(() => {
    const el = scrollRef?.current
    if (!el) return

    const firstId = messages[0]?.id
    const wasPrepend = firstId !== firstMessageIdRef.current && firstMessageIdRef.current !== undefined && messages.length > 0
    firstMessageIdRef.current = firstId

    // Don't auto-scroll to bottom when prepending older messages
    if (wasPrepend) return

    if (!didInitialScroll.current && messages.length > 0) {
      // On initial load, use instant scroll after a brief delay to let layout settle
      didInitialScroll.current = true
      let raf2: number
      const raf = requestAnimationFrame(() => {
        raf2 = requestAnimationFrame(() => {
          el.scrollTo({ top: el.scrollHeight, behavior: "instant" })
        })
      })
      return () => { cancelAnimationFrame(raf); cancelAnimationFrame(raf2) }
    }
    el.scrollTo({ top: el.scrollHeight, behavior: "smooth" })
  }, [messages.length, streamingContent, scrollRef])

  return (
    <div className="flex flex-col gap-4 py-4">
      {(hasMore || loadingMore) && (
        <div className="flex justify-center py-2">
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        </div>
      )}

      {messages.map((msg, i) => {
        // The context-set confirmation is the first assistant reply (index 1, no more pages above)
        const isContextSetConfirmation = msg.role === "assistant" && !hasMore && i === 1
        return (
          <MessageBubble key={msg.id} message={msg} onReply={isContextSetConfirmation ? undefined : onReply} isSystemPrompt={showSystemPrompt && !hasMore && i === 0 && msg.role === "user"} onEditSystemPrompt={showSystemPrompt && !hasMore && i === 0 && msg.role === "user" ? onEditSystemPrompt : undefined} />
        )
      })}

      {sending && !streamingContent && (
        <div className="flex gap-1 px-4 py-2">
          <span className="w-2 h-2 rounded-full bg-muted-foreground/50 animate-bounce [animation-delay:-0.3s]" />
          <span className="w-2 h-2 rounded-full bg-muted-foreground/50 animate-bounce [animation-delay:-0.15s]" />
          <span className="w-2 h-2 rounded-full bg-muted-foreground/50 animate-bounce" />
        </div>
      )}

      {streamingContent && (
        <MessageBubble
          message={{
            id: "__streaming__",
            session_id: "",
            role: "assistant",
            content: streamingContent,
            created_at: new Date().toISOString(),
          }}
          streaming
        />
      )}
    </div>
  )
}
