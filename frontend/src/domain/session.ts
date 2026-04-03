import type { Session } from "@/domain/entities/chat"

export function getSessionLabel(session: Session): string {
  if (session.name && session.name !== "New conversation") {
    return session.name
  }

  if (session.system_prompt) {
    return session.system_prompt
  }

  return "New conversation"
}
