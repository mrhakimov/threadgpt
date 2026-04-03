import * as chatApi from "@/data/chatApi"
import { sessionRepository } from "@/repositories/sessionRepository"

export const chatRepository = {
  initSession: sessionRepository.initSession,
  fetchSession: sessionRepository.fetchSession,
  fetchHistory: sessionRepository.fetchHistory,
  sendChatMessage: chatApi.sendChatMessage,
}
