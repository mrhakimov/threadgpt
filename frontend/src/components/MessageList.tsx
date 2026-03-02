"use client"

import { useEffect, useRef } from "react"
import { Message } from "@/types"
import MessageBubble from "./MessageBubble"

interface Props {
  messages: Message[]
  streamingContent?: string
  onReply?: (message: Message) => void
}

export default function MessageList({ messages, streamingContent, onReply }: Props) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [messages, streamingContent])

  return (
    <div className="flex flex-col gap-4 py-4">
      {messages.map((msg) => (
        <MessageBubble key={msg.id} message={msg} onReply={onReply} />
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

      <div ref={bottomRef} />
    </div>
  )
}
