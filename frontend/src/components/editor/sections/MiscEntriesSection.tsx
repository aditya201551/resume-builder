import { useEffect, useState, type ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Button } from '@/components/ui/button'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import { useEntryPanel } from '@/hooks/useEntryPanel'
import { usePreviewOverride } from '@/hooks/usePreviewOverride'
import type { MiscEntry } from '@/types/resume'

function toDateInput(v: string | null) {
  return v ? v.slice(0, 10) : ''
}

function formFromEntry(entry: MiscEntry) {
  return {
    title: entry.title,
    issuer_or_org: entry.issuer_or_org ?? '',
    url: entry.url ?? '',
    entry_date: toDateInput(entry.entry_date),
    description: entry.description ?? '',
  }
}

function toPatch(form: ReturnType<typeof formFromEntry>) {
  return {
    title: form.title,
    issuer_or_org: form.issuer_or_org || null,
    url: form.url || null,
    entry_date: form.entry_date ? new Date(form.entry_date).toISOString() : null,
    description: form.description || null,
  }
}

function MiscEntryRow({
  entry,
  onSave,
  onDelete,
  dragHandle,
  open,
  isNew,
  onOpenChange,
}: {
  entry: MiscEntry
  onSave: (input: Partial<MiscEntry>) => Promise<unknown>
  onDelete: () => void
  dragHandle?: ReactNode
  open: boolean
  isNew: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [form, setForm] = useState(() => formFromEntry(entry))
  const isDirty = JSON.stringify(form) !== JSON.stringify(formFromEntry(entry))

  const { setOverride, clearOverride } = usePreviewOverride()
  useEffect(() => {
    if (!open) return
    return () => clearOverride(entry.id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, entry.id])
  useEffect(() => {
    if (!open) return
    setOverride(entry.id, toPatch(form))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, form, entry.id])

  function handleDone() {
    onSave({ ...toPatch(form), sort_order: entry.sort_order })
  }

  return (
    <RowCard
      onDone={handleDone}
      onDiscard={() => setForm(formFromEntry(entry))}
      isDirty={isDirty}
      onDelete={onDelete}
      title={form.title}
      subtitle={form.issuer_or_org}
      dragHandle={dragHandle}
      editorTitle="Edit entry"
      open={open}
      isNew={isNew}
      onOpenChange={onOpenChange}
    >
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label>Title</Label>
          <Input value={form.title} onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Organization</Label>
          <Input
            value={form.issuer_or_org}
            onChange={(e) => setForm((f) => ({ ...f, issuer_or_org: e.target.value }))}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Date</Label>
          <Input
            type="date"
            value={form.entry_date}
            onChange={(e) => setForm((f) => ({ ...f, entry_date: e.target.value }))}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>URL</Label>
          <Input value={form.url} onChange={(e) => setForm((f) => ({ ...f, url: e.target.value }))} />
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label>Description</Label>
        <Textarea
          rows={2}
          value={form.description}
          onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
        />
      </div>
    </RowCard>
  )
}

interface MiscEntriesSectionProps {
  resumeId: string
  kind: MiscEntry['kind']
  items: MiscEntry[]
  icon: LucideIcon
  emptyMessage: string
  addLabel: string
}

export default function MiscEntriesSection({
  resumeId,
  kind,
  items,
  icon,
  emptyMessage,
  addLabel,
}: MiscEntriesSectionProps) {
  const { create, update, remove, reorder } = useEntityMutations(resumeId, `/api/resumes/${resumeId}/misc-entries`)
  const entries = items.filter((e) => e.kind === kind)
  const { openNew, getPanelProps } = useEntryPanel(entries.map((e) => e.id))

  if (entries.length === 0) {
    return (
      <EmptyState
        icon={icon}
        message={emptyMessage}
        ctaLabel={addLabel}
        onClick={() => {
          const id = create.mutate({ kind, title: '', sort_order: entries.length })
          openNew(id)
        }}
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <SortableList
        items={entries}
        onReorder={(ids) => reorder.mutate(ids)}
        dragHandlePlacement="inline"
        renderItem={(entry, _index, dragHandle) => (
          <MiscEntryRow
            entry={entry}
            onSave={(input) => update.mutateAsync({ id: entry.id, input })}
            onDelete={() => remove.mutate(entry.id)}
            dragHandle={dragHandle}
            {...getPanelProps(entry.id)}
          />
        )}
      />
      <Button
        type="button"
        variant="secondary"
        size="sm"
        className="w-fit"
        onClick={() => {
          const id = create.mutate({ kind, title: '', sort_order: entries.length })
          openNew(id)
        }}
      >
        <Plus className="size-3.5" /> {addLabel}
      </Button>
    </div>
  )
}
