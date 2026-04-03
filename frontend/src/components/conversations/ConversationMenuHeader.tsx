import { PanelLeftClose, PanelLeftOpen } from "lucide-react"
import { Button } from "@/components/ui/button"

interface Props {
  collapsed: boolean
  onToggle: () => void
}

export default function ConversationMenuHeader({ collapsed, onToggle }: Props) {
  return (
    <div className="flex items-center border-b px-2 py-3 gap-1">
      <Button
        variant="ghost"
        size="icon"
        className="shrink-0"
        onClick={onToggle}
        aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
      >
        {collapsed ? (
          <PanelLeftOpen className="h-5 w-5" />
        ) : (
          <PanelLeftClose className="h-5 w-5" />
        )}
      </Button>
      {!collapsed && <span className="text-sm font-medium">Conversations</span>}
    </div>
  )
}
