"use client"

import { useEffect, useRef, RefObject } from "react"
import { Message } from "@/types"
import MessageBubble from "./MessageBubble"
import LoadingSpinner from "@/components/shared/LoadingSpinner"

interface Props {
  messages: Message[]
  streamingContent?: string
  sending?: boolean
  onReply?: (message: Message) => void
  onEditSystemPrompt?: (newContent: string) => Promise<void>
  scrollRef?: RefObject<HTMLDivElement | null>
  scrollContextKey?: string
  onInitialScrollComplete?: () => void
  showSystemPrompt?: boolean
  hasMore?: boolean
  loadingMore?: boolean
  onLoadMore?: () => void
}

export default function MessageList({ messages, streamingContent, sending, onReply, onEditSystemPrompt, scrollRef, scrollContextKey, onInitialScrollComplete, showSystemPrompt, hasMore, loadingMore, onLoadMore }: Props) {
  const didInitialScroll = useRef(false)
  const firstMessageIdRef = useRef<string | undefined>(undefined)
  const lastMessageIdRef = useRef<string | undefined>(undefined)
  const messageCountRef = useRef(0)
  const streamingContentRef = useRef<string | undefined>(undefined)

  void onLoadMore

  useEffect(() => {
    didInitialScroll.current = false
    firstMessageIdRef.current = undefined
    lastMessageIdRef.current = undefined
    messageCountRef.current = 0
    streamingContentRef.current = undefined
  }, [scrollContextKey])

  useEffect(() => {
    const el = scrollRef?.current
    if (!el) return

    const prevFirstId = firstMessageIdRef.current
    const prevLastId = lastMessageIdRef.current
    const prevMessageCount = messageCountRef.current
    const prevStreamingContent = streamingContentRef.current

    const firstId = messages[0]?.id
    const lastId = messages[messages.length - 1]?.id
    const wasPrepend =
      prevFirstId !== undefined &&
      firstId !== prevFirstId &&
      lastId === prevLastId &&
      messages.length > prevMessageCount
    const tailChanged = prevLastId !== undefined && lastId !== prevLastId
    const streamingChanged = streamingContent !== prevStreamingContent

    firstMessageIdRef.current = firstId
    lastMessageIdRef.current = lastId
    messageCountRef.current = messages.length
    streamingContentRef.current = streamingContent

    // Don't auto-scroll to bottom when prepending older messages
    if (wasPrepend) return

    if (!didInitialScroll.current && messages.length > 0) {
      didInitialScroll.current = true
      let raf1 = 0
      let raf2 = 0

      const settleAtBottom = () => {
        el.scrollTop = el.scrollHeight
        raf1 = requestAnimationFrame(() => {
          el.scrollTop = el.scrollHeight
          raf2 = requestAnimationFrame(() => {
            el.scrollTop = el.scrollHeight
            onInitialScrollComplete?.()
          })
        })
      }

      settleAtBottom()

      return () => {
        cancelAnimationFrame(raf1)
        cancelAnimationFrame(raf2)
      }
    }

    if (!tailChanged && !streamingChanged) return

    el.scrollTo({ top: el.scrollHeight, behavior: "smooth" })
  }, [messages, streamingContent, scrollRef, onInitialScrollComplete])

  return (
    <div className="min-h-full flex flex-col justify-end gap-4 py-4">
      {loadingMore && (
        <div className="flex justify-center py-2">
          <LoadingSpinner className="h-4 w-4" />
        </div>
      )}

      {messages.map((msg, i) => {
        // The context-set confirmation is the first assistant reply (index 1, no more pages above)
        const isContextSetConfirmation = msg.role === "assistant" && !hasMore && i === 1
        return (
          <MessageBubble key={msg.id} message={msg} onReply={isContextSetConfirmation ? undefined : onReply} isSystemPrompt={showSystemPrompt && !hasMore && i === 0 && msg.role === "user"} onEditSystemPrompt={showSystemPrompt && !hasMore && i === 0 && msg.role === "user" ? onEditSystemPrompt : undefined} />
        )
      })}

      {sending && !streamingContent && (
        <div className="flex gap-1 px-4 py-2">
          <span className="w-2 h-2 rounded-full bg-muted-foreground/50 animate-bounce [animation-delay:-0.3s]" />
          <span className="w-2 h-2 rounded-full bg-muted-foreground/50 animate-bounce [animation-delay:-0.15s]" />
          <span className="w-2 h-2 rounded-full bg-muted-foreground/50 animate-bounce" />
        </div>
      )}

      {streamingContent && (
        <MessageBubble
          message={{
            id: "__streaming__",
            session_id: "",
            role: "assistant",
            content: streamingContent,
            created_at: new Date().toISOString(),
          }}
          streaming
        />
      )}
    </div>
  )
}
