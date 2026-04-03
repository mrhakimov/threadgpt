"use client"

import { useState, useEffect, useRef, useCallback } from "react"
import type { KeyboardEvent, MouseEvent } from "react"
import { Session } from "@/types"
import { MIN_LOADING_MS } from "@/lib/constants"
import ConversationMenuHeader from "@/components/conversations/ConversationMenuHeader"
import NewConversationButton from "@/components/conversations/NewConversationButton"
import ConversationListItem from "@/components/conversations/ConversationListItem"
import LoadingSpinner from "@/components/shared/LoadingSpinner"
import {
  deleteConversation,
  getConversationLabel,
  listSessions,
  renameConversation,
} from "@/services/sessionService"

interface Props {
  activeSessionId: string | null
  isCurrentEmpty?: boolean
  collapsed: boolean
  onToggle: () => void
  onSelectSession: (sessionId: string | null) => void
  onRenameActive?: (name: string) => void
  refreshTrigger?: number
}
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

  const loadSessions = useCallback(async () => {
    loadSessionsStartRef.current = Date.now()
    setLoadingSessions(true)
    setShowLoadingSessions(true)
    try {
      const data = await listSessions()
      setSessions(data.sessions)
      setHasMore(data.has_more)
    } catch {
      // silently fail
    } finally {
      setLoadingSessions(false)
    }
  }, [])

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
  }, [collapsed, loadSessions])

  useEffect(() => {
    if (refreshTrigger && !collapsed) loadSessions()
  }, [refreshTrigger, collapsed, loadSessions])

  useEffect(() => {
    if (editingId) editInputRef.current?.focus()
  }, [editingId])

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

  const loadMoreSessions = useCallback(async () => {
    if (loadingMore || !hasMore) return
    setLoadingMore(true)
    try {
      const data = await listSessions(undefined, sessions.length)
      setSessions((prev) => [...prev, ...data.sessions])
      setHasMore(data.has_more)
    } catch {
      // silently fail
    } finally {
      setLoadingMore(false)
    }
  }, [hasMore, loadingMore, sessions.length])

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

  function startEditing(s: Session, e: MouseEvent) {
    e.stopPropagation()
    setEditingId(s.session_id ?? null)
    setEditingName(getConversationLabel(s))
  }

  async function commitRename(sessionId: string) {
    setEditingId(null)
    try {
      const name = await renameConversation(sessionId, editingName)
      if (!name) return
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

  function handleDeleteClick(s: Session, e: MouseEvent) {
    e.stopPropagation()
    setConfirmDeleteId(s.session_id ?? null)
  }

  async function confirmDelete(sessionId: string) {
    try {
      await deleteConversation(sessionId)
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

  return (
    <aside
      className={`shrink-0 flex flex-col border-r bg-background transition-all duration-200 ${
        collapsed ? "w-[56px]" : "w-64"
      }`}
    >
      <ConversationMenuHeader collapsed={collapsed} onToggle={onToggle} />
      <NewConversationButton collapsed={collapsed} disabled={isCurrentEmpty} onClick={handleNewConversation} />

      {expanded && (
        <div className="flex flex-col flex-1 overflow-hidden p-2 pt-1">
          {error && <p className="text-xs text-destructive mb-2 px-1">{error}</p>}

          <div ref={listRef} onScroll={handleListScroll} className="flex-1 overflow-y-auto space-y-1">
            {showLoadingSessions ? (
              <div className="flex justify-center py-4">
                <LoadingSpinner className="h-4 w-4" />
              </div>
            ) : sessions.length === 0 ? (
              <p className="text-xs text-muted-foreground text-center py-4">No conversations yet</p>
            ) : sessions.map((s) => (
              <ConversationListItem
                key={s.session_id}
                session={s}
                active={s.session_id === activeSessionId}
                label={getConversationLabel(s)}
                editingId={editingId}
                editingName={editingName}
                confirmDeleteId={confirmDeleteId}
                editInputRef={editInputRef}
                onSelect={onSelectSession}
                onStartEditing={startEditing}
                onEditingNameChange={setEditingName}
                onEditKeyDown={handleEditKeyDown}
                onCommitRename={commitRename}
                onDeleteClick={handleDeleteClick}
                onConfirmDelete={confirmDelete}
                onCancelDelete={(e) => {
                  e.stopPropagation()
                  setConfirmDeleteId(null)
                }}
              />
            ))}
            {!showLoadingSessions && loadingMore && (
              <div className="flex justify-center py-2">
                <LoadingSpinner className="h-3.5 w-3.5" />
              </div>
            )}
          </div>
        </div>
      )}
    </aside>
  )
}
