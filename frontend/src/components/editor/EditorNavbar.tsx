import { useEffect, useRef, useState } from 'react'
import { Link, useBlocker } from 'react-router'
import { ArrowLeft, Check, Download, Loader2, Pencil } from 'lucide-react'
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
import { apiDownload } from '@/lib/http'
import { cn } from '@/lib/utils'
import type { Resume } from '@/types/resume'

function EditableResumeName({ resume }: { resume: Resume }) {
  const { dispatch } = useResumeDraftContext()
  const [editing, setEditing] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (editing) inputRef.current?.select()
  }, [editing])

  function startEditing() {
    setEditing(true)
  }

  if (editing) {
    return (
      <div className="flex min-w-0 items-center gap-1.5">
        <Input
          ref={inputRef}
          value={resume.label}
          onChange={(e) => dispatch({ type: 'update_meta', patch: { label: e.target.value } })}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === 'Escape') setEditing(false)
          }}
          className="h-8 max-w-72 text-sm font-semibold"
        />
        <Button variant="ghost" size="icon" className="size-8 shrink-0 text-accent" onClick={() => setEditing(false)}>
          <Check className="size-4" />
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
  const { flushNow } = useResumeDraftContext()
  const [phase, setPhase] = useState<'idle' | 'saving' | 'generating'>('idle')
  const [error, setError] = useState<string | null>(null)

  async function handleDownload() {
    setError(null)
    try {
      setPhase('saving')
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
  const { state, syncStatus } = useResumeDraftContext()
  const dirty = isDraftDirty(state.data, state.lastSynced)
  const label = syncStatus === 'error'
    ? "Couldn't sync ? retrying automatically"
    : syncStatus === 'syncing' || dirty ? 'Syncing?' : 'All changes saved'
  return (
    <span role="status" className={cn('inline-flex items-center gap-1.5 text-xs', syncStatus === 'error' ? 'text-destructive' : 'text-muted-foreground')}>
      <span className={cn('size-1.5 rounded-full', syncStatus === 'error' ? 'bg-destructive' : dirty ? 'animate-pulse bg-muted-foreground/50' : 'bg-accent')} />
      {label}
    </span>
  )
}

function BackButton() {
  const { state, flushNow } = useResumeDraftContext()
  const dirty = isDraftDirty(state.data, state.lastSynced)
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && currentLocation.pathname !== nextLocation.pathname)
  const [saving, setSaving] = useState(false)
  const [syncFailed, setSyncFailed] = useState(false)

  async function syncAndLeave() {
    setSaving(true)
    const ok = await flushNow()
    setSaving(false)
    if (ok) blocker.proceed?.()
    else setSyncFailed(true)
  }

  useEffect(() => {
    if (blocker.state !== 'blocked') {
      setSyncFailed(false)
      return
    }
    void syncAndLeave()
    // Flush automatically when navigating; a dialog is needed only if syncing fails.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [blocker.state])

  return (
    <>
      <Button variant="ghost" size="icon" asChild>
        <Link to="/resumes" aria-label="Back to resumes">
          {saving ? <Loader2 className="size-4 animate-spin" /> : <ArrowLeft className="size-4" />}
        </Link>
      </Button>
      <AlertDialog open={blocker.state === 'blocked' && syncFailed} onOpenChange={(open) => {
        if (!open && !saving) blocker.reset?.()
      }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Changes haven't synced yet</AlertDialogTitle>
            <AlertDialogDescription>
              Your changes are kept on this device. You can retry now or leave and resume syncing when you reopen this resume.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={saving} onClick={() => blocker.reset?.()}>Stay</AlertDialogCancel>
            <Button type="button" variant="outline" disabled={saving} onClick={() => blocker.proceed?.()}>Leave for now</Button>
            <Button type="button" disabled={saving} onClick={syncAndLeave}>
              {saving && <Loader2 className="size-3.5 animate-spin" />} Retry
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
