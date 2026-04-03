import type { ReactNode } from "react"

interface Props {
  href: string
  label: string
  children: ReactNode
}

export default function IconLinkButton({ href, label, children }: Props) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center justify-center h-9 w-9 rounded-md hover:bg-accent hover:text-accent-foreground transition-colors"
      aria-label={label}
    >
      {children}
    </a>
  )
}
