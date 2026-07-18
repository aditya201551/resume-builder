import { useEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { Trash2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useEditorPanel } from '@/hooks/useEditorPanel'

interface RowCardProps {
  /** Commits the currently edited fields — called only when the user clicks Done. */
  onDone?: () => void
  /** Reverts in-progress edits — called when the panel closes any other way (X, Escape,
   * opening a different entry) so the next open starts from the last-saved values again. */
  onDiscard?: () => void
  isDirty?: boolean
  onDelete: () => void
  title: ReactNode
  subtitle?: ReactNode
  dragHandle?: ReactNode
  editorTitle?: string
  /** True for an entry that was just created and hasn't been closed once yet — if it's
   * closed any way other than Done (X, Escape, opening another entry), the add is
   * treated as cancelled and the entry is discarded instead of left empty. */
  isNew?: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
  children: ReactNode
}

export default function RowCard({
  onDone,
  onDiscard,
  isDirty = false,
  onDelete,
  title,
  subtitle,
  dragHandle,
  editorTitle = 'Edit entry',
  isNew = false,
  open,
  onOpenChange,
  children,
}: RowCardProps) {
  const { slot, pushOpen } = useEditorPanel()
  const [confirmOpen, setConfirmOpen] = useState(false)

  const committedRef = useRef(false)

  // Ctrl/Cmd+S commits without closing — keep a ref so the panel-stack handle
  // (registered once per open, below) always calls the latest onDone/form.
  const commitRef = useRef(() => {})
  commitRef.current = () => {
    committedRef.current = true
    onDone?.()
  }

  useEffect(() => {
    if (!open) return
    return pushOpen({ close: () => onOpenChange(false), commit: () => commitRef.current() })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, pushOpen])

  const wasOpenRef = useRef(open)
  useEffect(() => {
    const wasOpen = wasOpenRef.current
    wasOpenRef.current = open
    if (!wasOpen || open) return
    const wasCommitted = committedRef.current
    committedRef.current = false
    if (wasCommitted) return
    // Closed without clicking Done — a fresh, never-saved entry is a cancelled
    // add and goes away entirely; an existing one just reverts its edits.
    if (isNew) {
      onDelete()
    } else {
      onDiscard?.()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  function handleDone() {
    commitRef.current()
    onOpenChange(false)
  }

  function handleDelete() {
    committedRef.current = true
    onDelete()
    onOpenChange(false)
  }

  function confirmDelete() {
    setConfirmOpen(false)
    handleDelete()
  }

  return (
    <>
      <div className="flex items-center gap-1 rounded-md border border-border bg-card p-2">
        {dragHandle && <span className="shrink-0">{dragHandle}</span>}
        <button
          type="button"
          onClick={() => onOpenChange(true)}
          className="flex min-w-0 flex-1 items-center gap-2 rounded-md p-2 text-left"
        >
          <span className="min-w-0 flex-1 truncate text-sm">
            <span className="font-semibold">{title || 'Untitled'}</span>
            {subtitle && <span className="text-muted-foreground">, {subtitle}</span>}
          </span>
        </button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 shrink-0 text-muted-foreground hover:text-destructive"
          aria-label="Delete"
          onClick={() => setConfirmOpen(true)}
        >
          <Trash2 className="size-3.5" />
        </Button>
      </div>

      {open &&
        slot &&
        createPortal(
          <div className="absolute inset-0 flex flex-col bg-background">
            <div className="flex items-center justify-between gap-2 border-b border-border p-4">
              <h2 className="text-lg font-bold">{editorTitle}</h2>
              <div className="flex items-center gap-1">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="text-muted-foreground hover:text-destructive"
                  aria-label="Delete entry"
                  onClick={() => setConfirmOpen(true)}
                >
                  <Trash2 className="size-4" />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label="Close editor"
                  onClick={() => onOpenChange(false)}
                >
                  <X className="size-4" />
                </Button>
              </div>
            </div>
            <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">{children}</div>
            <div className="flex items-center justify-between gap-2 border-t border-border p-4">
              <span className="text-xs text-muted-foreground">{isDirty ? 'Unsaved changes' : ''}</span>
              <Button type="button" onClick={handleDone}>
                Done
              </Button>
            </div>
          </div>,
          slot,
        )}

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {typeof title === 'string' && title ? `"${title}"` : 'this entry'}?</AlertDialogTitle>
            <AlertDialogDescription>This can&apos;t be undone.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={confirmDelete}>Delete</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
