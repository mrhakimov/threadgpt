"use client"

import { useState, useEffect, useCallback, useRef } from "react"
import { Message, Session } from "@/types"
import { isUnauthorizedError, toErrorMessage } from "@/domain/errors"
import {
  applySystemPromptLocally,
  buildOptimisticChatMessage,
  incrementLocalReplyCount,
  loadChatSession,
  loadCompleteChatHistory,
  loadOlderChatMessages,
  sendChatTurn,
} from "@/services/chatService"

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

        const data = await loadChatSession(sessionId)
        if (!cancelled) {
          setMessages(data.messages)
          setHasMoreMessages(data.hasMoreMessages)
          setSession(data.session)
          if (data.resolvedSessionId) {
            onSessionResolvedRef.current?.(data.resolvedSessionId)
          }
        }
      } catch (e) {
        if (!cancelled) {
          if (isUnauthorizedError(e)) {
            onUnauthorizedRef.current?.()
          } else {
            setError(toErrorMessage(e))
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

  const loadMoreMessages = useCallback(async (
    scrollEl?: HTMLDivElement | null,
    options?: { anchor?: "preserve" | "bottom" },
  ) => {
    if (loadingMore || !hasMoreMessages) return
    const activeSessionId = sessionId || session?.session_id
    if (!activeSessionId) return
    setLoadingMore(true)
    const el = scrollEl ?? scrollRefForPreserve.current
    const prevScrollHeight = el?.scrollHeight ?? 0
    try {
      // Backend returns desc (newest first, then reversed to asc). offset from newest end.
      // We already have `messages.length` newest messages; fetch the next older batch.
      const historyData = await loadOlderChatMessages(activeSessionId, messages.length)
      setMessages((prev) => [...historyData.messages, ...prev])
      setHasMoreMessages(historyData.has_more)
      // Preserve scroll position after prepend
      if (el) {
        requestAnimationFrame(() => {
          if (options?.anchor === "bottom") {
            el.scrollTop = el.scrollHeight
            return
          }
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
    const el = scrollEl ?? scrollRefForPreserve.current

    if (!hasMoreMessages) {
      if (el) {
        requestAnimationFrame(() => {
          el.scrollTop = 0
        })
      }
      return
    }

    setLoadingMore(true)
    try {
      const historyData = await loadCompleteChatHistory(activeSessionId)
      setMessages(historyData.messages)
      setHasMoreMessages(false)
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
  }, [sessionId, session, hasMoreMessages])

  const sendMessage = useCallback(async (content: string) => {
    if (sending) return
    setSending(true)
    setError(null)

    const controller = new AbortController()
    sendAbortRef.current = controller

    const userMsg: Message = buildOptimisticChatMessage(content, session?.session_id)
    setMessages((prev) => [...prev, userMsg])
    setStreamingContent("")

    try {
      let accumulated = ""
      const result = await sendChatTurn({
        content,
        requestedSessionId: sessionId,
        currentSession: session,
        onChunk: (chunk) => {
          accumulated += chunk
          setStreamingContent(accumulated)
        },
        signal: controller.signal,
      })

      if (controller.signal.aborted) return

      setMessages(result.history.messages)
      setHasMoreMessages(result.history.has_more)
      setStreamingContent("")
      setSession(result.session)

      if (
        result.resolvedSessionId &&
        result.resolvedSessionId !== (sessionId || session?.session_id)
      ) {
        onSessionResolvedRef.current?.(result.resolvedSessionId)
      }
    } catch (e) {
      if (controller.signal.aborted) return
      if (isUnauthorizedError(e)) {
        onUnauthorizedRef.current?.()
      } else {
        setError(toErrorMessage(e))
      }
      setStreamingContent("")
    } finally {
      setSending(false)
    }
  }, [sending, session, sessionId])

  function updateLocalSystemPrompt(content: string) {
    setSession((prev) => prev ? { ...prev, system_prompt: content } : prev)
    setMessages((prev) => applySystemPromptLocally(prev, content))
  }

  function incrementReplyCount(messageId: string, by: number) {
    setMessages((prev) => incrementLocalReplyCount(prev, messageId, by))
  }

  return { messages, hasMoreMessages, loadingMore, session, loading, sending, streamingContent, error, sendMessage, loadMoreMessages, loadAllMessages, updateLocalSystemPrompt, incrementReplyCount }
}
