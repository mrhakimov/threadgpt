"use client"

import { useState, useEffect } from "react"
import ApiKeyGate from "@/components/ApiKeyGate"
import ChatView from "@/components/ChatView"
import {
  authenticateWithApiKey,
  checkAuthorization,
} from "@/services/authService"
import { storageService } from "@/services/storageService"

export default function Home() {
  const [loggedIn, setLoggedIn] = useState(false)
  const [sessionId, setSessionId] = useState<string | null | undefined>(undefined)
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    checkAuthorization().then((ok) => {
      setLoggedIn(ok)
      // undefined = auto-detect latest; null = blank new; string = specific session
      // If nothing stored, use undefined so useChat auto-detects the latest session
      setSessionId(storageService.getSelectedSessionId() ?? null)
      setMounted(true)
    })
  }, [])

  async function handleApiKey(key: string) {
    await authenticateWithApiKey(key)
    storageService.setSelectedSessionId(null)
    setSessionId(null)
    setLoggedIn(true)
  }

  function handleSelectSession(id: string | null) {
    // null = blank new conversation (clear storage); string = specific session
    setSessionId(id)
    storageService.setSelectedSessionId(id)
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
