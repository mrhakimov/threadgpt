"use client"

import { useState, useEffect } from "react"
import ApiKeyGate from "@/components/ApiKeyGate"
import ChatView from "@/components/ChatView"

const STORAGE_KEY = "threadgpt_api_key"

export default function Home() {
  const [apiKey, setApiKey] = useState<string | null>(null)
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) setApiKey(stored)
    setMounted(true)
  }, [])

  function handleApiKey(key: string) {
    localStorage.setItem(STORAGE_KEY, key)
    setApiKey(key)
  }

  if (!mounted) return null

  if (!apiKey) {
    return <ApiKeyGate onSubmit={handleApiKey} />
  }

  return <ChatView apiKey={apiKey} />
}
