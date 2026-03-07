"use client"

import { useState, useEffect, useRef } from "react"
import { Session } from "@/types"
import { Button } from "@/components/ui/button"
import { Menu, Plus, MessageSquare, X } from "lucide-react"
import { fetchSessions, createSession } from "@/lib/api"

interface Props {
  apiKey: string
  activeSessionId: string | null
  isCurrentEmpty?: boolean
  onSelectSession: (sessionId: string | null) => void
  onSessionCreated: (session: Session) => void
}

export default function ConversationMenu({ apiKey, activeSessionId, isCurrentEmpty, onSelectSession, onSessionCreated }: Props) {
  const [open, setOpen] = useState(false)
  const [sessions, setSessions] = useState<Session[]>([])
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    loadSessions()
  }, [open])

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [open])

  async function loadSessions() {
    try {
      const data = await fetchSessions(apiKey)
      setSessions(data)
    } catch {
      // silently fail
    }
  }

  async function handleNewConversation() {
    if (isCurrentEmpty) return
    setCreating(true)
    setError(null)
    try {
      const session = await createSession(apiKey, "New conversation")
      setSessions((prev) => [session, ...prev])
      onSessionCreated(session)
      onSelectSession(session.session_id ?? null)
      setOpen(false)
    } catch (e) {
      setError(String(e))
    } finally {
      setCreating(false)
    }
  }

  function getSessionLabel(s: Session) {
    if (s.name && s.name !== "New conversation") return s.name
    if (s.system_prompt) return s.system_prompt
    return "New conversation"
  }

  return (
    <div className="relative" ref={menuRef}>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => setOpen((v) => !v)}
        aria-label="Conversations"
      >
        <Menu className="h-5 w-5" />
      </Button>

      {open && (
        <div className="absolute left-0 top-10 z-50 w-72 rounded-lg border bg-background shadow-lg">
          <div className="flex items-center justify-between px-3 py-2 border-b">
            <span className="text-sm font-medium">Conversations</span>
            <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => setOpen(false)}>
              <X className="h-4 w-4" />
            </Button>
          </div>

          <div className="p-2">
            <Button
              variant="outline"
              size="sm"
              className="w-full justify-start gap-2 mb-2"
              onClick={handleNewConversation}
              disabled={creating}
            >
              <Plus className="h-4 w-4" />
              New conversation
            </Button>
            {error && <p className="text-xs text-destructive mb-2 px-1">{error}</p>}

            <div className="max-h-80 overflow-y-auto space-y-1">
              {sessions.length === 0 && (
                <p className="text-xs text-muted-foreground text-center py-4">No conversations yet</p>
              )}
              {sessions.map((s) => (
                <button
                  key={s.session_id}
                  className={`w-full text-left rounded-md px-3 py-2 text-sm flex items-start gap-2 hover:bg-muted transition-colors ${
                    s.session_id === activeSessionId ? "bg-muted font-medium" : ""
                  }`}
                  onClick={() => {
                    onSelectSession(s.session_id ?? null)
                    setOpen(false)
                  }}
                >
                  <MessageSquare className="h-4 w-4 mt-0.5 shrink-0 text-muted-foreground" />
                  <span className="truncate">{getSessionLabel(s)}</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
