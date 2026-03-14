"use client"

import { useEffect, RefObject } from "react"
import { Message } from "@/types"
import MessageBubble from "./MessageBubble"

interface Props {
  messages: Message[]
  streamingContent?: string
  sending?: boolean
  onReply?: (message: Message) => void
  onEditSystemPrompt?: (newContent: string) => Promise<void>
  scrollRef?: RefObject<HTMLDivElement | null>
}

export default function MessageList({ messages, streamingContent, sending, onReply, onEditSystemPrompt, scrollRef }: Props) {
  useEffect(() => {
    const el = scrollRef?.current
    if (!el) return
    const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    // Only auto-scroll if user is already near the bottom (within 200px)
    if (distFromBottom <= 200) {
      el.scrollTo({ top: el.scrollHeight, behavior: "smooth" })
    }
  }, [messages, streamingContent, scrollRef])

  return (
    <div className="flex flex-col gap-4 py-4">
      {messages.map((msg, i) => (
        <MessageBubble key={msg.id} message={msg} onReply={onReply} isSystemPrompt={i === 0 && msg.role === "user"} onEditSystemPrompt={i === 0 && msg.role === "user" ? onEditSystemPrompt : undefined} />
      ))}

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
