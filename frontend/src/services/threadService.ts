import { MESSAGE_PAGE_SIZE } from "@/domain/constants"
import type { HistoryPage } from "@/domain/entities/chat"
import { threadRepository } from "@/repositories/threadRepository"

export async function loadThreadHistory(
  parentMessageId: string,
): Promise<HistoryPage> {
  return threadRepository.fetchThreadMessages(
    parentMessageId,
    MESSAGE_PAGE_SIZE,
    0,
  )
}

export async function loadOlderThreadMessages(
  parentMessageId: string,
  loadedCount: number,
): Promise<HistoryPage> {
  return threadRepository.fetchThreadMessages(
    parentMessageId,
    MESSAGE_PAGE_SIZE,
    loadedCount,
  )
}

export async function streamThreadTurn(
  parentMessageId: string,
  userMessage: string,
  onChunk: (chunk: string) => void,
  signal?: AbortSignal,
) {
  return threadRepository.sendThreadMessage(parentMessageId, userMessage, onChunk, signal)
}
