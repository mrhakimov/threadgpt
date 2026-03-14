"use client"

import { useState, useEffect, useRef, KeyboardEvent } from "react"
import { Button } from "@/components/ui/button"
import { Menu, Plus, MessageSquare, X, Pencil, Trash2, Check } from "lucide-react"
import { fetchSessions, renameSession, deleteSession } from "@/lib/api"

interface Props {
  token: string
  activeSessionId: string | null
  isCurrentEmpty?: boolean
  onSelectSession: (sessionId: string | null) => void
  onRenameActive?: (name: string) => void
}

export default function ConversationMenu({ token, activeSessionId, isCurrentEmpty, onSelectSession, onRenameActive }: Props) {
  const [open, setOpen] = useState(false)
  const [sessions, setSessions] = useState<Session[]>([])
  const [error, setError] = useState<string | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editingName, setEditingName] = useState("")
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const editInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!open) return
    loadSessions()
  }, [open])

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setOpen(false)
        setEditingId(null)
        setConfirmDeleteId(null)
      }
    }
    if (open) document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [open])

  useEffect(() => {
    if (editingId) editInputRef.current?.focus()
  }, [editingId])

  async function loadSessions() {
    try {
      const data = await fetchSessions(token)
      setSessions(data)
    } catch {
      // silently fail
    }
  }

  function handleNewConversation() {
    if (isCurrentEmpty) return
    onSelectSession(null)
    setOpen(false)
  }

  function startEditing(s: Session, e: React.MouseEvent) {
    e.stopPropagation()
    setEditingId(s.session_id ?? null)
    setEditingName(getSessionLabel(s))
  }

  async function commitRename(sessionId: string) {
    const name = editingName.trim()
    setEditingId(null)
    if (!name) return
    try {
      await renameSession(token, sessionId, name)
      setSessions((prev) =>
        prev.map((s) => (s.session_id === sessionId ? { ...s, name } : s))
      )
      if (sessionId === activeSessionId) onRenameActive?.(name)
    } catch (e) {
      setError(String(e))
    }
  }

  function handleEditKeyDown(e: KeyboardEvent<HTMLInputElement>, sessionId: string) {
    if (e.key === "Enter") commitRename(sessionId)
    else if (e.key === "Escape") setEditingId(null)
  }

  function handleDeleteClick(s: Session, e: React.MouseEvent) {
    e.stopPropagation()
    setConfirmDeleteId(s.session_id ?? null)
  }

  async function confirmDelete(sessionId: string) {
    try {
      await deleteSession(token, sessionId)
      setSessions((prev) => prev.filter((x) => x.session_id !== sessionId))
      if (activeSessionId === sessionId) {
        onSelectSession(null)
        setOpen(false)
      }
    } catch (e) {
      setError(String(e))
    } finally {
      setConfirmDeleteId(null)
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
                <div
                  key={s.session_id}
                  className={`group w-full text-left rounded-md px-3 py-2 text-sm flex items-center gap-2 hover:bg-muted transition-colors ${
                    s.session_id === activeSessionId ? "bg-muted font-medium" : ""
                  }`}
                >
                  {editingId === s.session_id ? (
                    <>
                      <MessageSquare className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <input
                        ref={editInputRef}
                        className="flex-1 min-w-0 bg-transparent outline-none text-sm"
                        value={editingName}
                        onChange={(e) => setEditingName(e.target.value)}
                        onKeyDown={(e) => handleEditKeyDown(e, s.session_id!)}
                        onBlur={() => commitRename(s.session_id!)}
                      />
                      <button
                        className="shrink-0 text-muted-foreground hover:text-foreground"
                        onMouseDown={(e) => { e.preventDefault(); commitRename(s.session_id!) }}
                      >
                        <Check className="h-3.5 w-3.5" />
                      </button>
                    </>
                  ) : (
                    <>
                      <button
                        className="flex-1 min-w-0 flex items-start gap-2 text-left"
                        onClick={() => {
                          onSelectSession(s.session_id ?? null)
                          setOpen(false)
                        }}
                      >
                        <MessageSquare className="h-4 w-4 mt-0.5 shrink-0 text-muted-foreground" />
                        <span className="truncate">{getSessionLabel(s)}</span>
                      </button>
                      {confirmDeleteId === s.session_id ? (
                        <div className="shrink-0 flex items-center gap-1">
                          <span className="text-xs text-muted-foreground">Delete?</span>
                          <button
                            className="px-1.5 py-0.5 rounded text-xs bg-destructive text-destructive-foreground hover:bg-destructive/90"
                            onClick={(e) => { e.stopPropagation(); confirmDelete(s.session_id!) }}
                          >
                            Yes
                          </button>
                          <button
                            className="px-1.5 py-0.5 rounded text-xs hover:bg-accent text-muted-foreground"
                            onClick={(e) => { e.stopPropagation(); setConfirmDeleteId(null) }}
                          >
                            No
                          </button>
                        </div>
                      ) : (
                        <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                          <button
                            className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-accent"
                            onClick={(e) => startEditing(s, e)}
                            title="Rename"
                          >
                            <Pencil className="h-3.5 w-3.5" />
                          </button>
                          <button
                            className="p-1 rounded text-muted-foreground hover:text-destructive hover:bg-accent"
                            onClick={(e) => handleDeleteClick(s, e)}
                            title="Delete"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      )}
                    </>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
