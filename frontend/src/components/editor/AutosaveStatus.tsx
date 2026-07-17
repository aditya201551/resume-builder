import type { SaveStatus } from '@/hooks/useAutosave'
import { cn } from '@/lib/utils'

const COPY: Record<SaveStatus, string> = {
  idle: 'All changes saved',
  saving: 'Saving…',
  saved: 'Saved just now',
  error: "Couldn't save — retrying",
}

export default function AutosaveStatus({
  status,
  className,
  compact = false,
}: {
  status: SaveStatus
  className?: string
  compact?: boolean
}) {
  return (
    <span
      title={COPY[status]}
      className={cn(
        'inline-flex items-center gap-1.5 text-xs',
        status === 'error' ? 'text-destructive' : 'text-muted-foreground',
        className,
      )}
    >
      <span
        className={cn(
          'size-1.5 shrink-0 rounded-full',
          status === 'saving' && 'bg-accent animate-pulse',
          status === 'saved' && 'bg-accent',
          status === 'error' && 'bg-destructive',
          status === 'idle' && 'bg-muted-foreground/40',
        )}
      />
      {!compact && COPY[status]}
    </span>
  )
}
