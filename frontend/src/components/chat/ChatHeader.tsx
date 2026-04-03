import { Settings } from "lucide-react"
import { Button } from "@/components/ui/button"
import IconLinkButton from "@/components/shared/IconLinkButton"
import { cn } from "@/lib/utils"

interface Props {
  title: string
  subtitle?: string | null
  onTitleClick?: () => void
  onOpenSettings: () => void
  settingsOpen: boolean
  settingsClosing: boolean
}

export default function ChatHeader({
  title,
  subtitle,
  onTitleClick,
  onOpenSettings,
  settingsOpen,
  settingsClosing,
}: Props) {
  return (
    <header className="shrink-0 border-b px-4 py-3 flex items-center gap-3">
      <h1 className="font-semibold">{title}</h1>

      {subtitle && (
        <button
          onClick={onTitleClick}
          className="text-xs text-muted-foreground truncate max-w-xs hover:text-foreground transition-colors"
        >
          {subtitle}
        </button>
      )}

      <div className="ml-auto flex items-center gap-1">
        <IconLinkButton href="https://x.com/omtiness" label="X (Twitter)">
          <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-4.714-6.231-5.401 6.231H2.744l7.737-8.835L2.25 2.25h6.902l4.263 5.634zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
          </svg>
        </IconLinkButton>

        <IconLinkButton href="https://github.com/mrhakimov/threadgpt" label="GitHub">
          <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M12 2C6.477 2 2 6.477 2 12c0 4.418 2.865 8.166 6.839 9.489.5.092.682-.217.682-.482 0-.237-.009-.868-.013-1.703-2.782.604-3.369-1.342-3.369-1.342-.454-1.155-1.11-1.463-1.11-1.463-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0 1 12 6.836a9.59 9.59 0 0 1 2.504.337c1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.741 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
          </svg>
        </IconLinkButton>

        <Button variant="ghost" size="icon" onClick={onOpenSettings}>
          <Settings
            className={cn(
              "h-4 w-4 transition-transform duration-200",
              settingsOpen && !settingsClosing && "rotate-90",
            )}
          />
        </Button>
      </div>
    </header>
  )
}
