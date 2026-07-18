import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
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
import { useResumeDraftContext } from '@/hooks/useResumeDraft'

interface RowCardProps {
  /** Commits the currently edited fields — called only when the user clicks Done. */
  onDone?: () => void
  /** Reverts in-progress edits — called when the panel closes without saving (confirmed
   * discard via X/Escape, or opening a different entry) so the next open starts from the
   * last-saved values again. */
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

// NOTE: X/Escape now confirm before discarding dirty edits (see
// attemptCloseRef below). Switching to a *different* entry while this one is
// dirty still silently discards — that path closes this panel via a prop
// change from the parent's shared open-id state, not a call this component
// makes, so there's no point to intercept a confirmation at.

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
  const { registerPanel } = useResumeDraftContext()
  const panelKey = useId()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmCloseOpen, setConfirmCloseOpen] = useState(false)

  const committedRef = useRef(false)

  // Ctrl/Cmd+S commits without closing — keep a ref so the panel-stack handle
  // (registered once per open, below) always calls the latest onDone/form.
  const commitRef = useRef(() => {})
  commitRef.current = () => {
    committedRef.current = true
    onDone?.()
  }

  // X / Escape close with unsaved form edits go through this instead of
  // closing straight away, so work isn't silently lost — see attemptClose.
  const attemptCloseRef = useRef(() => {})
  attemptCloseRef.current = () => {
    if (isDirty) setConfirmCloseOpen(true)
    else onOpenChange(false)
  }

  useEffect(() => {
    if (!open) return
    return pushOpen({ close: () => attemptCloseRef.current(), commit: () => commitRef.current() })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, pushOpen])

  // Uncommitted form edits are real unsaved work even though they haven't
  // reached the draft yet — register a commit callback so the app-wide
  // "unsaved changes" signal (EditorNavbar's SyncStatus) knows about it, and
  // "Save & leave" (also in EditorNavbar) can commit this panel the same way
  // Ctrl+S already does via the panel stack above.
  useEffect(() => {
    if (open && isDirty) registerPanel(panelKey, () => commitRef.current())
    else registerPanel(panelKey, null)
    return () => registerPanel(panelKey, null)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, isDirty, panelKey, registerPanel])

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

  function discardAndClose() {
    setConfirmCloseOpen(false)
    onOpenChange(false)
  }

  function saveAndClose() {
    setConfirmCloseOpen(false)
    handleDone()
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
                  onClick={() => attemptCloseRef.current()}
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

      <AlertDialog open={confirmCloseOpen} onOpenChange={setConfirmCloseOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Save changes to {typeof title === 'string' && title ? `"${title}"` : 'this entry'}?</AlertDialogTitle>
            <AlertDialogDescription>You have unsaved edits in this entry.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <Button type="button" variant="outline" className="text-destructive hover:text-destructive" onClick={discardAndClose}>
              Discard
            </Button>
            <AlertDialogAction variant="default" onClick={saveAndClose}>
              Save changes
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
