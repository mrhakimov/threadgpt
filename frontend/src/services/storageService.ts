import {
  AUTH_STORAGE_KEY,
  SESSION_STORAGE_KEY,
  SIDEBAR_COLLAPSED_STORAGE_KEY,
  THEME_STORAGE_KEY,
} from "@/domain/constants"
import type { Theme } from "@/domain/theme"

function getStorage(storage: "local" | "session"): Storage | null {
  if (typeof window === "undefined") {
    return null
  }

  return storage === "local" ? window.localStorage : window.sessionStorage
}

export const storageService = {
  getSelectedSessionId(): string | null {
    return getStorage("session")?.getItem(SESSION_STORAGE_KEY) ?? null
  },

  setSelectedSessionId(sessionId: string | null): void {
    const storage = getStorage("session")
    if (!storage) {
      return
    }

    if (sessionId) {
      storage.setItem(SESSION_STORAGE_KEY, sessionId)
      return
    }

    storage.removeItem(SESSION_STORAGE_KEY)
  },

  hasPersistedAuth(): boolean {
    return getStorage("local")?.getItem(AUTH_STORAGE_KEY) === "1"
  },

  persistAuth(): void {
    getStorage("local")?.setItem(AUTH_STORAGE_KEY, "1")
  },

  clearPersistedAuth(): void {
    getStorage("local")?.removeItem(AUTH_STORAGE_KEY)
  },

  getStoredTheme(): Theme {
    return (getStorage("local")?.getItem(THEME_STORAGE_KEY) as Theme) ?? "system"
  },

  setStoredTheme(theme: Theme): void {
    getStorage("local")?.setItem(THEME_STORAGE_KEY, theme)
  },

  getSidebarCollapsed(): boolean {
    return getStorage("local")?.getItem(SIDEBAR_COLLAPSED_STORAGE_KEY) === "true"
  },

  setSidebarCollapsed(collapsed: boolean): void {
    getStorage("local")?.setItem(
      SIDEBAR_COLLAPSED_STORAGE_KEY,
      String(collapsed),
    )
  },
}
