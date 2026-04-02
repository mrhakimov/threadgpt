"use client"

import { useState, useEffect, useCallback, useRef } from "react"
import { Message, Session } from "@/types"
import { initSession, fetchHistory, fetchSession, sendChatMessage } from "@/lib/api"

const PAGE_SIZE = 10

// sessionId: string = load that session, null = blank new conversation, undefined = auto-detect latest
export function useChat(sessionId?: string | null, onSessionResolved?: (sessionId: string) => void, onUnauthorized?: () => void) {
  const onSessionResolvedRef = useRef(onSessionResolved)
  onSessionResolvedRef.current = onSessionResolved
  const onUnauthorizedRef = useRef(onUnauthorized)
  onUnauthorizedRef.current = onUnauthorized

  const [messages, setMessages] = useState<Message[]>([])
  const [hasMoreMessages, setHasMoreMessages] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [session, setSession] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [streamingContent, setStreamingContent] = useState("")
  const [error, setError] = useState<string | null>(null)

  const sessionRef = useRef(session)
  sessionRef.current = session

  useEffect(() => {
    let cancelled = false

    async function init() {
      try {
        // If the session was just resolved from a new conversation (null → ID),
        // the messages are already in state from sendMessage — skip re-fetching.
        if (sessionId && sessionRef.current?.session_id === sessionId) {
          return
        }

        setLoading(true)
        setMessages([])
        setHasMoreMessages(false)
        setSession(null)

        if (sessionId === null) {
          // Explicit blank new conversation — don't load anything
          setLoading(false)
          return
        }

        if (sessionId) {
          // Load a specific existing session
          const [historyData, sessionData] = await Promise.all([
            fetchHistory(sessionId, PAGE_SIZE, 0),
            fetchSession(sessionId),
          ])
          if (!cancelled) {
            setMessages(historyData.messages)
            setHasMoreMessages(historyData.has_more)
            setSession({ session_id: sessionId, is_new: false, name: sessionData.name, system_prompt: sessionData.system_prompt })
          }
        } else {
          // undefined: auto-detect latest session
          const sessionData = await initSession()
          if (cancelled) return
          setSession(sessionData)

          if (!sessionData.is_new) {
            const historyData = await fetchHistory(undefined, PAGE_SIZE, 0)
            if (!cancelled) {
              setMessages(historyData.messages)
              setHasMoreMessages(historyData.has_more)
              if (sessionData.session_id) onSessionResolvedRef.current?.(sessionData.session_id)
            }
          }
        }
      } catch (e) {
        if (!cancelled) {
          const msg = String(e)
          if (msg.toLowerCase().includes("unauthorized") || msg.includes("401")) {
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
  }, [sessionId])

  const sendAbortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    return () => {
      sendAbortRef.current?.abort()
      setSending(false)
      setStreamingContent("")
    }
  }, [sessionId])

  const scrollRefForPreserve = useRef<HTMLDivElement | null>(null)

  const loadMoreMessages = useCallback(async (scrollEl?: HTMLDivElement | null) => {
    if (loadingMore || !hasMoreMessages) return
    const activeSessionId = sessionId || session?.session_id
    if (!activeSessionId) return
    setLoadingMore(true)
    const el = scrollEl ?? scrollRefForPreserve.current
    const prevScrollHeight = el?.scrollHeight ?? 0
    try {
      // Backend returns desc (newest first, then reversed to asc). offset from newest end.
      // We already have `messages.length` newest messages; fetch the next older batch.
      const historyData = await fetchHistory(activeSessionId, PAGE_SIZE, messages.length)
      setMessages((prev) => [...historyData.messages, ...prev])
      setHasMoreMessages(historyData.has_more)
      // Preserve scroll position after prepend
      if (el) {
        requestAnimationFrame(() => {
          el.scrollTop = el.scrollHeight - prevScrollHeight
        })
      }
    } catch {
      // silently fail
    } finally {
      setLoadingMore(false)
    }
  }, [sessionId, session, messages.length, loadingMore, hasMoreMessages])

  const loadAllMessages = useCallback(async (scrollEl?: HTMLDivElement | null) => {
    const activeSessionId = sessionId || session?.session_id
    if (!activeSessionId) return
    setLoadingMore(true)
    try {
      const historyData = await fetchHistory(activeSessionId, 10000, 0)
      setMessages(historyData.messages)
      setHasMoreMessages(false)
      const el = scrollEl ?? scrollRefForPreserve.current
      if (el) {
        requestAnimationFrame(() => {
          el.scrollTop = 0
        })
      }
    } catch {
      // silently fail
    } finally {
      setLoadingMore(false)
    }
  }, [sessionId, session])

  const sendMessage = useCallback(async (content: string) => {
    if (sending) return
    setSending(true)
    setError(null)

    const controller = new AbortController()
    sendAbortRef.current = controller

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
      const streamedSessionId = await sendChatMessage(content, (chunk) => {
        accumulated += chunk
        setStreamingContent(accumulated)
      }, activeSessionId, forceNew, controller.signal)

      if (controller.signal.aborted) return

      // Use the session ID returned from the stream; fall back to activeSessionId
      const resolvedSessionId = streamedSessionId ?? activeSessionId
      const historyData = await fetchHistory(resolvedSessionId, PAGE_SIZE, 0)

      if (controller.signal.aborted) return

      setMessages(historyData.messages)
      setHasMoreMessages(historyData.has_more)
      setStreamingContent("")

      if (resolvedSessionId && resolvedSessionId !== (sessionId || session?.session_id)) {
        setSession({ session_id: resolvedSessionId, is_new: false })
        onSessionResolvedRef.current?.(resolvedSessionId)
      } else if (!session?.assistant_id && resolvedSessionId) {
        const sessionData = await fetchSession(resolvedSessionId)
        if (!controller.signal.aborted) {
          setSession({ session_id: resolvedSessionId, is_new: false, name: sessionData.name, system_prompt: sessionData.system_prompt })
        }
      }
    } catch (e) {
      if (controller.signal.aborted) return
      const msg = String(e)
      if (msg.toLowerCase().includes("unauthorized") || msg.includes("401")) {
        onUnauthorizedRef.current?.()
      } else {
        setError(msg)
      }
      setStreamingContent("")
    } finally {
      setSending(false)
    }
  }, [sending, session, sessionId])

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

  return { messages, hasMoreMessages, loadingMore, session, loading, sending, streamingContent, error, sendMessage, loadMoreMessages, loadAllMessages, updateLocalSystemPrompt, incrementReplyCount }
}
