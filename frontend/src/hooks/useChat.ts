"use client"

import { useState, useEffect, useCallback } from "react"
import { Message, Session } from "@/types"
import { initSession, fetchHistory, sendChatMessage } from "@/lib/api"

export function useChat(apiKey: string) {
  const [messages, setMessages] = useState<Message[]>([])
  const [session, setSession] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [streamingContent, setStreamingContent] = useState("")
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!apiKey) return
    let cancelled = false

    async function init() {
      try {
        setLoading(true)
        const sessionData = await initSession(apiKey)
        if (cancelled) return
        setSession(sessionData)

        if (!sessionData.is_new) {
          const history = await fetchHistory(apiKey)
          if (!cancelled) setMessages(history)
        }
      } catch (e) {
        if (!cancelled) setError(String(e))
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    init()
    return () => { cancelled = true }
  }, [apiKey])

  const sendMessage = useCallback(async (content: string) => {
    if (sending) return
    setSending(true)
    setError(null)

    const userMsg: Message = {
      id: crypto.randomUUID(),
      session_id: session?.session_id ?? "",
      role: "user",
      content,
      created_at: new Date().toISOString(),
    }
    setMessages((prev) => [...prev, userMsg])
    setStreamingContent("")

    let accumulated = ""

    try {
      await sendChatMessage(apiKey, content, (chunk) => {
        accumulated += chunk
        setStreamingContent(accumulated)
      })

      // Reload history from Supabase to get real DB-persisted IDs
      // (needed so Reply uses the real message ID, not a client-generated UUID)
      const history = await fetchHistory(apiKey)
      setMessages(history)

      // Update session if this was the first message (assistant now exists)
      if (session?.is_new || !session?.assistant_id) {
        const sessionData = await initSession(apiKey)
        setSession(sessionData)
      }
    } catch (e) {
      setError(String(e))
    } finally {
      setStreamingContent("")
      setSending(false)
    }
  }, [apiKey, sending, session])

  return { messages, session, loading, sending, streamingContent, error, sendMessage }
}
