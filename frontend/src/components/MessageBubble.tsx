"use client"

import { useState, useEffect, useRef } from "react"
import { Message } from "@/types"
import { Button } from "@/components/ui/button"
import { MessageSquare, Info, Pencil, Check, X } from "lucide-react"
import { cn } from "@/lib/utils"

interface Props {
  message: Message
  streaming?: boolean
  onReply?: (message: Message) => void
  isSystemPrompt?: boolean
  onEditSystemPrompt?: (newContent: string) => Promise<void>
}

export default function MessageBubble({ message, streaming, onReply, isSystemPrompt, onEditSystemPrompt }: Props) {
  const isAssistant = message.role === "assistant"
  const [showTooltip, setShowTooltip] = useState(false)
  const [editing, setEditing] = useState(false)
  const [editValue, setEditValue] = useState(message.content)

  useEffect(() => {
    if (!editing) setEditValue(message.content)
  }, [message.content, editing])
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const tooltipRef = useRef<HTMLSpanElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

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

  useEffect(() => {
    if (editing && textareaRef.current) {
      textareaRef.current.focus()
      textareaRef.current.selectionStart = textareaRef.current.value.length
    }
  }, [editing])

  async function handleSave() {
    if (!onEditSystemPrompt || editValue.trim() === message.content.trim()) {
      setEditing(false)
      return
    }
    setSaving(true)
    setSaveError(null)
    try {
      await onEditSystemPrompt(editValue.trim())
      setEditing(false)
    } catch (e) {
      setSaveError(String(e))
    } finally {
      setSaving(false)
    }
  }

  function handleCancel() {
    setEditValue(message.content)
    setEditing(false)
  }

  return (
    <div
      className={cn("flex w-full", isAssistant ? "justify-start" : "justify-end")}
    >
      <div className={cn("group relative max-w-[80%]", isAssistant ? "items-start" : "items-end")}>
        <div
          className={cn(
            "rounded-2xl px-4 py-3 text-sm leading-relaxed",
            isAssistant
              ? "bg-muted text-foreground rounded-tl-sm"
              : isSystemPrompt
              ? "bg-muted/60 text-foreground border border-border rounded-tr-sm"
              : "bg-primary text-primary-foreground rounded-tr-sm"
          )}
        >
          {editing ? (
            <textarea
              ref={textareaRef}
              value={editValue}
              onChange={(e) => setEditValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Escape") handleCancel()
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) handleSave()
              }}
              className="w-full min-w-[240px] bg-transparent resize-none outline-none whitespace-pre-wrap"
              rows={Math.max(3, editValue.split("\n").length)}
              disabled={saving}
            />
          ) : (
            <span className="whitespace-pre-wrap">{message.content}</span>
          )}
          {saveError && (
            <p className="text-xs text-destructive mt-1">{saveError}</p>
          )}
          {isSystemPrompt && (
            <span className="flex justify-end items-center gap-1.5 mt-1">
              {editing ? (
                <>
                  <button onClick={handleCancel} disabled={saving} className="text-muted-foreground hover:text-foreground transition-colors">
                    <X className="h-3.5 w-3.5" />
                  </button>
                  <button onClick={handleSave} disabled={saving} className="text-muted-foreground hover:text-foreground transition-colors">
                    <Check className="h-3.5 w-3.5" />
                  </button>
                </>
              ) : (
                onEditSystemPrompt && (
                  <button
                    onClick={() => setEditing(true)}
                    className="opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-foreground"
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </button>
                )
              )}
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
