import ChatInput from "@/components/ChatInput"

interface Props {
  error?: string | null
  disabled: boolean
  focusTrigger: number
  placeholder: string
  onSend: (message: string) => void
}

export default function ChatComposer({
  error,
  disabled,
  focusTrigger,
  placeholder,
  onSend,
}: Props) {
  return (
    <div className="shrink-0 border-t px-4 py-3">
      <div className="max-w-3xl mx-auto w-full">
        {error && <p className="text-xs text-destructive mb-2">{error}</p>}
        <ChatInput
          onSend={onSend}
          disabled={disabled}
          focusTrigger={focusTrigger}
          placeholder={placeholder}
        />
      </div>
    </div>
  )
}
