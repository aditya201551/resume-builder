import { useEffect, useRef, useState } from 'react'
import { Link, useBlocker } from 'react-router'
import { ArrowLeft, Check, Download, Loader2, Pencil, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import { isDraftDirty } from '@/lib/resumeDraftSync'
import { clearDraft } from '@/lib/resumeDraftStorage'
import { apiDownload } from '@/lib/http'
import { cn } from '@/lib/utils'
import type { Resume } from '@/types/resume'

function EditableResumeName({ resume }: { resume: Resume }) {
  const { dispatch } = useResumeDraftContext()
  const [editing, setEditing] = useState(false)
  const [value, setValue] = useState(resume.label)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (editing) inputRef.current?.select()
  }, [editing])

  function startEditing() {
    setValue(resume.label)
    setEditing(true)
  }

  function commit() {
    const trimmed = value.trim()
    if (trimmed && trimmed !== resume.label) {
      dispatch({ type: 'update_meta', patch: { label: trimmed } })
    }
    setEditing(false)
  }

  function discard() {
    setValue(resume.label)
    setEditing(false)
  }

  if (editing) {
    return (
      <div className="flex min-w-0 items-center gap-1.5">
        <Input
          ref={inputRef}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') commit()
            if (e.key === 'Escape') discard()
          }}
          className="h-8 max-w-72 text-sm font-semibold"
        />
        <Button variant="ghost" size="icon" className="size-8 shrink-0 text-accent" onClick={commit}>
          <Check className="size-4" />
        </Button>
        <Button variant="ghost" size="icon" className="size-8 shrink-0 text-muted-foreground" onClick={discard}>
          <X className="size-4" />
        </Button>
      </div>
    )
  }

  return (
    <button
      type="button"
      onClick={startEditing}
      className="flex w-fit min-w-0 max-w-72 items-center gap-1.5 text-left"
    >
      <span className="truncate text-sm font-semibold text-foreground">{resume.label || 'Untitled resume'}</span>
      <Pencil className="size-3 shrink-0 text-muted-foreground" />
    </button>
  )
}

function DownloadButton({ resume }: { resume: Resume }) {
  const { flushNow, commitDirtyPanels } = useResumeDraftContext()
  const [phase, setPhase] = useState<'idle' | 'saving' | 'generating'>('idle')
  const [error, setError] = useState<string | null>(null)

  async function handleDownload() {
    setError(null)
    try {
      // The export renders whatever the backend already has — commit any
      // open panel's in-progress edits and flush the draft first (same as
      // Ctrl+S/"Save & leave"), so the PDF reflects what's on screen instead
      // of stale, previously-saved data.
      setPhase('saving')
      commitDirtyPanels()
      await new Promise((resolve) => setTimeout(resolve, 0))
      const saved = await flushNow()
      if (!saved) {
        setError("Couldn't save your latest changes — try again before downloading.")
        return
      }
      setPhase('generating')
      await apiDownload(`/api/resumes/${resume.id}/export/pdf`, `${resume.label || 'resume'}.pdf`)
    } catch {
      setError("Couldn't generate PDF")
    } finally {
      setPhase('idle')
    }
  }

  return (
    <div className="flex items-center gap-2">
      {error && <span className="text-xs text-destructive">{error}</span>}
      <Button type="button" variant="secondary" size="sm" disabled={phase !== 'idle'} onClick={handleDownload}>
        {phase !== 'idle' ? <Loader2 className="size-3.5 animate-spin" /> : <Download className="size-3.5" />}
        {phase === 'saving' ? 'Saving…' : phase === 'generating' ? 'Generating…' : 'Download'}
      </Button>
    </div>
  )
}

function SyncStatus() {
  const { state, syncStatus, hasDirtyPanel } = useResumeDraftContext()
  const dirty = isDraftDirty(state.data, state.lastSynced) || hasDirtyPanel
  const label =
    syncStatus === 'syncing'
      ? 'Saving…'
      : syncStatus === 'error'
        ? "Couldn't save — press Ctrl+S to retry"
        : dirty
          ? 'Unsaved changes — press Ctrl+S to save'
          : 'All changes saved'
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 text-xs',
        syncStatus === 'error' ? 'text-destructive' : 'text-muted-foreground',
      )}
    >
      <span
        className={cn(
          'size-1.5 rounded-full',
          syncStatus === 'syncing' && 'animate-pulse bg-accent',
          syncStatus === 'error' && 'bg-destructive',
          syncStatus === 'idle' && (dirty ? 'bg-muted-foreground/50' : 'bg-accent'),
        )}
      />
      {label}
    </span>
  )
}

function BackButton() {
  const { state, resumeId, flushNow, hasDirtyPanel, commitDirtyPanels } = useResumeDraftContext()
  const dirty = isDraftDirty(state.data, state.lastSynced) || hasDirtyPanel
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && currentLocation.pathname !== nextLocation.pathname)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState(false)

  function discardAndLeave() {
    // The draft persists to localStorage on every edit (so a closed tab can
    // recover unsaved work) — that's the right default, but an explicit
    // "discard" means the user wants those edits gone, not just off-screen.
    // Without clearing it here, the next time this resume opens it would
    // hydrate from the same stale local draft and silently bring the
    // "discarded" changes right back. Clearing it forces the next mount to
    // fall through to GET /full instead (see useResumeDraft's hydrate effect).
    clearDraft(resumeId)
    blocker.proceed?.()
  }

  async function saveAndLeave() {
    setSaving(true)
    setSaveError(false)
    // Commit any open panel's in-progress edits first — same thing Ctrl+S
    // does — then wait a tick for that dispatch to land before flushing,
    // otherwise flushNow reads the pre-commit state.
    commitDirtyPanels()
    await new Promise((resolve) => setTimeout(resolve, 0))
    const ok = await flushNow()
    setSaving(false)
    if (ok) {
      blocker.proceed?.()
    } else {
      setSaveError(true)
    }
  }

  return (
    <>
      <Button variant="ghost" size="icon" asChild>
        <Link to="/resumes">
          <ArrowLeft className="size-4" />
        </Link>
      </Button>

      <AlertDialog
        open={blocker.state === 'blocked'}
        onOpenChange={(open) => {
          // AlertDialogAction/Cancel auto-close on click, which would fire
          // this and reset the blocker mid-save — Save & leave is a plain
          // Button (below) specifically so its async work can't race this.
          if (!open && !saving) {
            blocker.reset?.()
            setSaveError(false)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Save changes before leaving?</AlertDialogTitle>
            <AlertDialogDescription>
              You have unsaved changes that haven&apos;t been saved to the server yet.
              {saveError && (
                <span className="mt-1.5 block text-destructive">Couldn&apos;t save — check your connection and try again.</span>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={saving} onClick={() => blocker.reset?.()}>
              Cancel
            </AlertDialogCancel>
            <Button
              type="button"
              variant="outline"
              disabled={saving}
              className="text-destructive hover:text-destructive"
              onClick={discardAndLeave}
            >
              Discard &amp; leave
            </Button>
            <Button type="button" disabled={saving} onClick={saveAndLeave}>
              {saving && <Loader2 className="size-3.5 animate-spin" />}
              {saving ? 'Saving…' : 'Save & leave'}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

export type EditorMode = 'content' | 'design' | 'chat'

const NAV_ITEMS: { key: EditorMode; label: string }[] = [
  { key: 'content', label: 'Content' },
  { key: 'design', label: 'Design' },
  { key: 'chat', label: 'Chat' },
]

function EditorNavItems({ mode, onModeChange }: { mode: EditorMode; onModeChange: (mode: EditorMode) => void }) {
  return (
    <div className="flex items-center gap-1 rounded-lg bg-secondary/60 p-1">
      {NAV_ITEMS.map((item) => (
        <button
          key={item.key}
          type="button"
          onClick={() => onModeChange(item.key)}
          className={cn(
            'rounded-md px-2.5 py-1.5 text-sm font-medium text-muted-foreground hover:text-foreground',
            mode === item.key && 'bg-background text-foreground shadow-sm',
          )}
        >
          {item.label}
        </button>
      ))}
    </div>
  )
}

export default function EditorNavbar({
  resume,
  mode,
  onModeChange,
}: {
  resume: Resume
  mode: EditorMode
  onModeChange: (mode: EditorMode) => void
}) {
  return (
    <div className="flex shrink-0 items-center gap-3 border-b border-border bg-background px-6 py-3">
      <BackButton />

      <EditorNavItems mode={mode} onModeChange={onModeChange} />

      <div className="flex-1" />

      <EditableResumeName resume={resume} />

      <DownloadButton resume={resume} />

      <SyncStatus />
    </div>
  )
}
