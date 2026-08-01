import { useState } from 'react'
import { Check, ChevronDown, Loader2, Lock, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { canApplyProposal, describeProposal, proposalDetailEntries, type DisplayItem } from '@/lib/agentProposal'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'

function formatValue(v: unknown): string {
  if (v === null || v === undefined) return '—'
  if (typeof v === 'boolean') return v ? 'yes' : 'no'
  return String(v)
}

/**
 * One propose_* tool call, merged with the proposal it produced: a single
 * collapsible row instead of a separate activity pill plus detail card.
 * Collapsed by default — the header alone (status + description + outcome)
 * is enough to follow along with what the assistant did; the field-level
 * detail of what was actually sent is a click away rather than always
 * taking up space.
 */
export default function ActionItem({
  item,
  onAccept,
  onReject,
}: {
  item: Extract<DisplayItem, { kind: 'action' }>
  onAccept: () => void
  onReject: () => void
}) {
  const [open, setOpen] = useState(false)
  const { event, part } = item
  const { proposal, status } = part
  const entries = proposalDetailEntries(proposal)

  const isRunning = event.status === 'running'
  const isToolError = event.status === 'error'

  const { state } = useResumeDraftContext()
  const applicability = canApplyProposal(proposal, state.data)

  return (
    <div className="animate-in fade-in slide-in-from-bottom-1 overflow-hidden rounded-lg border border-border bg-card text-sm duration-200">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left"
        aria-expanded={open}
      >
        {isRunning ? (
          <Loader2 className="size-3.5 shrink-0 animate-spin text-accent" />
        ) : isToolError ? (
          <X className="size-3.5 shrink-0 text-destructive" />
        ) : (
          <Check className="size-3.5 shrink-0 text-emerald-600 dark:text-emerald-400" />
        )}

        <span className="min-w-0 flex-1 truncate font-medium text-foreground">{describeProposal(proposal)}</span>

        {isToolError && <span className="shrink-0 text-xs text-destructive">{event.error || 'failed'}</span>}
        {status === 'accepted' && <span className="shrink-0 text-xs font-medium text-emerald-600 dark:text-emerald-400">Accepted</span>}
        {status === 'rejected' && <span className="shrink-0 text-xs font-medium text-muted-foreground">Rejected</span>}
        {status === 'pending' && !isRunning && !isToolError && (
          <div className="flex shrink-0 items-center gap-1.5" onClick={(e) => e.stopPropagation()}>
            <Button type="button" size="sm" onClick={onAccept} disabled={!applicability.ok}>
              <Check className="size-3.5" /> Accept
            </Button>
            <Button type="button" size="sm" variant="secondary" onClick={onReject}>
              <X className="size-3.5" /> Reject
            </Button>
          </div>
        )}

        <ChevronDown className={`size-3.5 shrink-0 text-muted-foreground transition-transform ${open ? 'rotate-180' : ''}`} aria-hidden />
      </button>

      {open && (
        <div className="flex flex-col gap-2 border-t border-border px-3 py-2">
          {entries.length > 0 && (
            <dl className="flex flex-col gap-0.5 text-xs text-muted-foreground">
              {entries.map(([key, value]) => (
                <div key={key} className="flex gap-1.5">
                  <dt className="shrink-0 font-medium">{key.replace(/_/g, ' ')}:</dt>
                  <dd className="min-w-0">{formatValue(value)}</dd>
                </div>
              ))}
            </dl>
          )}

          {status === 'pending' && !applicability.ok && (
            <div className="flex items-center gap-1.5 rounded-md bg-muted px-2 py-1.5 text-xs text-muted-foreground">
              <Lock className="size-3 shrink-0" />
              <span>{applicability.reason}</span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
