"use client"

import { useState, useRef, useEffect } from "react"
import { Textarea } from "@/components/ui/textarea"
import { ArrowUp } from "lucide-react"
import { cn } from "@/lib/utils"

interface Props {
  onSend: (message: string) => void
  disabled?: boolean
  placeholder?: string
  focusTrigger?: number
}

export default function ChatInput({ onSend, disabled, placeholder, focusTrigger }: Props) {
  const [value, setValue] = useState("")
  const [isFocused, setIsFocused] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto"
      textareaRef.current.style.height = Math.min(textareaRef.current.scrollHeight, 160) + "px"
    }
  }, [value])

  useEffect(() => {
    if (focusTrigger) textareaRef.current?.focus()
  }, [focusTrigger])

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey && !disabled) {
      e.preventDefault()
      handleSend()
    }
  }

  function handleSend() {
    const trimmed = value.trim()
    if (!trimmed || disabled) return
    onSend(trimmed)
    setValue("")
  }

  const hasContent = value.trim().length > 0

  return (
    <div
      onClick={() => textareaRef.current?.focus()}
      className={cn(
        "relative flex flex-col rounded-2xl border bg-muted/50 px-4 pt-3 pb-3 transition-shadow cursor-text",
        isFocused ? "shadow-md" : "shadow-md"
      )}
    >
      <Textarea
        ref={textareaRef}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        onFocus={() => setIsFocused(true)}
        onBlur={() => setIsFocused(false)}
        placeholder={placeholder ?? "Message ThreadGPT"}
        rows={1}
        className="resize-none min-h-[24px] max-h-[160px] overflow-y-auto border-0 bg-transparent shadow-none p-0 focus-visible:ring-0 focus-visible:ring-offset-0 text-sm leading-relaxed cursor-text"
      />
      <div className="flex justify-end mt-2">
        <button
          onClick={(e) => { e.stopPropagation(); handleSend() }}
          disabled={disabled || !hasContent}
          aria-label="Send message"
          className={cn(
            "inline-flex items-center justify-center h-7 w-7 rounded-full",
            "bg-primary/15 text-primary hover:bg-primary/25",
            "transition-all duration-150",
            hasContent && !disabled ? "opacity-100 cursor-pointer" : "opacity-0 pointer-events-none"
          )}
        >
          <ArrowUp className="h-4 w-4" />
        </button>
      </div>
    </div>
  )
}
