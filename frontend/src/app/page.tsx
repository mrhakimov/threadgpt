"use client"

import { useState, useEffect } from "react"
import ApiKeyGate from "@/components/ApiKeyGate"
import ChatView from "@/components/ChatView"
import { auth, checkAuth } from "@/lib/api"

const SESSION_KEY = "threadgpt_session_id"

export default function Home() {
  const [loggedIn, setLoggedIn] = useState(false)
  const [sessionId, setSessionId] = useState<string | null | undefined>(undefined)
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    checkAuth().then((ok) => {
      setLoggedIn(ok)
      const storedSession = sessionStorage.getItem(SESSION_KEY)
      // undefined = auto-detect latest; null = blank new; string = specific session
      // If nothing stored, use undefined so useChat auto-detects the latest session
      setSessionId(storedSession ?? null)
      setMounted(true)
    })
  }, [])

  async function handleApiKey(key: string) {
    await auth(key)
    sessionStorage.removeItem(SESSION_KEY)
    setSessionId(null)
    setLoggedIn(true)
  }

  function handleSelectSession(id: string | null) {
    // null = blank new conversation (clear storage); string = specific session
    setSessionId(id)
    if (id) {
      sessionStorage.setItem(SESSION_KEY, id)
    } else {
      sessionStorage.removeItem(SESSION_KEY)
    }
  }

  function handleUnauthorized() {
    setLoggedIn(false)
  }

  if (!mounted) return null

  if (!loggedIn) {
    return <ApiKeyGate onSubmit={handleApiKey} />
  }

  return (
    <ChatView
      sessionId={sessionId}
      onSelectSession={handleSelectSession}
      onUnauthorized={handleUnauthorized}
    />
  )
}
