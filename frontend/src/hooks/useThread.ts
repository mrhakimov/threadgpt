"use client"

import { useState, useCallback, useEffect } from "react"
import { Message } from "@/types"
import { fetchThreadMessages, sendThreadMessage } from "@/lib/api"

export function useThread(token: string, parentMessageId: string, onReplySent?: () => void) {
  const [messages, setMessages] = useState<Message[]>([])
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)

  useEffect(() => {
    setLoading(true)
    fetchThreadMessages(token, parentMessageId)
      .then((msgs: Message[]) => { if (msgs?.length) setMessages(msgs) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [token, parentMessageId])
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
      await sendThreadMessage(token, parentMessageId, content, (chunk) => {
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
      setStreamingContent("")
      onReplySent?.()
    } catch (e) {
      setError(String(e))
      setStreamingContent("")
    } finally {
      setSending(false)
    }
  }, [token, parentMessageId, sending, onReplySent])

  return { messages, loading, sending, streamingContent, error, sendMessage }
}
