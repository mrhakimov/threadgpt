import { SESSION_PAGE_SIZE } from "@/domain/constants"
import { getSessionLabel } from "@/domain/session"
import type { Session } from "@/domain/entities/chat"
import { sessionRepository } from "@/repositories/sessionRepository"
import { storageService } from "@/services/storageService"

export async function listSessions(limit = SESSION_PAGE_SIZE, offset = 0) {
  return sessionRepository.fetchSessions(limit, offset)
}

export async function renameConversation(
  sessionId: string,
  name: string,
): Promise<string> {
  const trimmedName = name.trim()
  if (!trimmedName || trimmedName.length > 256) {
    return ""
  }

  await sessionRepository.renameSession(sessionId, trimmedName)
  return trimmedName
}

export async function deleteConversation(sessionId: string): Promise<void> {
  await sessionRepository.deleteSession(sessionId)
}

export async function updateConversationSystemPrompt(
  sessionId: string,
  content: string,
): Promise<void> {
  await sessionRepository.updateSystemPrompt(sessionId, content)
}

export function getConversationLabel(session: Session): string {
  return getSessionLabel(session)
}

export function getStoredSidebarState(): boolean {
  return storageService.getSidebarCollapsed()
}

export function setStoredSidebarState(collapsed: boolean): void {
  storageService.setSidebarCollapsed(collapsed)
}
