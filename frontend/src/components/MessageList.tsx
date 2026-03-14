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
  showSystemPrompt?: boolean
}

export default function MessageList({ messages, streamingContent, sending, onReply, onEditSystemPrompt, scrollRef, showSystemPrompt }: Props) {
  useEffect(() => {
    const el = scrollRef?.current
    if (!el) return
    el.scrollTo({ top: el.scrollHeight, behavior: "smooth" })
  }, [messages.length, streamingContent, scrollRef])

  return (
    <div className="flex flex-col gap-4 py-4">
      {messages.map((msg, i) => {
        const isContextSetConfirmation = msg.role === "assistant" && !msg.openai_thread_id
        return (
          <MessageBubble key={msg.id} message={msg} onReply={isContextSetConfirmation ? undefined : onReply} isSystemPrompt={showSystemPrompt && i === 0 && msg.role === "user"} onEditSystemPrompt={showSystemPrompt && i === 0 && msg.role === "user" ? onEditSystemPrompt : undefined} />
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
