"use client"

import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"

interface Props {
  onSubmit: (apiKey: string) => Promise<void>
}

const isInsecureContext =
  typeof window !== "undefined" &&
  window.location.protocol !== "https:" &&
  window.location.hostname !== "localhost" &&
  window.location.hostname !== "127.0.0.1"

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
    <div className="min-h-screen flex flex-col items-center justify-center bg-background p-4">
      {isInsecureContext && (
        <div className="w-full max-w-md mb-4 rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          <strong>Warning:</strong> This page is loaded over HTTP. Your API key will be transmitted unencrypted. Use HTTPS in production.
        </div>
      )}
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>ThreadGPT</CardTitle>
          <CardDescription>
            Enter your OpenAI API key to start chatting. Your key is encrypted in server memory for your session and never written to disk or stored in the database.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <Input
              type="password"
              placeholder="sk-..."
              value={value}
              onChange={(e) => setValue(e.target.value)}
              autoFocus
            />
            {error && <p className="text-xs text-destructive">{error}</p>}
            <Button type="submit" disabled={!value.trim() || loading}>
              {loading ? "Connecting..." : "Continue"}
            </Button>
          </form>
        </CardContent>
      </Card>
      <p className="mt-4 text-xs text-muted-foreground">
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
  )
}
