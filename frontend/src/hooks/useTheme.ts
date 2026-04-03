"use client"

import { useEffect, useState } from "react"
import type { Theme } from "@/domain/theme"
import {
  applyTheme,
  getStoredTheme,
  resolveTheme,
  setStoredTheme,
} from "@/services/themeService"

export type { Theme } from "@/domain/theme"

export function useTheme() {
  const [theme, setThemeState] = useState<Theme>(() => {
    if (typeof window === "undefined") return "system"
    return getStoredTheme()
  })

  const [isDark, setIsDark] = useState(false)

  useEffect(() => {
    const resolved = resolveTheme(theme)
    setIsDark(resolved)
    applyTheme(resolved)
  }, [theme])

  // Listen for OS theme changes when in "system" mode
  useEffect(() => {
    if (theme !== "system") return
    const mq = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = (e: MediaQueryListEvent) => {
      setIsDark(e.matches)
      applyTheme(e.matches)
    }
    mq.addEventListener("change", handler)
    return () => mq.removeEventListener("change", handler)
  }, [theme])

  function setTheme(next: Theme) {
    setStoredTheme(next)
    setThemeState(next)
  }

  return { theme, setTheme, isDark }
}
