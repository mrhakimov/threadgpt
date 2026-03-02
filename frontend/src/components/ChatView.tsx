"use client"

import { useState } from "react"
import { Message } from "@/types"
import { useChat } from "@/hooks/useChat"
import MessageList from "./MessageList"
import ChatInput from "./ChatInput"
import ThreadDrawer from "./ThreadDrawer"

interface Props {
  apiKey: string
}

export default function ChatView({ apiKey }: Props) {
  const { messages, session, loading, sending, streamingContent, error, sendMessage } =
    useChat(apiKey)
  const [threadParent, setThreadParent] = useState<Message | null>(null)

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-muted-foreground text-sm">Loading...</p>
      </div>
    )
  }

  const isEmpty = messages.length === 0 && !streamingContent
  const isFirstMessage = !session?.assistant_id && !session?.session_id

  return (
    <div className="min-h-screen flex flex-col bg-background">
      {/* Header */}
      <header className="border-b px-4 py-3 flex items-center gap-3">
        <h1 className="font-semibold">ThreadGPT</h1>
        {session?.system_prompt && (
          <span className="text-xs text-muted-foreground truncate max-w-xs">
            Context: {session.system_prompt}
          </span>
        )}
      </header>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 max-w-3xl mx-auto w-full">
        {isEmpty ? (
          <div className="flex flex-col items-center justify-center h-full min-h-[60vh] gap-3 text-center px-4">
            <h2 className="text-lg font-medium">
              {isFirstMessage ? "Set your conversation context" : "Start chatting"}
            </h2>
            <p className="text-sm text-muted-foreground max-w-sm">
              {isFirstMessage
                ? "Your first message becomes the assistant's instructions for this entire conversation. Make it count!"
                : "Send a message to continue your conversation."}
            </p>
          </div>
        ) : (
          <MessageList
            messages={messages}
            streamingContent={streamingContent}
            onReply={setThreadParent}
          />
        )}
      </div>

      {error && (
        <p className="text-xs text-destructive text-center pb-2">{error}</p>
      )}

      {/* Input */}
      <div className="border-t px-4 py-3 max-w-3xl mx-auto w-full">
        {isFirstMessage && (
          <p className="text-xs text-muted-foreground mb-2">
            Your first message sets the context for this entire conversation
          </p>
        )}
        <ChatInput
          onSend={sendMessage}
          disabled={sending}
          placeholder={
            isFirstMessage
              ? "Set the context for your conversation..."
              : "Send a message..."
          }
        />
      </div>

      {/* Thread Drawer */}
      {threadParent && (
        <ThreadDrawer
          apiKey={apiKey}
          parentMessage={threadParent}
          onClose={() => setThreadParent(null)}
        />
      )}
    </div>
  )
}
