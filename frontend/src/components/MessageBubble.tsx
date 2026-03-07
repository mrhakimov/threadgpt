"use client"

import { Message } from "@/types"
import { Button } from "@/components/ui/button"
import { MessageSquare } from "lucide-react"
import { cn } from "@/lib/utils"

interface Props {
  message: Message
  streaming?: boolean
  onReply?: (message: Message) => void
}

export default function MessageBubble({ message, streaming, onReply }: Props) {
  const isAssistant = message.role === "assistant"

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
              : "bg-primary text-primary-foreground rounded-tr-sm"
          )}
        >
          {message.content}
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
