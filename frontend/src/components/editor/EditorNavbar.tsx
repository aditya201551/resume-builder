import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router'
import { ArrowLeft, Check, Download, Loader2, Pencil, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import { isDraftDirty } from '@/lib/resumeDraftSync'
import { apiDownload } from '@/lib/http'
import { cn } from '@/lib/utils'
import type { Resume } from '@/types/resume'

type EditorMode = 'content' | 'layout'

function EditableResumeName({ resume }: { resume: Resume }) {
  const { dispatch, flushNow } = useResumeDraftContext()
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
      // dispatch's state update lands on the next render, so defer the flush
      // a tick — otherwise it reads the pre-rename state.
      setTimeout(() => void flushNow(), 0)
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
  const [isDownloading, setIsDownloading] = useState(false)
  const [error, setError] = useState(false)

  async function handleDownload() {
    setIsDownloading(true)
    setError(false)
    try {
      await apiDownload(`/api/resumes/${resume.id}/export/pdf`, `${resume.label || 'resume'}.pdf`)
    } catch {
      setError(true)
    } finally {
      setIsDownloading(false)
    }
  }

  return (
    <div className="flex items-center gap-2">
      {error && <span className="text-xs text-destructive">Couldn&apos;t generate PDF</span>}
      <Button type="button" variant="secondary" size="sm" disabled={isDownloading} onClick={handleDownload}>
        {isDownloading ? <Loader2 className="size-3.5 animate-spin" /> : <Download className="size-3.5" />}
        {isDownloading ? 'Generating…' : 'Download'}
      </Button>
    </div>
  )
}

const NAV_ITEMS: { key: EditorMode; label: string }[] = [
  { key: 'content', label: 'Content' },
  { key: 'layout', label: 'Layout' },
]

function EditorNavItems({ mode, onModeChange }: { mode: EditorMode; onModeChange: (mode: EditorMode) => void }) {
  return (
    <div className="flex items-center gap-1">
      {NAV_ITEMS.map((item) => (
        <button
          key={item.key}
          type="button"
          onClick={() => onModeChange(item.key)}
          className={cn(
            'rounded-md px-2.5 py-1.5 text-sm font-medium text-muted-foreground hover:text-foreground',
            mode === item.key && 'bg-secondary text-foreground',
          )}
        >
          {item.label}
        </button>
      ))}
    </div>
  )
}

function SyncStatus() {
  const { state, syncStatus } = useResumeDraftContext()
  const dirty = isDraftDirty(state.data, state.lastSynced)
  const label =
    syncStatus === 'syncing'
      ? 'Saving…'
      : syncStatus === 'error'
        ? "Couldn't save — retrying"
        : dirty
          ? 'Unsaved changes'
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
      <Button variant="ghost" size="icon" asChild>
        <Link to="/resumes">
          <ArrowLeft className="size-4" />
        </Link>
      </Button>

      <EditorNavItems mode={mode} onModeChange={onModeChange} />

      <div className="flex-1" />

      <EditableResumeName resume={resume} />

      <DownloadButton resume={resume} />

      <SyncStatus />
    </div>
  )
}
