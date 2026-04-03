"use client"

import { useState, useCallback, useEffect, useRef } from "react"
import { Message } from "@/types"
import { createOptimisticUserMessage } from "@/domain/chat"
import { toErrorMessage } from "@/domain/errors"
import {
  loadOlderThreadMessages,
  loadThreadHistory,
  streamThreadTurn,
} from "@/services/threadService"

export function useThread(parentMessageId: string, onReplySent?: () => void) {
  const [messages, setMessages] = useState<Message[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [sending, setSending] = useState(false)

  useEffect(() => {
    setLoading(true)
    loadThreadHistory(parentMessageId)
      .then((data) => {
        setMessages(data.messages ?? [])
        setHasMore(data.has_more ?? false)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [parentMessageId])

  const [streamingContent, setStreamingContent] = useState("")
  const [error, setError] = useState<string | null>(null)
  const abortRef = useRef<AbortController | null>(null)
  const activeRef = useRef(true)

  useEffect(() => {
    activeRef.current = true
    return () => {
      activeRef.current = false
      abortRef.current?.abort()
    }
  }, [])

  const loadMore = useCallback(async (
    scrollEl?: HTMLDivElement | null,
    options?: { anchor?: "preserve" | "bottom" },
  ) => {
    if (loadingMore || !hasMore) return
    setLoadingMore(true)
    const prevScrollHeight = scrollEl?.scrollHeight ?? 0
    try {
      // Backend returns desc reversed to asc. offset from newest end — older messages.
      const data = await loadOlderThreadMessages(parentMessageId, messages.length)
      setMessages((prev) => [...data.messages, ...prev])
      setHasMore(data.has_more)
      if (scrollEl) {
        requestAnimationFrame(() => {
          if (options?.anchor === "bottom") {
            scrollEl.scrollTop = scrollEl.scrollHeight
            return
          }
          scrollEl.scrollTop = scrollEl.scrollHeight - prevScrollHeight
        })
      }
    } catch {
      // silently fail
    } finally {
      setLoadingMore(false)
    }
  }, [parentMessageId, messages.length, loadingMore, hasMore])

  const sendMessage = useCallback(async (content: string) => {
    if (sending) return
    setSending(true)
    setError(null)

    const controller = new AbortController()
    abortRef.current = controller

    const userMsg: Message = createOptimisticUserMessage(content)
    setMessages((prev) => [...prev, userMsg])
    setStreamingContent("")

    let accumulated = ""

    try {
      await streamThreadTurn(parentMessageId, content, (chunk) => {
        accumulated += chunk
        setStreamingContent(accumulated)
      }, controller.signal)

      if (!activeRef.current || controller.signal.aborted) return
      const assistantMsg: Message = {
        ...createOptimisticUserMessage(""),
        role: "assistant",
        content: accumulated,
      }
      setMessages((prev) => [...prev, assistantMsg])
      setStreamingContent("")
      onReplySent?.()
    } catch (e) {
      if (e instanceof Error && e.name === "AbortError") return
      if (!activeRef.current) return
      setError(toErrorMessage(e))
      setStreamingContent("")
    } finally {
      if (activeRef.current) setSending(false)
    }
  }, [parentMessageId, sending, onReplySent])

  const abort = useCallback(() => {
    activeRef.current = false
    abortRef.current?.abort()
  }, [])

  return { messages, hasMore, loading, loadingMore, sending, streamingContent, error, sendMessage, loadMore, abort }
}
