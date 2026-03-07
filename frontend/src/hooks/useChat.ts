"use client"

import { useState, useEffect, useCallback, useRef } from "react"
import { Message, Session } from "@/types"
import { initSession, fetchHistory, sendChatMessage } from "@/lib/api"

export function useChat(apiKey: string, sessionId?: string, onSessionResolved?: (sessionId: string) => void) {
  const onSessionResolvedRef = useRef(onSessionResolved)
  onSessionResolvedRef.current = onSessionResolved

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
        setMessages([])

        if (sessionId) {
          // Load a specific existing session
          const history = await fetchHistory(apiKey, sessionId)
          if (!cancelled) {
            setMessages(history)
            setSession({ session_id: sessionId, is_new: false })
          }
        } else {
          const sessionData = await initSession(apiKey)
          if (cancelled) return
          setSession(sessionData)

          if (!sessionData.is_new) {
            const history = await fetchHistory(apiKey)
            if (!cancelled) {
              setMessages(history)
              if (sessionData.session_id) onSessionResolvedRef.current?.(sessionData.session_id)
            }
          }
        }
      } catch (e) {
        if (!cancelled) setError(String(e))
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    init()
    return () => { cancelled = true }
  }, [apiKey, sessionId])

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
      }, session?.session_id || sessionId)

      const history = await fetchHistory(apiKey, session?.session_id || sessionId)
      setMessages(history)

      if (session?.is_new || !session?.assistant_id) {
        const sessionData = await initSession(apiKey)
        setSession(sessionData)
        if (sessionData.session_id && !sessionId) onSessionResolvedRef.current?.(sessionData.session_id)
      }
    } catch (e) {
      setError(String(e))
    } finally {
      setStreamingContent("")
      setSending(false)
    }
  }, [apiKey, sending, session, sessionId])

  return { messages, session, loading, sending, streamingContent, error, sendMessage }
}
