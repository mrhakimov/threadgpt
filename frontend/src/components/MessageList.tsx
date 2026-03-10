"use client"

import { useEffect, RefObject } from "react"
import { Message } from "@/types"
import MessageBubble from "./MessageBubble"

interface Props {
  messages: Message[]
  streamingContent?: string
  onReply?: (message: Message) => void
  scrollRef?: RefObject<HTMLDivElement>
}

export default function MessageList({ messages, streamingContent, onReply, scrollRef }: Props) {
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
        <MessageBubble key={msg.id} message={msg} onReply={onReply} isSystemPrompt={i === 0 && msg.role === "user"} />
      ))}

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
