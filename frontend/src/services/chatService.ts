import { MESSAGE_PAGE_SIZE } from "@/domain/constants"
import {
  createOptimisticUserMessage,
  getChatInputPlaceholder,
  incrementMessageReplyCount,
  isFirstMessageSession,
  updateSystemPromptInMessages,
} from "@/domain/chat"
import type {
  ConversationHistoryPage,
  Message,
  Session,
} from "@/domain/entities/chat"
import { chatRepository } from "@/repositories/chatRepository"

export { getChatInputPlaceholder, isFirstMessageSession }

export const INITIAL_CHAT_CONFIRMATION =
  "Context set! Your assistant has been configured with this as its instructions. Send your next message to start chatting."

export interface ChatStateSnapshot {
  messages: Message[]
  hasMoreMessages: boolean
  session: Session | null
  loadedConversationCount: number
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

export interface SendChatTurnResult extends ChatStateSnapshot {
  resolvedSessionId?: string
}

export interface ChatPageLoadResult {
  messages: Message[]
  has_more: boolean
  loadedConversationCount: number
}

export async function loadChatSession(
  requestedSessionId?: string | null,
): Promise<ChatLoadResult> {
  if (requestedSessionId === null) {
    return {
      messages: [],
      hasMoreMessages: false,
      session: null,
      loadedConversationCount: 0,
    }
  }

  if (requestedSessionId) {
    const [history, session] = await Promise.all([
      chatRepository.fetchHistory(requestedSessionId, MESSAGE_PAGE_SIZE, 0),
      chatRepository.fetchSession(requestedSessionId),
    ])

    return {
      messages: renderConversationHistory(session, history),
      hasMoreMessages: history.has_more,
      session: {
        session_id: requestedSessionId,
        is_new: false,
        name: session.name,
        system_prompt: session.system_prompt,
        assistant_id: session.assistant_id,
        created_at: session.created_at,
      },
      loadedConversationCount: history.conversations.length,
    }
  }

  const session = await chatRepository.initSession()
  if (session.is_new || !session.session_id) {
    return {
      messages: [],
      hasMoreMessages: false,
      session,
      loadedConversationCount: 0,
    }
  }

  const history = await chatRepository.fetchHistory(undefined, MESSAGE_PAGE_SIZE, 0)

  return {
    messages: renderConversationHistory(session, history),
    hasMoreMessages: history.has_more,
    session,
    resolvedSessionId: session.session_id,
    loadedConversationCount: history.conversations.length,
  }
}

export async function loadOlderChatMessages(
  session: Session,
  loadedConversationCount: number,
): Promise<ChatPageLoadResult> {
  const history = await chatRepository.fetchHistory(
    session.session_id,
    MESSAGE_PAGE_SIZE,
    loadedConversationCount,
  )

  return {
    messages: renderConversationPreviews(history, session.session_id ?? ""),
    has_more: history.has_more,
    loadedConversationCount: history.conversations.length,
  }
}

export async function loadCompleteChatHistory(
  session: Session,
): Promise<ChatPageLoadResult> {
  const history = await chatRepository.fetchHistory(session.session_id, 10000, 0)

  return {
    messages: renderConversationHistory(session, history),
    has_more: false,
    loadedConversationCount: history.conversations.length,
  }
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

  const needsFreshSession =
    !!resolvedSessionId &&
    (
      resolvedSessionId !== (requestedSessionId || currentSession?.session_id) ||
      !currentSession?.system_prompt
    )

  const session = resolvedSessionId && needsFreshSession
    ? await chatRepository.fetchSession(resolvedSessionId)
    : currentSession

  const normalizedSession = session
    ? {
        session_id: resolvedSessionId ?? session.session_id,
        is_new: false,
        name: session.name,
        system_prompt: session.system_prompt,
        assistant_id: session.assistant_id,
        created_at: session.created_at,
      }
    : currentSession

  return {
    messages: renderConversationHistory(normalizedSession, history),
    hasMoreMessages: history.has_more,
    loadedConversationCount: history.conversations.length,
    session: normalizedSession,
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

function renderConversationHistory(
  session: Session | null,
  history: ConversationHistoryPage,
): Message[] {
  const sessionId = session?.session_id ?? ""
  const messages: Message[] = []

  if (session?.system_prompt) {
    messages.push({
      id: `system-prompt:${sessionId}`,
      session_id: sessionId,
      role: "user",
      content: session.system_prompt,
      created_at: session.created_at ?? "",
    })

    messages.push({
      id: `system-prompt-confirmation:${sessionId}`,
      session_id: sessionId,
      role: "assistant",
      content: INITIAL_CHAT_CONFIRMATION,
      created_at: session.created_at ?? "",
    })
  }

  return messages.concat(renderConversationPreviews(history, sessionId))
}

function renderConversationPreviews(
  history: ConversationHistoryPage,
  sessionId: string,
): Message[] {
  return history.conversations.flatMap((conversation) => ([
    {
      id: `${conversation.conversation_id}:user`,
      session_id: sessionId || conversation.session_id,
      role: "user" as const,
      content: conversation.user_message,
      created_at: conversation.created_at,
    },
    {
      id: conversation.conversation_id,
      session_id: sessionId || conversation.session_id,
      role: "assistant" as const,
      content: conversation.assistant_message,
      reply_count: conversation.reply_count,
      created_at: conversation.created_at,
    },
  ]))
}
