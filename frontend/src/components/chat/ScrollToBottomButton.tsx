import { ChevronDown } from "lucide-react"
import { Button } from "@/components/ui/button"

interface Props {
  visible: boolean
  onClick: () => void
}

export default function ScrollToBottomButton({ visible, onClick }: Props) {
  if (!visible) {
    return null
  }

  return (
    <div className="absolute bottom-24 left-1/2 -translate-x-1/2 z-10">
      <Button
        size="sm"
        className="rounded-full shadow-lg h-8 px-3 gap-1 text-xs bg-background text-foreground border border-border hover:bg-muted"
        onClick={onClick}
      >
        <ChevronDown className="h-3.5 w-3.5" />
        Scroll to bottom
      </Button>
    </div>
  )
}
