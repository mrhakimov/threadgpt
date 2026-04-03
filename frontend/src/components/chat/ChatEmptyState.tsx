interface Props {
  isFirstMessage: boolean
}

export default function ChatEmptyState({ isFirstMessage }: Props) {
  return (
    <div className="flex flex-col items-center justify-center h-full gap-3 text-center px-4">
      <h2 className="text-lg font-medium">
        {isFirstMessage ? "Set your conversation context" : "Start chatting"}
      </h2>
      <p className="text-sm text-muted-foreground max-w-sm">
        {isFirstMessage
          ? "Your first message becomes the assistant's instructions for this entire conversation. Make it count!"
          : "Send a message to continue your conversation."}
      </p>
    </div>
  )
}
