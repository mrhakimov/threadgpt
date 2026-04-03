import { Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"

interface Props {
  className?: string
}

export default function LoadingSpinner({ className }: Props) {
  return <Loader2 className={cn("animate-spin text-muted-foreground", className)} />
}
