"use client"

import { X } from "lucide-react"
import { Button } from "@/components/ui/button"

interface Props {
  apiKey: string
  onClose: () => void
}

function maskApiKey(key: string): string {
  if (key.length <= 12) return key
  const start = key.slice(0, 6)
  const end = key.slice(-4)
  const masked = "•".repeat(Math.min(key.length - 10, 20))
  return `${start}${masked}${end}`
}

export default function SettingsPage({ apiKey, onClose }: Props) {
  return (
    <div className="fixed inset-0 z-50 bg-background flex flex-col">
      <header className="shrink-0 border-b px-4 py-3 flex items-center justify-between">
        <h1 className="font-semibold">Settings</h1>
        <Button variant="ghost" size="icon" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </header>

      <div className="flex-1 overflow-y-auto px-4 py-6">
        <div className="max-w-lg mx-auto space-y-6">
          <section>
            <h2 className="text-sm font-medium text-muted-foreground uppercase tracking-wide mb-3">
              API Key
            </h2>
            <div className="rounded-md border px-4 py-3 font-mono text-sm break-all bg-muted/40">
              {maskApiKey(apiKey)}
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              Your API key is stored locally and never sent to our servers.
            </p>
          </section>
        </div>
      </div>
    </div>
  )
}
