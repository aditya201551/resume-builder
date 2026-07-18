import { useEffect, useState, type ReactNode } from 'react'
import { Briefcase, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import RichTextEditor from '@/components/editor/RichTextEditor'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import { useEntryPanel } from '@/hooks/useEntryPanel'
import { usePreviewOverride } from '@/hooks/usePreviewOverride'
import type { WorkExperience } from '@/types/resume'

function toDateInput(v: string | null) {
  return v ? v.slice(0, 10) : ''
}

function formFromEntry(entry: WorkExperience) {
  return {
    company: entry.company,
    company_url: entry.company_url ?? '',
    title: entry.title,
    location: entry.location ?? '',
    employment_type: entry.employment_type ?? '',
    start_date: toDateInput(entry.start_date),
    end_date: toDateInput(entry.end_date),
    is_current: entry.is_current,
    content: entry.content,
    technologies: entry.technologies.join(', '),
  }
}

function toPatch(form: ReturnType<typeof formFromEntry>) {
  return {
    company: form.company,
    company_url: form.company_url || null,
    title: form.title,
    location: form.location || null,
    employment_type: form.employment_type || null,
    start_date: form.start_date ? new Date(form.start_date).toISOString() : null,
    end_date: form.is_current || !form.end_date ? null : new Date(form.end_date).toISOString(),
    is_current: form.is_current,
    content: form.content,
    technologies: form.technologies
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean),
  }
}

function WorkExperienceRow({
  entry,
  onSave,
  onDelete,
  dragHandle,
  open,
  isNew,
  onOpenChange,
}: {
  entry: WorkExperience
  onSave: (input: Partial<WorkExperience>) => Promise<unknown>
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
      subtitle={form.company}
      dragHandle={dragHandle}
      editorTitle="Edit work experience"
      open={open}
      isNew={isNew}
      onOpenChange={onOpenChange}
    >
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label>Company</Label>
          <Input value={form.company} onChange={(e) => setForm((f) => ({ ...f, company: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Title</Label>
          <Input value={form.title} onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5 sm:col-span-2">
          <Label>Company link</Label>
          <Input
            type="url"
            value={form.company_url}
            onChange={(e) => setForm((f) => ({ ...f, company_url: e.target.value }))}
            placeholder="https://company.com"
          />
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
  const { openNew, getPanelProps } = useEntryPanel(items.map((i) => i.id))

  if (items.length === 0) {
    return (
      <EmptyState
        icon={Briefcase}
        message="No work experience added yet."
        ctaLabel="Add work experience"
        onClick={() => {
          const id = create.mutate({ company: '', title: '', content: '', technologies: [], sort_order: items.length })
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
          <WorkExperienceRow
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
          const id = create.mutate({ company: '', title: '', content: '', technologies: [], sort_order: items.length })
          openNew(id)
        }}
      >
        <Plus className="size-3.5" /> Add work experience
      </Button>
    </div>
  )
}
