import {
  MESSAGE_PAGE_SIZE,
} from "@/domain/constants"
import {
  createOptimisticUserMessage,
  getChatInputPlaceholder,
  incrementMessageReplyCount,
  isFirstMessageSession,
  updateSystemPromptInMessages,
} from "@/domain/chat"
import type { HistoryPage, Message, Session } from "@/domain/entities/chat"
import { chatRepository } from "@/repositories/chatRepository"

export { getChatInputPlaceholder, isFirstMessageSession }

export interface ChatStateSnapshot {
  messages: Message[]
  hasMoreMessages: boolean
  session: Session | null
}

export interface ChatLoadResult extends ChatStateSnapshot {
  resolvedSessionId?: string
}

export interface SendChatTurnParams {
  content: string
  requestedSessionId?: string | null
  currentSession: Session | null
  onChunk: (chunk: string) => void
  signal?: AbortSignal
}

export interface SendChatTurnResult {
  history: HistoryPage
  session: Session | null
  resolvedSessionId?: string
}

export async function loadChatSession(
  requestedSessionId?: string | null,
): Promise<ChatLoadResult> {
  if (requestedSessionId === null) {
    return {
      messages: [],
      hasMoreMessages: false,
      session: null,
    }
  }

  if (requestedSessionId) {
    const [history, session] = await Promise.all([
      chatRepository.fetchHistory(requestedSessionId, MESSAGE_PAGE_SIZE, 0),
      chatRepository.fetchSession(requestedSessionId),
    ])

    return {
      messages: history.messages,
      hasMoreMessages: history.has_more,
      session: {
        session_id: requestedSessionId,
        is_new: false,
        name: session.name,
        system_prompt: session.system_prompt,
        assistant_id: session.assistant_id,
      },
    }
  }

  const session = await chatRepository.initSession()
  if (session.is_new || !session.session_id) {
    return {
      messages: [],
      hasMoreMessages: false,
      session,
    }
  }

  const history = await chatRepository.fetchHistory(undefined, MESSAGE_PAGE_SIZE, 0)

  return {
    messages: history.messages,
    hasMoreMessages: history.has_more,
    session,
    resolvedSessionId: session.session_id,
  }
}

export async function loadOlderChatMessages(
  sessionId: string,
  loadedCount: number,
): Promise<HistoryPage> {
  return chatRepository.fetchHistory(sessionId, MESSAGE_PAGE_SIZE, loadedCount)
}

export async function loadCompleteChatHistory(
  sessionId: string,
): Promise<HistoryPage> {
  return chatRepository.fetchHistory(sessionId, 10000, 0)
}

export async function sendChatTurn({
  content,
  requestedSessionId,
  currentSession,
  onChunk,
  signal,
}: SendChatTurnParams): Promise<SendChatTurnResult> {
  const forceNew = requestedSessionId === null
  const activeSessionId = forceNew
    ? undefined
    : requestedSessionId || currentSession?.session_id || undefined

  const streamedSessionId = await chatRepository.sendChatMessage(
    content,
    onChunk,
    activeSessionId,
    forceNew,
    signal,
  )

  const resolvedSessionId = streamedSessionId ?? activeSessionId
  const history = await chatRepository.fetchHistory(
    resolvedSessionId,
    MESSAGE_PAGE_SIZE,
    0,
  )

  if (!resolvedSessionId) {
    return {
      history,
      session: currentSession,
    }
  }

  if (resolvedSessionId !== (requestedSessionId || currentSession?.session_id)) {
    return {
      history,
      session: { session_id: resolvedSessionId, is_new: false },
      resolvedSessionId,
    }
  }

  if (!currentSession?.assistant_id) {
    const session = await chatRepository.fetchSession(resolvedSessionId)
    return {
      history,
      session: {
        session_id: resolvedSessionId,
        is_new: false,
        name: session.name,
        system_prompt: session.system_prompt,
        assistant_id: session.assistant_id,
      },
      resolvedSessionId,
    }
  }

  return {
    history,
    session: currentSession,
    resolvedSessionId,
  }
}

export function buildOptimisticChatMessage(
  content: string,
  sessionId?: string,
): Message {
  return createOptimisticUserMessage(content, sessionId)
}

export function applySystemPromptLocally(
  messages: Message[],
  content: string,
): Message[] {
  return updateSystemPromptInMessages(messages, content)
}

export function incrementLocalReplyCount(
  messages: Message[],
  messageId: string,
  by: number,
): Message[] {
  return incrementMessageReplyCount(messages, messageId, by)
}
