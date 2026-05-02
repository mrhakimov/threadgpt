import { MESSAGE_PAGE_SIZE } from "@/domain/constants"
import type { HistoryPage } from "@/domain/entities/chat"
import { threadRepository } from "@/repositories/threadRepository"

export async function loadThreadHistory(
  conversationId: string,
): Promise<HistoryPage> {
  return threadRepository.fetchThreadMessages(
    conversationId,
    MESSAGE_PAGE_SIZE,
    0,
  )
}

export async function loadOlderThreadMessages(
  conversationId: string,
  loadedCount: number,
): Promise<HistoryPage> {
  return threadRepository.fetchThreadMessages(
    conversationId,
    MESSAGE_PAGE_SIZE,
    loadedCount,
  )
}

export async function streamThreadTurn(
  conversationId: string,
  userMessage: string,
  onChunk: (chunk: string) => void,
  signal?: AbortSignal,
) {
  return threadRepository.sendThreadMessage(conversationId, userMessage, onChunk, signal)
}
