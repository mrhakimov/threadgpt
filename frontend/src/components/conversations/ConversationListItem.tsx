import type {
  KeyboardEvent,
  MouseEvent,
  RefObject,
} from "react"
import { Check, Pencil, Trash2 } from "lucide-react"
import type { Session } from "@/domain/entities/chat"

interface Props {
  session: Session
  active: boolean
  label: string
  editingId: string | null
  editingName: string
  confirmDeleteId: string | null
  editInputRef: RefObject<HTMLInputElement | null>
  onSelect: (sessionId: string | null) => void
  onStartEditing: (session: Session, event: MouseEvent) => void
  onEditingNameChange: (value: string) => void
  onEditKeyDown: (event: KeyboardEvent<HTMLInputElement>, sessionId: string) => void
  onCommitRename: (sessionId: string) => void
  onDeleteClick: (session: Session, event: MouseEvent) => void
  onConfirmDelete: (sessionId: string) => void
  onCancelDelete: (event: MouseEvent) => void
}

export default function ConversationListItem({
  session,
  active,
  label,
  editingId,
  editingName,
  confirmDeleteId,
  editInputRef,
  onSelect,
  onStartEditing,
  onEditingNameChange,
  onEditKeyDown,
  onCommitRename,
  onDeleteClick,
  onConfirmDelete,
  onCancelDelete,
}: Props) {
  const sessionId = session.session_id ?? null

  return (
    <div
      className={`group w-full text-left rounded-md px-2 py-2 text-sm flex items-center gap-2 hover:bg-muted transition-colors ${
        active ? "bg-muted font-medium" : ""
      }`}
    >
      {editingId === sessionId ? (
        <>
          <input
            ref={editInputRef}
            className="flex-1 min-w-0 bg-transparent outline-none text-sm"
            value={editingName}
            onChange={(event) => onEditingNameChange(event.target.value)}
            onKeyDown={(event) => onEditKeyDown(event, sessionId!)}
            onBlur={() => onCommitRename(sessionId!)}
          />
          <button
            className="shrink-0 text-muted-foreground hover:text-foreground"
            onMouseDown={(event) => {
              event.preventDefault()
              onCommitRename(sessionId!)
            }}
          >
            <Check className="h-3.5 w-3.5" />
          </button>
        </>
      ) : (
        <>
          <button className="flex-1 min-w-0 text-left" onClick={() => onSelect(sessionId)}>
            <span className="block truncate">{label}</span>
          </button>

          {confirmDeleteId === sessionId ? (
            <div className="shrink-0 flex items-center gap-1">
              <span className="text-xs text-muted-foreground">Delete?</span>
              <button
                className="px-1.5 py-0.5 rounded text-xs bg-destructive text-destructive-foreground hover:bg-destructive/90"
                onClick={(event) => {
                  event.stopPropagation()
                  onConfirmDelete(sessionId!)
                }}
              >
                Yes
              </button>
              <button
                className="px-1.5 py-0.5 rounded text-xs hover:bg-accent text-muted-foreground"
                onClick={onCancelDelete}
              >
                No
              </button>
            </div>
          ) : (
            <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
              <button
                className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-accent"
                onClick={(event) => onStartEditing(session, event)}
                title="Rename"
              >
                <Pencil className="h-3.5 w-3.5" />
              </button>
              <button
                className="p-1 rounded text-muted-foreground hover:text-destructive hover:bg-accent"
                onClick={(event) => onDeleteClick(session, event)}
                title="Delete"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
