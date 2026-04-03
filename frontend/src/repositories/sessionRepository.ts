import * as sessionApi from "@/data/sessionApi"

export const sessionRepository = {
  initSession: sessionApi.initSession,
  fetchSession: sessionApi.fetchSession,
  fetchSessions: sessionApi.fetchSessions,
  createSession: sessionApi.createSession,
  renameSession: sessionApi.renameSession,
  updateSystemPrompt: sessionApi.updateSystemPrompt,
  deleteSession: sessionApi.deleteSession,
  fetchHistory: sessionApi.fetchHistory,
}
