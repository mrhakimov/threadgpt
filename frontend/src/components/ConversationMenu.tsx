"use client"

import { useState, useEffect, useRef, useCallback, KeyboardEvent } from "react"
import { Button } from "@/components/ui/button"
import { PanelLeftOpen, PanelLeftClose, Plus, X, Pencil, Trash2, Check, Loader2 } from "lucide-react"
import { fetchSessions, renameSession, deleteSession } from "@/lib/api"
import { Session } from "@/types"
import { MIN_LOADING_MS } from "@/lib/constants"

interface Props {
  activeSessionId: string | null
  isCurrentEmpty?: boolean
  collapsed: boolean
  onToggle: () => void
  onSelectSession: (sessionId: string | null) => void
  onRenameActive?: (name: string) => void
  refreshTrigger?: number
}

const SESSIONS_PAGE_SIZE = 20

export default function ConversationMenu({ activeSessionId, isCurrentEmpty, collapsed, onToggle, onSelectSession, onRenameActive, refreshTrigger }: Props) {
  const [expanded, setExpanded] = useState(!collapsed)
  const [sessions, setSessions] = useState<Session[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [loadingSessions, setLoadingSessions] = useState(false)
  const [showLoadingSessions, setShowLoadingSessions] = useState(false)
  const loadSessionsStartRef = useRef(0)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editingName, setEditingName] = useState("")
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null)
  const editInputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!collapsed) {
      setSessions([])
      const t = setTimeout(() => {
        setExpanded(true)
        loadSessions()
      }, 200)
      return () => clearTimeout(t)
    } else {
      setExpanded(false)
    }
  }, [collapsed])

  useEffect(() => {
    if (refreshTrigger && !collapsed) loadSessions()
  }, [refreshTrigger])

  useEffect(() => {
    if (editingId) editInputRef.current?.focus()
  }, [editingId])

  async function loadSessions() {
    loadSessionsStartRef.current = Date.now()
    setLoadingSessions(true)
    setShowLoadingSessions(true)
    try {
      const data = await fetchSessions(SESSIONS_PAGE_SIZE, 0)
      setSessions(data.sessions)
      setHasMore(data.has_more)
    } catch {
      // silently fail
    } finally {
      setLoadingSessions(false)
    }
  }

  useEffect(() => {
    if (!loadingSessions) {
      const elapsed = Date.now() - loadSessionsStartRef.current
      const remaining = MIN_LOADING_MS - elapsed
      if (remaining > 0) {
        const t = setTimeout(() => setShowLoadingSessions(false), remaining)
        return () => clearTimeout(t)
      }
      setShowLoadingSessions(false)
    }
  }, [loadingSessions])

  async function loadMoreSessions() {
    if (loadingMore || !hasMore) return
    setLoadingMore(true)
    try {
      const data = await fetchSessions(SESSIONS_PAGE_SIZE, sessions.length)
      setSessions((prev) => [...prev, ...data.sessions])
      setHasMore(data.has_more)
    } catch {
      // silently fail
    } finally {
      setLoadingMore(false)
    }
  }

  const handleListScroll = useCallback(() => {
    const el = listRef.current
    if (!el) return
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 100 && hasMore && !loadingMore) {
      loadMoreSessions()
    }
  }, [hasMore, loadingMore, loadMoreSessions])

  function handleNewConversation() {
    if (isCurrentEmpty) return
    onSelectSession(null)
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
    if (name.length > 256) return
    try {
      await renameSession(sessionId, name)
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
      await deleteSession(sessionId)
      setSessions((prev) => prev.filter((x) => x.session_id !== sessionId))
      if (activeSessionId === sessionId) {
        onSelectSession(null)
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
    <aside
      className={`shrink-0 flex flex-col border-r bg-background transition-all duration-200 ${
        collapsed ? "w-[56px]" : "w-64"
      }`}
    >
      {/* Toggle button */}
      <div className="flex items-center border-b px-2 py-3 gap-1">
        <Button variant="ghost" size="icon" className="shrink-0" onClick={onToggle} aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}>
          {collapsed ? <PanelLeftOpen className="h-5 w-5" /> : <PanelLeftClose className="h-5 w-5" />}
        </Button>
        {!collapsed && <span className="text-sm font-medium">Conversations</span>}
      </div>

      {/* New conversation button — always rendered so it animates with the sidebar */}
      <div className="flex items-center px-2 py-1">
        <Button
          variant="outline"
          size="sm"
          onClick={handleNewConversation}
          disabled={isCurrentEmpty}
          title="New conversation"
          className={`w-full justify-start gap-2 overflow-hidden ${collapsed ? "border-transparent bg-transparent shadow-none hover:bg-accent" : ""}`}
        >
          <Plus className="h-4 w-4 shrink-0" />
          <span className={`truncate ${collapsed ? "hidden" : ""}`}>New conversation</span>
        </Button>
      </div>

      {/* Sidebar content (session list) */}
      {expanded && (
        <div className="flex flex-col flex-1 overflow-hidden p-2 pt-1">
          {error && <p className="text-xs text-destructive mb-2 px-1">{error}</p>}

          <div ref={listRef} onScroll={handleListScroll} className="flex-1 overflow-y-auto space-y-1">
            {showLoadingSessions ? (
              <div className="flex justify-center py-4">
                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              </div>
            ) : sessions.length === 0 ? (
              <p className="text-xs text-muted-foreground text-center py-4">No conversations yet</p>
            ) : sessions.map((s) => (
              <div
                key={s.session_id}
                className={`group w-full text-left rounded-md px-2 py-2 text-sm flex items-center gap-2 hover:bg-muted transition-colors ${
                  s.session_id === activeSessionId ? "bg-muted font-medium" : ""
                }`}
              >
                {editingId === s.session_id ? (
                  <>
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
                      className="flex-1 min-w-0 text-left"
                      onClick={() => onSelectSession(s.session_id ?? null)}
                    >
                      <span className="block truncate">{getSessionLabel(s)}</span>
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
            {(hasMore || loadingMore) && (
              <div className="flex justify-center py-2">
                <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />
              </div>
            )}
          </div>
        </div>
      )}
    </aside>
  )
}
