import { useEffect, useState, type ReactNode } from 'react'
import { Award, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import { useEntryPanel } from '@/hooks/useEntryPanel'
import { usePreviewOverride } from '@/hooks/usePreviewOverride'
import type { Certification } from '@/types/resume'

function toDateInput(v: string | null) {
  return v ? v.slice(0, 10) : ''
}

function formFromEntry(entry: Certification) {
  return {
    name: entry.name,
    issuer: entry.issuer ?? '',
    issue_date: toDateInput(entry.issue_date),
    expiry_date: toDateInput(entry.expiry_date),
    credential_url: entry.credential_url ?? '',
  }
}

function toPatch(form: ReturnType<typeof formFromEntry>) {
  return {
    name: form.name,
    issuer: form.issuer || null,
    issue_date: form.issue_date ? new Date(form.issue_date).toISOString() : null,
    expiry_date: form.expiry_date ? new Date(form.expiry_date).toISOString() : null,
    credential_url: form.credential_url || null,
  }
}

function CertificationRow({
  entry,
  onSave,
  onDelete,
  dragHandle,
  open,
  isNew,
  onOpenChange,
}: {
  entry: Certification
  onSave: (input: Partial<Certification>) => Promise<unknown>
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
      title={form.name}
      subtitle={form.issuer}
      dragHandle={dragHandle}
      editorTitle="Edit certification"
      open={open}
      isNew={isNew}
      onOpenChange={onOpenChange}
    >
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label>Name</Label>
          <Input value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Issuer</Label>
          <Input value={form.issuer} onChange={(e) => setForm((f) => ({ ...f, issuer: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Issue date</Label>
          <Input
            type="date"
            value={form.issue_date}
            onChange={(e) => setForm((f) => ({ ...f, issue_date: e.target.value }))}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Expiry date</Label>
          <Input
            type="date"
            value={form.expiry_date}
            onChange={(e) => setForm((f) => ({ ...f, expiry_date: e.target.value }))}
          />
        </div>
        <div className="flex flex-col gap-1.5 sm:col-span-2">
          <Label>Credential URL</Label>
          <Input
            value={form.credential_url}
            onChange={(e) => setForm((f) => ({ ...f, credential_url: e.target.value }))}
          />
        </div>
      </div>
    </RowCard>
  )
}

export default function CertificationsSection({ resumeId, items }: { resumeId: string; items: Certification[] }) {
  const { create, update, remove, reorder } = useEntityMutations(resumeId, `/api/resumes/${resumeId}/certifications`)
  const { openNew, getPanelProps } = useEntryPanel(items.map((i) => i.id))

  if (items.length === 0) {
    return (
      <EmptyState
        icon={Award}
        message="No certifications added yet."
        ctaLabel="Add certification"
        onClick={() => {
          const id = create.mutate({ name: '', sort_order: items.length })
          openNew(id)
        }}
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <SortableList
        items={items}
        onReorder={(ids) => reorder.mutate(ids)}
        dragHandlePlacement="inline"
        renderItem={(entry, _index, dragHandle) => (
          <CertificationRow
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
          const id = create.mutate({ name: '', sort_order: items.length })
          openNew(id)
        }}
      >
        <Plus className="size-3.5" /> Add certification
      </Button>
    </div>
  )
}
