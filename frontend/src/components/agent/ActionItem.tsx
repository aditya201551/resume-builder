import { useRef, useState } from 'react'
import { Check, ChevronDown, Loader2, Lock, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import {
  canApplyProposal,
  describeProposal,
  isEditableProposalText,
  proposalCurrentValues,
  proposalDetailEntries,
  type DisplayItem,
} from '@/lib/agentProposal'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import { resolveEntryReferences } from '@/lib/resumeDraftIds'

function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return 'Not set'
  if (typeof value === 'boolean') return value ? 'Yes' : 'No'
  if (Array.isArray(value)) return value.map(formatValue).join(', ')
  if (typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value)
}

export default function ActionItem({
  item,
  onAccept,
  onReject,
  canResolve = true,
}: {
  item: Extract<DisplayItem, { kind: 'action' }>
  onAccept: (edits?: Record<string, string>) => void
  onReject: () => void
  canResolve?: boolean
}) {
  const [open, setOpen] = useState(true)
  const [editing, setEditing] = useState(false)
  const [edits, setEdits] = useState<Record<string, string>>({})
  const { event, part } = item
  const { proposal, status } = part
  const { state, resolveEntryId } = useResumeDraftContext()
  const resolved = resolveEntryReferences(proposal, resolveEntryId)
  const entries = proposalDetailEntries(resolved)
  // Retain the original values after accepting so the comparison remains meaningful.
  const original = useRef(proposalCurrentValues(resolved, state.data))
  if (status === 'pending')
    original.current = proposalCurrentValues(resolved, state.data)
  const currentValues = original.current
  const applicability = canApplyProposal(resolved, state.data)
  const running = event.status === 'running'
  const error = event.status === 'error'
  const isRemoval = proposal.type.endsWith('_delete')

  return (
    <section
      className="proposal-card animate-in fade-in slide-in-from-bottom-1 duration-150"
      aria-label={describeProposal(proposal)}
    >
      <button
        type="button"
        className="proposal-summary"
        onClick={() => setOpen(!open)}
        aria-expanded={open}
      >
        {running ? (
          <Loader2 className="size-3.5 shrink-0 animate-spin text-primary" />
        ) : error ? (
          <X className="size-3.5 shrink-0 text-destructive" />
        ) : (
          <span className="flex size-6 shrink-0 items-center justify-center rounded-lg bg-secondary text-primary">
            <Check className="size-3" />
          </span>
        )}
        <span className="min-w-0 flex-1 text-left font-medium">
          {describeProposal(proposal)}
        </span>
        {status !== 'pending' && (
          <span className="text-[10px] text-muted-foreground">
            {status === 'accepted' ? 'Accepted' : 'Rejected'}
          </span>
        )}
        <ChevronDown
          className={`size-3.5 shrink-0 text-muted-foreground transition-transform duration-150 ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>
      {open && (
        <div className="proposal-body">
          {error && (
            <p role="alert" className="mb-2 text-xs text-destructive">
              {event.error || 'This suggestion could not be completed.'}
            </p>
          )}
          <div className="proposal-comparison">
            {(isRemoval ? Object.entries(currentValues) : entries).map(
              ([key, value]) => (
                <div key={key}>
                  {Object.hasOwn(currentValues, key) &&
                    currentValues[key] !== undefined &&
                    !isRemoval && (
                      <div className="proposal-value">
                        <span className="proposal-value-label">
                          Current · {key.replace(/_/g, ' ')}
                        </span>
                        {formatValue(currentValues[key])}
                      </div>
                    )}
                  <div
                    className={`proposal-value ${isRemoval ? '' : 'is-suggestion'}`}
                  >
                    <span className="proposal-value-label">
                      {isRemoval ? 'Remove' : 'Suggested'} ·{' '}
                      {key.replace(/_/g, ' ')}
                    </span>
                    {editing &&
                    status === 'pending' &&
                    isEditableProposalText(key, value) ? (
                      <Textarea
                        aria-label={`Suggested ${key.replace(/_/g, ' ')}`}
                        rows={4}
                        value={edits[key] ?? value}
                        onChange={(event) =>
                          setEdits((values) => ({
                            ...values,
                            [key]: event.target.value,
                          }))
                        }
                      />
                    ) : (
                      formatValue(edits[key] ?? value)
                    )}
                  </div>
                </div>
              ),
            )}
          </div>
          {status === 'pending' && !running && !error && (
            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                size="sm"
                disabled={!applicability.ok || !canResolve}
                onClick={() => onAccept(edits)}
              >
                <Check className="size-3.5" />{' '}
                {isRemoval ? 'Accept removal' : 'Accept change'}
              </Button>
              {!isRemoval &&
                entries.some(([key, value]) =>
                  isEditableProposalText(key, value),
                ) && (
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    disabled={!canResolve}
                    onClick={() => setEditing(!editing)}
                  >
                    {editing ? 'Done editing' : 'Edit suggestion'}
                  </Button>
                )}
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={!canResolve}
                onClick={onReject}
              >
                <X className="size-3.5" /> Reject
              </Button>
              {!canResolve && (
                <span className="text-[10px] text-muted-foreground">
                  Finishing this response...
                </span>
              )}
            </div>
          )}
          {status === 'pending' && !applicability.ok && (
            <div className="mt-3 flex items-center gap-1.5 text-[11px] text-muted-foreground">
              <Lock className="size-3 shrink-0" />
              {applicability.reason}
            </div>
          )}
        </div>
      )}
    </section>
  )
}
