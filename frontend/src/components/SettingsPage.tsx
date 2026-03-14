"use client"

import { X } from "lucide-react"
import { Button } from "@/components/ui/button"

interface Props {
  token: string
  onClose: () => void
}

export default function SettingsPage({ token, onClose }: Props) {
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
              Session Token
            </h2>
            <div className="rounded-md border px-4 py-3 font-mono text-sm break-all bg-muted/40">
              {token.slice(0, 8) + "••••••••"}
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              Your API key was exchanged for a session token. The raw key is never stored locally.
            </p>
          </section>
        </div>
      </div>
    </div>
  )
}
