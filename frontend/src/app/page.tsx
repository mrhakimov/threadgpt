"use client"

import { useState, useEffect } from "react"
import ApiKeyGate from "@/components/ApiKeyGate"
import ChatView from "@/components/ChatView"

const STORAGE_KEY = "threadgpt_api_key"
const SESSION_KEY = "threadgpt_session_id"

export default function Home() {
  const [apiKey, setApiKey] = useState<string | null>(null)
  const [sessionId, setSessionId] = useState<string | null | undefined>(undefined)
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) setApiKey(stored)
    const storedSession = localStorage.getItem(SESSION_KEY)
    // undefined = auto-detect latest; null = blank new; string = specific session
    setSessionId(storedSession ?? undefined)
    setMounted(true)
  }, [])

  function handleApiKey(key: string) {
    localStorage.setItem(STORAGE_KEY, key)
    setApiKey(key)
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

  if (!mounted) return null

  if (!apiKey) {
    return <ApiKeyGate onSubmit={handleApiKey} />
  }

  return (
    <ChatView
      apiKey={apiKey}
      sessionId={sessionId}
      onSelectSession={handleSelectSession}
    />
  )
}
