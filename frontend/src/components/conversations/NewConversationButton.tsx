import { Plus } from "lucide-react"
import { Button } from "@/components/ui/button"

interface Props {
  collapsed: boolean
  onClick: () => void
}

export default function NewConversationButton({
  collapsed,
  onClick,
}: Props) {
  return (
    <div className="flex items-center px-2 py-1">
      <Button
        variant="outline"
        size="sm"
        onClick={onClick}
        title="New conversation"
        className={`w-full justify-start gap-2 overflow-hidden ${
          collapsed ? "border-transparent bg-transparent shadow-none hover:bg-accent" : ""
        }`}
      >
        <Plus className="h-4 w-4 shrink-0" />
        <span className={collapsed ? "hidden" : "truncate"}>New conversation</span>
      </Button>
    </div>
  )
}
