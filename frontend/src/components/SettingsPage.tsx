"use client"

import { useState } from "react"
import { X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { logout } from "@/lib/api"

interface Props {
  onClose: () => void
  onLogout: () => void
}

export default function SettingsPage({ onClose, onLogout }: Props) {
  const [confirming, setConfirming] = useState(false)

  async function handleLogout() {
    await logout()
    onLogout()
  }

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
              Session
            </h2>
            <div className="rounded-md border px-4 py-3 text-sm bg-muted/40">
              Session active
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              Your API key is encrypted in server memory for your session and never written to disk or stored in the database.
            </p>
          </section>

          <section>
            <h2 className="text-sm font-medium text-muted-foreground uppercase tracking-wide mb-3">
              Account
            </h2>
            {confirming ? (
              <div className="rounded-md border border-destructive/40 px-4 py-3 space-y-3">
                <p className="text-sm">Are you sure you want to log out? You'll need to re-enter your API key.</p>
                <div className="flex gap-2">
                  <Button variant="destructive" size="sm" onClick={handleLogout}>
                    Log out
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => setConfirming(false)}>
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <Button variant="outline" onClick={() => setConfirming(true)}>
                Log out
              </Button>
            )}
          </section>
        </div>
      </div>
    </div>
  )
}
