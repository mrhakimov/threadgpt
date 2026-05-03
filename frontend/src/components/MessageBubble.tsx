"use client"

import { useState, useEffect, useRef } from "react"
import { Message } from "@/types"
import { Button } from "@/components/ui/button"
import { MessageSquare, Info, Pencil, Check, X, Copy, CopyCheck } from "lucide-react"
import { toErrorMessage } from "@/domain/errors"
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
  const [copied, setCopied] = useState(false)
  const [editing, setEditing] = useState(false)
  const [editValue, setEditValue] = useState(message.content)
  const bubbleRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!editing) setEditValue(message.content)
  }, [message.content, editing])
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const tooltipRef = useRef<HTMLSpanElement>(null)
  const editableRef = useRef<HTMLDivElement>(null)

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
    if (editing && editableRef.current) {
      const el = editableRef.current
      el.focus()
      const range = document.createRange()
      range.selectNodeContents(el)
      range.collapse(false)
      const sel = window.getSelection()
      sel?.removeAllRanges()
      sel?.addRange(range)
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
      setSaveError(toErrorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  function handleCancel() {
    if (editableRef.current) editableRef.current.textContent = message.content
    setEditValue(message.content)
    setEditing(false)
  }

  async function handleCopy() {
    await navigator.clipboard.writeText(message.content)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const copyButton = (
    <button onClick={handleCopy} className="text-muted-foreground hover:text-foreground transition-colors" title="Copy">
      {copied ? <CopyCheck className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
    </button>
  )

  return (
    <div
      className={cn("flex w-full", isAssistant ? "justify-start" : "justify-end")}
    >
      <div className={cn("group relative", isAssistant ? "w-[80%] items-start" : "max-w-[80%] items-end")}>
        <div
          ref={bubbleRef}
          className={cn(
            "relative rounded-2xl px-4 text-sm leading-relaxed",
            isAssistant
              ? "bg-muted text-foreground rounded-tl-sm py-3"
              : isSystemPrompt
              ? "bg-muted/60 text-foreground border border-border rounded-tr-sm py-3 min-w-[6rem]"
              : "bg-secondary text-secondary-foreground rounded-tr-sm py-3"
          )}
        >
          {/* Agent message: copy button top-right, always visible */}
          {isAssistant && !streaming && (
            <span className="absolute top-2 right-2">{copyButton}</span>
          )}

          <div
            ref={editableRef}
            contentEditable={editing}
            suppressContentEditableWarning
            onInput={(e) => setEditValue(e.currentTarget.textContent ?? "")}
            onKeyDown={(e) => {
              if (e.key === "Escape") { e.preventDefault(); handleCancel() }
              if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) { e.preventDefault(); handleSave() }
            }}
            className={cn("whitespace-pre-wrap outline-none", isAssistant && "pr-5", editing && "cursor-text")}
          >{message.content}</div>
          {saveError && (
            <p className="text-xs text-destructive mt-1">{saveError}</p>
          )}
          {isSystemPrompt && (
            <span className="flex justify-end items-center gap-1.5 mt-1.5">
              {/* slot 1: Pencil (view) / X (edit) */}
              {onEditSystemPrompt && (
                <span className="relative w-3.5 h-3.5">
                  <button
                    onClick={() => setEditing(true)}
                    className={cn("absolute inset-0 text-muted-foreground hover:text-foreground transition-colors", editing ? "invisible" : "opacity-0 group-hover:opacity-100 transition-opacity")}
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </button>
                  <button
                    onClick={handleCancel}
                    disabled={saving}
                    className={cn("absolute inset-0 text-muted-foreground hover:text-foreground transition-colors", !editing && "invisible")}
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                </span>
              )}
              {/* slot 2: Copy (view) / Check (edit) */}
              <span className="relative w-3.5 h-3.5">
                <button
                  onClick={handleCopy}
                  className={cn("absolute inset-0 text-muted-foreground hover:text-foreground transition-colors", editing ? "invisible" : "opacity-0 group-hover:opacity-100 transition-opacity")}
                  title="Copy"
                >
                  {copied ? <CopyCheck className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                </button>
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className={cn("absolute inset-0 text-muted-foreground hover:text-foreground transition-colors", !editing && "invisible")}
                >
                  <Check className="h-3.5 w-3.5" />
                </button>
              </span>
              {/* info icon — always visible */}
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

        {/* User message: copy button below pill, right side, hover only */}
        {!isAssistant && !isSystemPrompt && !streaming && (
          <div className="flex justify-end mt-1 opacity-0 group-hover:opacity-100 transition-opacity">
            {copyButton}
          </div>
        )}

        {isAssistant && onReply && !streaming && (
          <Button
            variant="ghost"
            size="sm"
            className="mt-0.5 h-6 w-full justify-start px-1 text-xs text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded-lg"
            onClick={() => onReply(message)}
          >
            <MessageSquare className="h-3 w-3 mr-1" />
            {message.reply_count && message.reply_count > 0
              ? message.reply_count === 1 ? "1 follow-up" : `${message.reply_count} follow-ups`
              : "Follow up"}
          </Button>
        )}
      </div>
    </div>
  )
}
