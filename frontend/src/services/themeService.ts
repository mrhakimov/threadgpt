import type { Theme } from "@/domain/theme"
import { storageService } from "@/services/storageService"

export function getStoredTheme(): Theme {
  return storageService.getStoredTheme()
}

export function setStoredTheme(theme: Theme): void {
  storageService.setStoredTheme(theme)
}

export function resolveTheme(theme: Theme): boolean {
  if (theme === "dark") {
    return true
  }

  if (theme === "light") {
    return false
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches
}

export function applyTheme(isDark: boolean): void {
  document.documentElement.classList.toggle("dark", isDark)
}
