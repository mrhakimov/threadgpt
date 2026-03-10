"use client"

import { useState, useEffect, useRef } from "react"
import { Message } from "@/types"
import { Button } from "@/components/ui/button"
import { MessageSquare, Info } from "lucide-react"
import { cn } from "@/lib/utils"

interface Props {
  message: Message
  streaming?: boolean
  onReply?: (message: Message) => void
  isSystemPrompt?: boolean
}

export default function MessageBubble({ message, streaming, onReply, isSystemPrompt }: Props) {
  const isAssistant = message.role === "assistant"
  const [showTooltip, setShowTooltip] = useState(false)
  const tooltipRef = useRef<HTMLSpanElement>(null)

  useEffect(() => {
    if (!showTooltip) return
    function handleClick(e: MouseEvent) {
      if (tooltipRef.current && !tooltipRef.current.contains(e.target as Node)) {
        setShowTooltip(false)
      }
    }
    document.addEventListener("mousedown", handleClick)
    return () => document.removeEventListener("mousedown", handleClick)
  }, [showTooltip])

  return (
    <div
      className={cn("flex w-full", isAssistant ? "justify-start" : "justify-end")}
    >
      <div className={cn("group relative max-w-[80%]", isAssistant ? "items-start" : "items-end")}>
        <div
          className={cn(
            "rounded-2xl px-4 py-3 text-sm leading-relaxed whitespace-pre-wrap",
            isAssistant
              ? "bg-muted text-foreground rounded-tl-sm"
              : isSystemPrompt
              ? "bg-muted/60 text-foreground border border-border rounded-tr-sm"
              : "bg-primary text-primary-foreground rounded-tr-sm"
          )}
        >
          {message.content}
          {isSystemPrompt && (
            <span className="flex justify-end mt-1">
              <span className="relative" ref={tooltipRef}>
                <Info
                  className="h-3.5 w-3.5 text-muted-foreground cursor-pointer"
                  onClick={() => setShowTooltip((v) => !v)}
                />
                {showTooltip && (
                  <span className="absolute bottom-full right-0 mb-1 px-2 py-1 rounded text-xs bg-popover text-popover-foreground border border-border shadow whitespace-nowrap">
                    system prompt
                  </span>
                )}
              </span>
            </span>
          )}
          {streaming && (
            <span className="inline-block w-2 h-4 ml-1 bg-current opacity-70 animate-pulse align-text-bottom" />
          )}
        </div>

        {isAssistant && onReply && !streaming && (
          <Button
            variant="ghost"
            size="sm"
            className="mt-1 h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
            onClick={() => onReply(message)}
          >
            <MessageSquare className="h-3 w-3 mr-1" />
            Reply
          </Button>
        )}
      </div>
    </div>
  )
}
