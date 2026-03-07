"use client"

import { useState, useEffect, useCallback, useRef } from "react"
import { Message, Session } from "@/types"
import { initSession, fetchHistory, sendChatMessage } from "@/lib/api"

// sessionId: string = load that session, null = blank new conversation, undefined = auto-detect latest
export function useChat(apiKey: string, sessionId?: string | null, onSessionResolved?: (sessionId: string) => void) {
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
        setSession(null)

        if (sessionId === null) {
          // Explicit blank new conversation — don't load anything
          setLoading(false)
          return
        }

        if (sessionId) {
          // Load a specific existing session
          const history = await fetchHistory(apiKey, sessionId)
          if (!cancelled) {
            setMessages(history)
            setSession({ session_id: sessionId, is_new: false })
          }
        } else {
          // undefined: auto-detect latest session
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
      const forceNew = sessionId === null
      const activeSessionId = forceNew ? undefined : (sessionId || session?.session_id || undefined)
      const streamedSessionId = await sendChatMessage(apiKey, content, (chunk) => {
        accumulated += chunk
        setStreamingContent(accumulated)
      }, activeSessionId, forceNew)

      // Use the session ID returned from the stream; fall back to activeSessionId
      const resolvedSessionId = streamedSessionId ?? activeSessionId
      const history = await fetchHistory(apiKey, resolvedSessionId)
      setMessages(history)

      if (resolvedSessionId && resolvedSessionId !== (sessionId || session?.session_id)) {
        setSession({ session_id: resolvedSessionId, is_new: false })
        onSessionResolvedRef.current?.(resolvedSessionId)
      } else if (!session?.assistant_id) {
        const sessionData = await initSession(apiKey)
        setSession(sessionData)
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
