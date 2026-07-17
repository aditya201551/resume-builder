import { useState } from 'react'
import { Briefcase, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import RichTextEditor from '@/components/editor/RichTextEditor'
import { useAutosave } from '@/hooks/useAutosave'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import type { WorkExperience } from '@/types/resume'

function toDateInput(v: string | null) {
  return v ? v.slice(0, 10) : ''
}

function WorkExperienceRow({
  entry,
  onSave,
  onDelete,
}: {
  entry: WorkExperience
  onSave: (input: Partial<WorkExperience>) => Promise<unknown>
  onDelete: () => void
}) {
  const [form, setForm] = useState({
    company: entry.company,
    title: entry.title,
    location: entry.location ?? '',
    employment_type: entry.employment_type ?? '',
    start_date: toDateInput(entry.start_date),
    end_date: toDateInput(entry.end_date),
    is_current: entry.is_current,
    content: entry.content,
    technologies: entry.technologies.join(', '),
  })

  const status = useAutosave(form, (value) =>
    onSave({
      company: value.company,
      title: value.title,
      location: value.location || null,
      employment_type: value.employment_type || null,
      start_date: value.start_date ? new Date(value.start_date).toISOString() : null,
      end_date: value.is_current || !value.end_date ? null : new Date(value.end_date).toISOString(),
      is_current: value.is_current,
      content: value.content,
      technologies: value.technologies
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean),
      sort_order: entry.sort_order,
    }),
  )

  return (
    <RowCard status={status} onDelete={onDelete}>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label>Company</Label>
          <Input value={form.company} onChange={(e) => setForm((f) => ({ ...f, company: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Title</Label>
          <Input value={form.title} onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Location</Label>
          <Input value={form.location} onChange={(e) => setForm((f) => ({ ...f, location: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Employment type</Label>
          <Input
            value={form.employment_type}
            onChange={(e) => setForm((f) => ({ ...f, employment_type: e.target.value }))}
            placeholder="Full-time"
          />
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
            disabled={form.is_current}
            value={form.end_date}
            onChange={(e) => setForm((f) => ({ ...f, end_date: e.target.value }))}
          />
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Switch
          checked={form.is_current}
          onCheckedChange={(checked) => setForm((f) => ({ ...f, is_current: checked }))}
        />
        <Label className="font-normal">I currently work here</Label>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Description</Label>
        <RichTextEditor value={form.content} onChange={(md) => setForm((f) => ({ ...f, content: md }))} />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Technologies</Label>
        <Input
          value={form.technologies}
          onChange={(e) => setForm((f) => ({ ...f, technologies: e.target.value }))}
          placeholder="Go, PostgreSQL, React (comma-separated)"
        />
      </div>
    </RowCard>
  )
}

export default function WorkExperienceSection({ resumeId, items }: { resumeId: string; items: WorkExperience[] }) {
  const { create, update, remove, reorder } = useEntityMutations(resumeId, `/api/resumes/${resumeId}/work-experiences`)

  if (items.length === 0) {
    return (
      <EmptyState
        icon={Briefcase}
        message="No work experience added yet."
        ctaLabel="Add work experience"
        onClick={() => create.mutate({ company: '', title: '', content: '', technologies: [], sort_order: items.length })}
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <SortableList
        items={items}
        onReorder={(ids) => reorder.mutate(ids)}
        renderItem={(entry) => (
          <WorkExperienceRow
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
        onClick={() => create.mutate({ company: '', title: '', content: '', technologies: [], sort_order: items.length })}
      >
        <Plus className="size-3.5" /> Add work experience
      </Button>
    </div>
  )
}
