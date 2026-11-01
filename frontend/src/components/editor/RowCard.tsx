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
import { useResumeDraftContext } from '@/hooks/useResumeDraft'

interface RowCardProps {
  onDelete: () => void
  title: ReactNode
  subtitle?: ReactNode
  dragHandle?: ReactNode
  editorTitle?: string
  open: boolean
  onOpenChange: (open: boolean) => void
  children: ReactNode
}


export default function RowCard({
  onDelete,
  title,
  subtitle,
  dragHandle,
  editorTitle = 'Edit entry',
  open,
  onOpenChange,
  children,
}: RowCardProps) {
  const { slot, pushOpen } = useEditorPanel()
  const { syncStatus } = useResumeDraftContext()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const closeRef = useRef(onOpenChange)
  closeRef.current = onOpenChange

  useEffect(() => {
    if (!open) return
    return pushOpen({ close: () => closeRef.current(false) })
  }, [open, pushOpen])

  function confirmDelete() {
    setConfirmOpen(false)
    onDelete()
    onOpenChange(false)
  }

  return (
    <>
      <div className="workspace-entry-row flex items-center gap-1 p-2">
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
          <div className="workspace-entry-panel absolute inset-0 flex flex-col">
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
              <span className="text-xs text-muted-foreground">
                {syncStatus === 'error' ? 'Changes kept on this device. Retrying sync...' : 'Changes save automatically'}
              </span>
              <Button type="button" onClick={() => onOpenChange(false)}>
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
