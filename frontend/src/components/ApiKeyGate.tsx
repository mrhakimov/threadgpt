"use client"

import { useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { API_URL } from "@/lib/api"

interface Props {
  onSubmit: (apiKey: string) => Promise<void>
}

const frontendInsecure =
  typeof window !== "undefined" &&
  window.location.protocol !== "https:" &&
  window.location.hostname !== "localhost" &&
  window.location.hostname !== "127.0.0.1"

const backendInsecure = (() => {
  try {
    const url = new URL(API_URL)
    return url.protocol === "http:" && url.hostname !== "localhost" && url.hostname !== "127.0.0.1"
  } catch {
    return false
  }
})()

const isInsecureContext = frontendInsecure || backendInsecure

export default function ApiKeyGate({ onSubmit }: Props) {
  const [value, setValue] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = value.trim()
    if (!trimmed) return
    setLoading(true)
    setError(null)
    try {
      await onSubmit(trimmed)
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm flex flex-col gap-8">

        {/* Wordmark */}
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold tracking-tight">ThreadGPT</h1>
          <p className="text-sm text-muted-foreground">
            Each message gets its own isolated context — no bloat, ever.
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          {isInsecureContext && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-xs text-destructive">
              <strong>Warning:</strong> This page is loaded over HTTP. Your API key will be transmitted unencrypted.
            </div>
          )}

          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-medium text-muted-foreground uppercase tracking-widest">
              OpenAI API Key
            </label>
            <Input
              type="password"
              placeholder="sk-..."
              value={value}
              onChange={(e) => setValue(e.target.value)}
              autoFocus
              className="font-mono text-sm border-0 border-b rounded-none px-0 focus-visible:ring-0 focus-visible:ring-offset-0"
            />
          </div>

          {error && (
            <p className="text-xs text-destructive">{error}</p>
          )}

          <Button type="submit" disabled={!value.trim() || loading} className="w-full">
            {loading ? "Connecting..." : "Continue"}
          </Button>
        </form>

        {/* Privacy note */}
        <div className="flex flex-col gap-3">
          <div className="border-t border-border" />
          <p className="text-xs text-muted-foreground">
            Your key is encrypted server-side for your session. Depending on how this server is configured, encrypted session data may also be persisted locally to survive restarts. Your raw API key is never stored in the database.
          </p>
        </div>

        {/* Footer */}
        <p className="text-xs text-muted-foreground">
          Built by{" "}
          <a
            href="https://x.com/omtiness"
            target="_blank"
            rel="noopener noreferrer"
            className="underline hover:text-foreground transition-colors"
          >
            @omtiness
          </a>
          {" · "}
          <a
            href="https://github.com/mrhakimov/threadgpt"
            target="_blank"
            rel="noopener noreferrer"
            className="underline hover:text-foreground transition-colors"
          >
            GitHub
          </a>
        </p>

      </div>
    </div>
  )
}
