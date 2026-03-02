"use client"

import { useState, useCallback } from "react"
import { Message } from "@/types"
import { sendThreadMessage } from "@/lib/api"

export function useThread(apiKey: string, parentMessageId: string) {
  const [messages, setMessages] = useState<Message[]>([])
  const [sending, setSending] = useState(false)
  const [streamingContent, setStreamingContent] = useState("")
  const [error, setError] = useState<string | null>(null)

  const sendMessage = useCallback(async (content: string) => {
    if (sending) return
    setSending(true)
    setError(null)

    const userMsg: Message = {
      id: crypto.randomUUID(),
      session_id: "",
      role: "user",
      content,
      created_at: new Date().toISOString(),
    }
    setMessages((prev) => [...prev, userMsg])
    setStreamingContent("")

    let accumulated = ""

    try {
      await sendThreadMessage(apiKey, parentMessageId, content, (chunk) => {
        accumulated += chunk
        setStreamingContent(accumulated)
      })

      const assistantMsg: Message = {
        id: crypto.randomUUID(),
        session_id: "",
        role: "assistant",
        content: accumulated,
        created_at: new Date().toISOString(),
      }
      setMessages((prev) => [...prev, assistantMsg])
    } catch (e) {
      setError(String(e))
    } finally {
      setStreamingContent("")
      setSending(false)
    }
  }, [apiKey, parentMessageId, sending])

  return { messages, sending, streamingContent, error, sendMessage }
}
