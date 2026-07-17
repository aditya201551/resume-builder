import { useState } from 'react'
import { Award, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import { useAutosave } from '@/hooks/useAutosave'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import type { Certification } from '@/types/resume'

function toDateInput(v: string | null) {
  return v ? v.slice(0, 10) : ''
}

function CertificationRow({
  entry,
  onSave,
  onDelete,
}: {
  entry: Certification
  onSave: (input: Partial<Certification>) => Promise<unknown>
  onDelete: () => void
}) {
  const [form, setForm] = useState({
    name: entry.name,
    issuer: entry.issuer ?? '',
    issue_date: toDateInput(entry.issue_date),
    expiry_date: toDateInput(entry.expiry_date),
    credential_url: entry.credential_url ?? '',
  })

  const status = useAutosave(form, (value) =>
    onSave({
      name: value.name,
      issuer: value.issuer || null,
      issue_date: value.issue_date ? new Date(value.issue_date).toISOString() : null,
      expiry_date: value.expiry_date ? new Date(value.expiry_date).toISOString() : null,
      credential_url: value.credential_url || null,
      sort_order: entry.sort_order,
    }),
  )

  return (
    <RowCard status={status} onDelete={onDelete}>
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

  if (items.length === 0) {
    return (
      <EmptyState
        icon={Award}
        message="No certifications added yet."
        ctaLabel="Add certification"
        onClick={() => create.mutate({ name: '', sort_order: items.length })}
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <SortableList
        items={items}
        onReorder={(ids) => reorder.mutate(ids)}
        renderItem={(entry) => (
          <CertificationRow
            entry={entry}
            onSave={(input) => update.mutateAsync({ id: entry.id, input })}
            onDelete={() => remove.mutate(entry.id)}
          />
        )}
      />
      <Button
        type="button"
        variant="secondary"
        size="sm"
        className="w-fit"
        onClick={() => create.mutate({ name: '', sort_order: items.length })}
      >
        <Plus className="size-3.5" /> Add certification
      </Button>
    </div>
  )
}
