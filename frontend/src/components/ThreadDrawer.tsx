"use client"

import { useEffect, useRef, useCallback, useState } from "react"
import { createPortal } from "react-dom"
import { Message } from "@/types"
import { useThread } from "@/hooks/useThread"
import MessageList from "./MessageList"
import ChatInput from "./ChatInput"
import { Button } from "@/components/ui/button"
import { X, Loader2 } from "lucide-react"
import { MIN_LOADING_MS } from "@/lib/constants"

interface Props {
  parentMessage: Message
  onClose: () => void
  onReply?: (parentMessageId: string) => void
  container?: HTMLElement | null
  onAbortRef?: (abortFn: (() => void) | null) => void
}

const DURATION = 300
const SLIDE_IN_MS = 300

export default function ThreadDrawer({ parentMessage, onClose, onReply, container, onAbortRef }: Props) {
  const { messages, hasMore, loadingMore, loading, sending, streamingContent, error, sendMessage, loadMore, abort } = useThread(
    parentMessage.id,
    onReply ? () => onReply(parentMessage.id) : undefined
  )
  const scrollRef = useRef<HTMLDivElement>(null)
  const [closing, setClosing] = useState(false)
  const [showLoading, setShowLoading] = useState(true)
  const minLoadingDoneRef = useRef(false)
  const dataLoadedRef = useRef(false)

  // After drawer opens, show spinner for MIN_LOADING_MS, then hide if data is also ready
  useEffect(() => {
    const openTimer = setTimeout(() => {
      setShowLoading(true)
      const minTimer = setTimeout(() => {
        minLoadingDoneRef.current = true
        if (dataLoadedRef.current) setShowLoading(false)
      }, MIN_LOADING_MS)
      return () => clearTimeout(minTimer)
    }, SLIDE_IN_MS)
    return () => clearTimeout(openTimer)
  }, [])

  useEffect(() => {
    if (!loading) {
      dataLoadedRef.current = true
      if (minLoadingDoneRef.current) setShowLoading(false)
    }
  }, [loading])

  function handleClose() {
    setClosing(true)
    setTimeout(onClose, DURATION)
  }

  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    if (el.scrollTop === 0 && hasMore && !loadingMore) loadMore(el)
  }, [hasMore, loadingMore, loadMore])

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
  }, [])

  const drawerAnim = closing
    ? "drawer-out 300ms ease-in-out forwards"
    : "drawer-in 300ms ease-in-out forwards"

  const overlayAnim = closing
    ? "overlay-out 300ms ease forwards"
    : "overlay-in 300ms ease forwards"

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

          <div className="px-4 py-3 border-b bg-muted/50">
            <p className="text-xs text-muted-foreground mb-1">Following up on</p>
            <p className="text-sm text-foreground whitespace-pre-wrap line-clamp-5">
              {parentMessage.content}
            </p>
          </div>

          <div ref={scrollRef} onScroll={handleScroll} className="flex-1 overflow-y-auto px-4">
            {showLoading ? (
              <div className="flex justify-center mt-8">
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              </div>
            ) : (
              <>
                {messages.length === 0 && !streamingContent && (
                  <p className="text-sm text-muted-foreground text-center mt-8">
                    Ask a follow-up question below.
                  </p>
                )}
                <MessageList messages={messages} streamingContent={streamingContent} sending={sending} scrollRef={scrollRef} hasMore={hasMore} loadingMore={loadingMore} onLoadMore={loadMore} />
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
