"use client"

import { useState, useCallback, useEffect } from "react"
import { Message } from "@/types"
import { fetchThreadMessages, sendThreadMessage } from "@/lib/api"

const PAGE_SIZE = 10

export function useThread(token: string, parentMessageId: string, onReplySent?: () => void) {
  const [messages, setMessages] = useState<Message[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [sending, setSending] = useState(false)

  useEffect(() => {
    setLoading(true)
    fetchThreadMessages(token, parentMessageId, PAGE_SIZE, 0)
      .then((data) => {
        setMessages(data.messages ?? [])
        setHasMore(data.has_more ?? false)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [token, parentMessageId])

  const [streamingContent, setStreamingContent] = useState("")
  const [error, setError] = useState<string | null>(null)

  const loadMore = useCallback(async (scrollEl?: HTMLDivElement | null) => {
    if (loadingMore || !hasMore) return
    setLoadingMore(true)
    const prevScrollHeight = scrollEl?.scrollHeight ?? 0
    try {
      // Backend returns desc reversed to asc. offset from newest end — older messages.
      const data = await fetchThreadMessages(token, parentMessageId, PAGE_SIZE, messages.length)
      setMessages((prev) => [...data.messages, ...prev])
      setHasMore(data.has_more)
      if (scrollEl) {
        requestAnimationFrame(() => {
          scrollEl.scrollTop = scrollEl.scrollHeight - prevScrollHeight
        })
      }
    } catch {
      // silently fail
    } finally {
      setLoadingMore(false)
    }
  }, [token, parentMessageId, messages.length, loadingMore, hasMore])

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

  return { messages, hasMore, loading, loadingMore, sending, streamingContent, error, sendMessage, loadMore }
}
