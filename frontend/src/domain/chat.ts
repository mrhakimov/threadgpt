import type { Message, Session } from "@/domain/entities/chat"

export function createOptimisticUserMessage(
  content: string,
  sessionId?: string,
): Message {
  return {
    id: crypto.randomUUID(),
    session_id: sessionId ?? "",
    role: "user",
    content,
    created_at: new Date().toISOString(),
  }
}

export function updateSystemPromptInMessages(
  messages: Message[],
  content: string,
): Message[] {
  if (messages.length === 0 || messages[0].role !== "user") {
    return messages
  }

  return [{ ...messages[0], content }, ...messages.slice(1)]
}

export function incrementMessageReplyCount(
  messages: Message[],
  messageId: string,
  by: number,
): Message[] {
  return messages.map((message) =>
    message.id === messageId
      ? { ...message, reply_count: (message.reply_count ?? 0) + by }
      : message,
  )
}

export function isFirstMessageSession(session: Session | null): boolean {
  return !session?.assistant_id
}

export function getChatInputPlaceholder(isFirstMessage: boolean): string {
  return isFirstMessage
    ? "Set the context for your conversation..."
    : "Send a message..."
}
