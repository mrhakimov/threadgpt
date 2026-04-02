"use client"

import { useEffect, useState } from "react"

export type Theme = "system" | "light" | "dark"

function resolveTheme(theme: Theme): boolean {
  if (theme === "dark") return true
  if (theme === "light") return false
  return window.matchMedia("(prefers-color-scheme: dark)").matches
}

function applyTheme(isDark: boolean) {
  document.documentElement.classList.toggle("dark", isDark)
}

export function useTheme() {
  const [theme, setThemeState] = useState<Theme>(() => {
    if (typeof window === "undefined") return "system"
    return (localStorage.getItem("theme") as Theme) ?? "system"
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
    localStorage.setItem("theme", next)
    setThemeState(next)
  }

  return { theme, setTheme, isDark }
}
