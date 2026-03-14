"use client"

import { useState, useEffect, useCallback, useRef } from "react"
import { Message, Session } from "@/types"
import { initSession, fetchHistory, fetchSession, sendChatMessage } from "@/lib/api"

// sessionId: string = load that session, null = blank new conversation, undefined = auto-detect latest
export function useChat(token: string, sessionId?: string | null, onSessionResolved?: (sessionId: string) => void, onUnauthorized?: () => void) {
  const onSessionResolvedRef = useRef(onSessionResolved)
  onSessionResolvedRef.current = onSessionResolved
  const onUnauthorizedRef = useRef(onUnauthorized)
  onUnauthorizedRef.current = onUnauthorized

  const [messages, setMessages] = useState<Message[]>([])
  const [session, setSession] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [streamingContent, setStreamingContent] = useState("")
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
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
          const [history, sessionData] = await Promise.all([
            fetchHistory(token, sessionId),
            fetchSession(sessionId, token),
          ])
          if (!cancelled) {
            setMessages(history)
            setSession({ session_id: sessionId, is_new: false, name: sessionData.name, system_prompt: sessionData.system_prompt })
          }
        } else {
          // undefined: auto-detect latest session
          const sessionData = await initSession(token)
          if (cancelled) return
          setSession(sessionData)

          if (!sessionData.is_new) {
            const history = await fetchHistory(token)
            if (!cancelled) {
              setMessages(history)
              if (sessionData.session_id) onSessionResolvedRef.current?.(sessionData.session_id)
            }
          }
        }
      } catch (e) {
        if (!cancelled) {
          const msg = String(e)
          if (msg.includes("unauthorized") || msg.includes("401")) {
            onUnauthorizedRef.current?.()
          } else {
            setError(msg)
          }
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    init()
    return () => { cancelled = true }
  }, [token, sessionId])

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
      const streamedSessionId = await sendChatMessage(token, content, (chunk) => {
        accumulated += chunk
        setStreamingContent(accumulated)
      }, activeSessionId, forceNew)

      // Use the session ID returned from the stream; fall back to activeSessionId
      const resolvedSessionId = streamedSessionId ?? activeSessionId
      const history = await fetchHistory(token, resolvedSessionId)
      setMessages(history)
      setStreamingContent("")

      if (resolvedSessionId && resolvedSessionId !== (sessionId || session?.session_id)) {
        setSession({ session_id: resolvedSessionId, is_new: false })
        onSessionResolvedRef.current?.(resolvedSessionId)
      } else if (!session?.assistant_id && resolvedSessionId) {
        const sessionData = await fetchSession(resolvedSessionId, token)
        setSession({ session_id: resolvedSessionId, is_new: false, name: sessionData.name, system_prompt: sessionData.system_prompt })
      }
    } catch (e) {
      const msg = String(e)
      if (msg.includes("unauthorized") || msg.includes("401")) {
        onUnauthorizedRef.current?.()
      } else {
        setError(msg)
      }
      setStreamingContent("")
    } finally {
      setSending(false)
    }
  }, [token, sending, session, sessionId])

  function updateLocalSystemPrompt(content: string) {
    setSession((prev) => prev ? { ...prev, system_prompt: content } : prev)
    setMessages((prev) => {
      if (prev.length === 0 || prev[0].role !== "user") return prev
      return [{ ...prev[0], content }, ...prev.slice(1)]
    })
  }

  function incrementReplyCount(messageId: string, by: number) {
    setMessages((prev) =>
      prev.map((m) => m.id === messageId ? { ...m, reply_count: (m.reply_count ?? 0) + by } : m)
    )
  }

  return { messages, session, loading, sending, streamingContent, error, sendMessage, updateLocalSystemPrompt, incrementReplyCount }
}
