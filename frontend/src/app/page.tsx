"use client"

import { useState, useEffect } from "react"
import ApiKeyGate from "@/components/ApiKeyGate"
import ChatView from "@/components/ChatView"
import { auth } from "@/lib/api"

const STORAGE_KEY = "threadgpt_token"
const SESSION_KEY = "threadgpt_session_id"

export default function Home() {
  const [token, setToken] = useState<string | null>(null)
  const [sessionId, setSessionId] = useState<string | null | undefined>(undefined)
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) setToken(stored)
    const storedSession = localStorage.getItem(SESSION_KEY)
    // undefined = auto-detect latest; null = blank new; string = specific session
    setSessionId(storedSession ?? undefined)
    setMounted(true)
  }, [])

  async function handleApiKey(key: string) {
    const { token: newToken } = await auth(key)
    localStorage.setItem(STORAGE_KEY, newToken)
    localStorage.removeItem(SESSION_KEY)
    setSessionId(undefined)
    setToken(newToken)
  }

  function handleSelectSession(id: string | null) {
    // null = blank new conversation (clear storage); string = specific session
    setSessionId(id)
    if (id) {
      localStorage.setItem(SESSION_KEY, id)
    } else {
      localStorage.removeItem(SESSION_KEY)
    }
  }

  function handleUnauthorized() {
    localStorage.removeItem(STORAGE_KEY)
    setToken(null)
  }

  if (!mounted) return null

  if (!token) {
    return <ApiKeyGate onSubmit={handleApiKey} />
  }

  return (
    <ChatView
      token={token}
      sessionId={sessionId}
      onSelectSession={handleSelectSession}
      onUnauthorized={handleUnauthorized}
    />
  )
}
