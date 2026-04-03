import * as threadApi from "@/data/threadApi"

export const threadRepository = {
  fetchThreadMessages: threadApi.fetchThreadMessages,
  sendThreadMessage: threadApi.sendThreadMessage,
}
