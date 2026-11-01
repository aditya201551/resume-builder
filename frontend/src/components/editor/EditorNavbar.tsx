import { useEffect, useRef, useState } from 'react'
import { Link, useBlocker } from 'react-router'
import * as m from 'motion/react-m'
import {
  ArrowLeft,
  Check,
  Download,
  FilePenLine,
  Loader2,
  Palette,
  Pencil,
  Sparkles,
} from 'lucide-react'
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
import { useResumePdf } from '@/hooks/useResumePdf'
import { downloadPdfBlob } from '@/lib/pdf/download'
import { cn } from '@/lib/utils'
import { useMediaQuery } from '@/hooks/useMediaQuery'
import Brand from '@/components/layout/Brand'
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
          onChange={(e) =>
            dispatch({ type: 'update_meta', patch: { label: e.target.value } })
          }
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === 'Escape') setEditing(false)
          }}
          className="h-8 max-w-72 text-sm font-semibold"
        />
        <Button
          variant="ghost"
          size="icon"
          className="size-8 shrink-0 text-accent"
          onClick={() => setEditing(false)}
        >
          <Check className="size-4" />
        </Button>
      </div>
    )
  }

  return (
    <button
      type="button"
      onClick={startEditing}
      aria-label="Rename resume"
      className="workspace-rename flex w-fit min-w-0 max-w-48 items-center gap-1.5 rounded-md text-left"
    >
      <span className="truncate text-sm font-semibold text-foreground">
        {resume.label || 'Untitled resume'}
      </span>
      <Pencil className="size-3 shrink-0 text-muted-foreground" />
    </button>
  )
}

function DownloadButton({ resume }: { resume: Resume }) {
  const { getLatest } = useResumePdf()
  const [phase, setPhase] = useState<'idle' | 'generating'>('idle')
  const [error, setError] = useState<string | null>(null)

  async function handleDownload() {
    setError(null)
    try {
      setPhase('generating')
      const pdf = await getLatest()
      downloadPdfBlob(pdf.blob, resume.label)
    } catch (error) {
      setError(error instanceof Error ? error.message : "Couldn't generate PDF")
    } finally {
      setPhase('idle')
    }
  }

  return (
    <div className="workspace-export flex items-center gap-2">
      {error && (
        <span className="max-w-40 text-xs text-destructive" role="alert">
          {error}
        </span>
      )}
      <Button
        type="button"
        size="sm"
        className="workspace-export-button h-9 rounded-xl px-3"
        aria-label={phase === 'generating' ? 'Generating PDF' : 'Export PDF'}
        title="Export PDF"
        disabled={phase !== 'idle'}
        onClick={handleDownload}
      >
        {phase !== 'idle' ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <Download className="size-3.5" />
        )}
        <span className="workspace-export-label">{phase === 'generating'
            ? 'Generating…'
            : 'Export PDF'}</span>
      </Button>
    </div>
  )
}

function SyncStatus() {
  const { state, syncStatus } = useResumeDraftContext()
  const dirty = isDraftDirty(state.data, state.lastSynced)
  const label =
    syncStatus === 'error'
      ? "Couldn't sync — retrying automatically"
      : syncStatus === 'syncing' || dirty
        ? 'Syncing…'
        : 'All changes saved'
  return (
    <span
      role="status"
      className={cn(
        'workspace-sync inline-flex items-center gap-1.5 text-[10px]',
        syncStatus === 'error' ? 'text-destructive' : 'text-muted-foreground',
      )}
    >
        {syncStatus === 'syncing' || dirty ? (
          <Loader2 className="size-3 animate-spin" aria-hidden />
        ) : syncStatus !== 'error' ? (
          <Check className="size-3" aria-hidden />
        ) : null}
      <span className="workspace-sync-label">{label}</span>
      <span className="workspace-sync-compact">{syncStatus === 'error' ? 'Sync issue' : syncStatus === 'syncing' || dirty ? 'Syncing…' : 'Saved'}</span>
    </span>
  )
}

function BackButton() {
  const { state, flushNow } = useResumeDraftContext()
  const dirty = isDraftDirty(state.data, state.lastSynced)
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      dirty && currentLocation.pathname !== nextLocation.pathname,
  )
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
          {saving ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <ArrowLeft className="size-4" />
          )}
        </Link>
      </Button>
      <AlertDialog
        open={blocker.state === 'blocked' && syncFailed}
        onOpenChange={(open) => {
          if (!open && !saving) blocker.reset?.()
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Changes haven’t synced yet</AlertDialogTitle>
            <AlertDialogDescription>
              Your changes are kept on this device. You can retry now or leave
              and resume syncing when you reopen this resume.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel
              disabled={saving}
              onClick={() => blocker.reset?.()}
            >
              Stay
            </AlertDialogCancel>
            <Button
              type="button"
              variant="outline"
              disabled={saving}
              onClick={() => blocker.proceed?.()}
            >
              Leave for now
            </Button>
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

const NAV_ITEMS = [
  { key: 'chat', label: 'AI workspace', icon: Sparkles },
  { key: 'content', label: 'Content', icon: FilePenLine },
  { key: 'design', label: 'Design', icon: Palette },
] as const

function EditorNavItems({
  mode,
  onModeChange,
  mobilePreview,
  onMobilePreviewChange,
}: {
  mode: EditorMode
  onModeChange: (mode: EditorMode) => void
  mobilePreview: boolean
  onMobilePreviewChange: (preview: boolean) => void
}) {
  const mobile = useMediaQuery('(max-width: 899px)')
  const index = NAV_ITEMS.findIndex((item) => item.key === mode)
  return (
    <nav className="workspace-mode-picker" aria-label="Editor mode">
      <m.span
        className="workspace-mode-indicator"
        initial={false}
        animate={{ x: `${index * 100}%` }}
        transition={{ duration: 0.22, ease: [0.2, 0, 0.38, 0.9] }}
        aria-hidden
      />
      {NAV_ITEMS.map(({ key, label, icon: Icon }) => (
        <button
          key={key}
          type="button"
          className="workspace-mode-button"
          aria-pressed={mode === key && !(mobile && mobilePreview)}
          aria-label={label}
          onClick={() => { onMobilePreviewChange(false); onModeChange(key) }}
        >
          <Icon className="size-3.5" aria-hidden />
          <span className="workspace-mode-label">{label}</span>
          <span className="workspace-mode-short">{key === 'chat' ? 'AI' : label}</span>
        </button>
      ))}
      <button
        type="button"
        className="workspace-mode-button workspace-mobile-preview"
        aria-pressed={mobilePreview}
        onClick={() => onMobilePreviewChange(true)}
      >
        Preview
      </button>
    </nav>
  )
}

export default function EditorNavbar({
  resume,
  mode,
  onModeChange,
  mobilePreview,
  onMobilePreviewChange,
}: {
  resume: Resume
  mode: EditorMode
  onModeChange: (mode: EditorMode) => void
  mobilePreview: boolean
  onMobilePreviewChange: (preview: boolean) => void
}) {
  return (
    <header className="workspace-navbar">
      <div className="workspace-back">
        <BackButton />
        <Link
          to="/resumes"
          className="brand-link"
          aria-label="Resume Builder home"
        >
          <Brand compact />
        </Link>
        <div className="workspace-resume-info min-w-0">
          <EditableResumeName resume={resume} />
          <div className="workspace-sync-wrap mt-1">
            <SyncStatus />
          </div>
        </div>
      </div>
      <EditorNavItems mode={mode} onModeChange={onModeChange} mobilePreview={mobilePreview} onMobilePreviewChange={onMobilePreviewChange} />
      <div className="workspace-actions">
        <DownloadButton resume={resume} />
      </div>
    </header>
  )
}
