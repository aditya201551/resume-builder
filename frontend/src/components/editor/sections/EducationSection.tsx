import { useState } from 'react'
import { GraduationCap, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Button } from '@/components/ui/button'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import { useAutosave } from '@/hooks/useAutosave'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import type { Education } from '@/types/resume'

function toDateInput(v: string | null) {
  return v ? v.slice(0, 10) : ''
}

function EducationRow({
  entry,
  onSave,
  onDelete,
}: {
  entry: Education
  onSave: (input: Partial<Education>) => Promise<unknown>
  onDelete: () => void
}) {
  const [form, setForm] = useState({
    institution: entry.institution,
    degree: entry.degree ?? '',
    field_of_study: entry.field_of_study ?? '',
    location: entry.location ?? '',
    start_date: toDateInput(entry.start_date),
    end_date: toDateInput(entry.end_date),
    gpa: entry.gpa ?? '',
    honors_description: entry.honors_description ?? '',
  })

  const status = useAutosave(form, (value) =>
    onSave({
      institution: value.institution,
      degree: value.degree || null,
      field_of_study: value.field_of_study || null,
      location: value.location || null,
      start_date: value.start_date ? new Date(value.start_date).toISOString() : null,
      end_date: value.end_date ? new Date(value.end_date).toISOString() : null,
      gpa: value.gpa || null,
      honors_description: value.honors_description || null,
      sort_order: entry.sort_order,
    }),
  )

  return (
    <RowCard status={status} onDelete={onDelete}>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label>Institution</Label>
          <Input value={form.institution} onChange={(e) => setForm((f) => ({ ...f, institution: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Degree</Label>
          <Input value={form.degree} onChange={(e) => setForm((f) => ({ ...f, degree: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Field of study</Label>
          <Input
            value={form.field_of_study}
            onChange={(e) => setForm((f) => ({ ...f, field_of_study: e.target.value }))}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Location</Label>
          <Input value={form.location} onChange={(e) => setForm((f) => ({ ...f, location: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Start date</Label>
          <Input
            type="date"
            value={form.start_date}
            onChange={(e) => setForm((f) => ({ ...f, start_date: e.target.value }))}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>End date</Label>
          <Input
            type="date"
            value={form.end_date}
            onChange={(e) => setForm((f) => ({ ...f, end_date: e.target.value }))}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>GPA</Label>
          <Input value={form.gpa} onChange={(e) => setForm((f) => ({ ...f, gpa: e.target.value }))} />
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label>Honors / description</Label>
        <Textarea
          rows={2}
          value={form.honors_description}
          onChange={(e) => setForm((f) => ({ ...f, honors_description: e.target.value }))}
        />
      </div>
    </RowCard>
  )
}

export default function EducationSection({ resumeId, items }: { resumeId: string; items: Education[] }) {
  const { create, update, remove, reorder } = useEntityMutations(resumeId, `/api/resumes/${resumeId}/educations`)

  if (items.length === 0) {
    return (
      <EmptyState
        icon={GraduationCap}
        message="No education added yet."
        ctaLabel="Add education"
        onClick={() => create.mutate({ institution: '', sort_order: items.length })}
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <SortableList
        items={items}
        onReorder={(ids) => reorder.mutate(ids)}
        renderItem={(entry) => (
          <EducationRow
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
        onClick={() => create.mutate({ institution: '', sort_order: items.length })}
      >
        <Plus className="size-3.5" /> Add education
      </Button>
    </div>
  )
}
