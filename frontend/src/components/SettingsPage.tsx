"use client"

import { useState } from "react"
import { X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { logoutUser } from "@/services/authService"
import { cn } from "@/lib/utils"
import type { Theme } from "@/hooks/useTheme"

interface Props {
  closing: boolean
  onClose: () => void
  onLogout: () => void
  theme: Theme
  setTheme: (t: Theme) => void
}

const THEME_OPTIONS: { label: string; value: Theme }[] = [
  { label: "System", value: "system" },
  { label: "Light", value: "light" },
  { label: "Dark", value: "dark" },
]

export default function SettingsPage({ closing, onClose, onLogout, theme, setTheme }: Props) {
  const [confirming, setConfirming] = useState(false)

  async function handleLogout() {
    await logoutUser()
    onLogout()
  }

  return (
    <div
      className={cn(
        "fixed inset-0 z-50 bg-background flex flex-col",
        closing ? "animate-settings-page-out" : "animate-settings-page-in"
      )}
    >
      <header className="shrink-0 border-b px-4 py-3 flex items-center gap-3">
        <h1 className="font-semibold">Settings</h1>
        <div className="ml-auto flex items-center gap-1">
          <Button variant="ghost" size="icon" onClick={onClose} disabled={closing}>
            <X className={cn("h-4 w-4", closing ? "animate-settings-close-out" : "animate-settings-close-in")} />
          </Button>
        </div>
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-6">
        <div className="max-w-lg mx-auto space-y-8">

          {/* Appearance */}
          <section>
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-widest mb-1">
              Appearance
            </p>
            <div className="border-b border-border" />
            <div className="flex items-center justify-between py-3">
              <span className="text-sm">Dark mode</span>
              <div className="flex items-center rounded-md border border-border overflow-hidden">
                {THEME_OPTIONS.map((opt) => (
                  <button
                    key={opt.value}
                    onClick={() => setTheme(opt.value)}
                    className={cn(
                      "px-3 py-1.5 text-xs font-medium transition-colors",
                      theme === opt.value
                        ? "bg-secondary text-secondary-foreground"
                        : "bg-background text-muted-foreground hover:text-foreground hover:bg-muted"
                    )}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>
          </section>

          {/* API Key */}
          <section>
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-widest mb-1">
              API Key
            </p>
            <div className="border-b border-border" />
            <div className="flex items-center justify-between py-3 border-b border-border">
              <span className="text-sm">Session key</span>
              <span className="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-1 text-[11px] font-medium text-emerald-700 dark:text-emerald-400">
                ● Active
              </span>
            </div>
            <div className="py-3">
              <p className="text-xs text-muted-foreground">
                Encrypted in server memory only — never written to disk or stored in the database.
              </p>
            </div>
          </section>

          {/* Account */}
          <section>
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-widest mb-1">
              Account
            </p>
            <div className="border-b border-border" />
            <div className="flex items-center justify-between py-3">
              <span className="text-sm">Log out</span>
              <div className="flex items-center gap-2">
                {confirming ? (
                  <>
                    <Button variant="ghost" size="sm" onClick={() => setConfirming(false)}>
                      Cancel
                    </Button>
                    <Button variant="destructive" size="sm" onClick={handleLogout}>
                      Confirm
                    </Button>
                  </>
                ) : (
                  <Button variant="outline" size="sm" onClick={() => setConfirming(true)}>
                    Log out
                  </Button>
                )}
              </div>
            </div>
          </section>

        </div>
      </div>
    </div>
  )
}
