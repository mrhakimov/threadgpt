"use client"

import { useEffect, useRef, useCallback, useState } from "react"
import { createPortal } from "react-dom"
import { Message } from "@/types"
import { useThread } from "@/hooks/useThread"
import MessageList from "./MessageList"
import ChatInput from "./ChatInput"
import { Button } from "@/components/ui/button"
import { X } from "lucide-react"
import { MIN_LOADING_MS } from "@/lib/constants"
import LoadingSpinner from "@/components/shared/LoadingSpinner"

interface Props {
  parentMessage: Message
  onClose: () => void
  onReply?: (parentMessageId: string) => void
  onAbortRef?: (abortFn: (() => void) | null) => void
}

const DURATION = 300
const SLIDE_IN_MS = 300
const LOAD_MORE_TOP_THRESHOLD = 200

export default function ThreadDrawer({ parentMessage, onClose, onReply, onAbortRef }: Props) {
  const { messages, hasMore, loadingMore, loading, sending, streamingContent, error, sendMessage, loadMore, abort } = useThread(
    parentMessage.id,
    onReply ? () => onReply(parentMessage.id) : undefined
  )
  const scrollRef = useRef<HTMLDivElement>(null)
  const [closing, setClosing] = useState(false)
  const [showLoading, setShowLoading] = useState(true)
  const [canLoadMoreOnScroll, setCanLoadMoreOnScroll] = useState(false)
  const minLoadingDoneRef = useRef(false)
  const dataLoadedRef = useRef(false)
  const minLoadingTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // After drawer opens, show spinner for MIN_LOADING_MS, then hide if data is also ready
  useEffect(() => {
    const openTimer = setTimeout(() => {
      setShowLoading(true)
      minLoadingTimerRef.current = setTimeout(() => {
        minLoadingDoneRef.current = true
        if (dataLoadedRef.current) setShowLoading(false)
      }, MIN_LOADING_MS)
    }, SLIDE_IN_MS)

    return () => {
      clearTimeout(openTimer)
      if (minLoadingTimerRef.current) {
        clearTimeout(minLoadingTimerRef.current)
      }
    }
  }, [])

  useEffect(() => {
    if (!loading) {
      dataLoadedRef.current = true
      if (minLoadingDoneRef.current) setShowLoading(false)
    }
  }, [loading])

  useEffect(() => {
    setCanLoadMoreOnScroll(false)
    if (scrollRef.current) scrollRef.current.scrollTop = 0
  }, [parentMessage.id])

  const handleClose = useCallback(() => {
    setClosing(true)
    setTimeout(onClose, DURATION)
  }, [onClose])

  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    if (canLoadMoreOnScroll && el.scrollTop <= LOAD_MORE_TOP_THRESHOLD && hasMore && !loadingMore) loadMore(el)
  }, [canLoadMoreOnScroll, hasMore, loadingMore, loadMore])

  useEffect(() => {
    onAbortRef?.(abort)
    return () => onAbortRef?.(null)
  }, [abort, onAbortRef])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") handleClose()
    }
    document.addEventListener("keydown", onKey)
    return () => document.removeEventListener("keydown", onKey)
  }, [handleClose])

  const drawerAnim = closing
    ? "drawer-out 300ms ease-in-out forwards"
    : "drawer-in 300ms ease-in-out forwards"

  const overlayAnim = closing
    ? "overlay-out 300ms ease forwards"
    : "overlay-in 300ms ease forwards"
  const showEmptyState = messages.length === 0 && !streamingContent && !sending
  const showThreadMessages = !showEmptyState || loadingMore
  const bodyClassName = showLoading
    ? "flex-1 overflow-hidden px-4"
    : showEmptyState
    ? "flex-1 overflow-hidden px-4"
    : "flex-1 overflow-y-auto px-4"

  return (
    <>
      {createPortal(
        <div
          className="fixed inset-0 z-40"
          style={{ backgroundColor: "rgba(0,0,0,0.25)", animation: overlayAnim }}
          onClick={handleClose}
        />,
        document.body
      )}

      {createPortal(
        <div
          className="fixed right-0 top-0 h-full w-full max-w-md bg-background border-l shadow-xl z-50 flex flex-col"
          style={{ animation: drawerAnim }}
        >
          <div className="flex items-center justify-between px-4 py-3 border-b">
            <span className="font-semibold text-sm">Thread</span>
            <Button variant="ghost" size="icon" onClick={handleClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>

          <div className="border-b">
            <div className="px-4 py-3">
              <div className="flex gap-3 items-start">
                <div className="w-0.5 self-stretch rounded-full bg-muted-foreground/25 shrink-0 mt-0.5" />
                <div className="min-w-0">
                  <p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground/50 mb-1">Following up on</p>
                  <p className="text-sm text-muted-foreground whitespace-pre-wrap line-clamp-4 leading-relaxed">
                    {parentMessage.content}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div ref={scrollRef} onScroll={handleScroll} className={bodyClassName}>
            {showLoading ? (
              <div className="flex h-full items-center justify-center">
                <LoadingSpinner className="h-5 w-5" />
              </div>
            ) : (
              <>
                {showEmptyState && (
                  <div className="pt-6">
                    <p className="text-sm text-center text-muted-foreground">
                      Ask a follow-up question below.
                    </p>
                  </div>
                )}
                {showThreadMessages && (
                  <MessageList
                    messages={messages}
                    streamingContent={streamingContent}
                    sending={sending}
                    scrollRef={scrollRef}
                    scrollContextKey={parentMessage.id}
                    onInitialScrollComplete={() => setCanLoadMoreOnScroll(true)}
                    hasMore={hasMore}
                    loadingMore={loadingMore}
                    onLoadMore={loadMore}
                    contentAlignment="top"
                    initialScrollPosition="bottom"
                  />
                )}
              </>
            )}
          </div>

          {error && <p className="text-xs text-destructive px-4 pb-2">{error}</p>}

          <div className="px-4 py-3 border-t">
            <ChatInput
              onSend={sendMessage}
              disabled={sending}
              placeholder="Ask a follow-up..."
              focusTrigger={1}
            />
          </div>
        </div>,
        document.body
      )}
    </>
  )
}
