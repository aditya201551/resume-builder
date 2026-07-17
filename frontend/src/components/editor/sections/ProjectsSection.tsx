import { useState } from 'react'
import { FolderGit2, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import RichTextEditor from '@/components/editor/RichTextEditor'
import { useAutosave } from '@/hooks/useAutosave'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import type { Project } from '@/types/resume'

function toDateInput(v: string | null) {
  return v ? v.slice(0, 10) : ''
}

function ProjectRow({
  entry,
  onSave,
  onDelete,
}: {
  entry: Project
  onSave: (input: Partial<Project>) => Promise<unknown>
  onDelete: () => void
}) {
  const [form, setForm] = useState({
    name: entry.name,
    role: entry.role ?? '',
    url: entry.url ?? '',
    start_date: toDateInput(entry.start_date),
    end_date: toDateInput(entry.end_date),
    content: entry.content,
    technologies: entry.technologies.join(', '),
  })

  const status = useAutosave(form, (value) =>
    onSave({
      name: value.name,
      role: value.role || null,
      url: value.url || null,
      start_date: value.start_date ? new Date(value.start_date).toISOString() : null,
      end_date: value.end_date ? new Date(value.end_date).toISOString() : null,
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
          <Label>Name</Label>
          <Input value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Role</Label>
          <Input value={form.role} onChange={(e) => setForm((f) => ({ ...f, role: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>URL</Label>
          <Input value={form.url} onChange={(e) => setForm((f) => ({ ...f, url: e.target.value }))} />
        </div>
        <div className="grid grid-cols-2 gap-3">
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
        </div>
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

export default function ProjectsSection({ resumeId, items }: { resumeId: string; items: Project[] }) {
  const { create, update, remove, reorder } = useEntityMutations(resumeId, `/api/resumes/${resumeId}/projects`)

  if (items.length === 0) {
    return (
      <EmptyState
        icon={FolderGit2}
        message="No projects added yet."
        ctaLabel="Add project"
        onClick={() => create.mutate({ name: '', content: '', technologies: [], sort_order: items.length })}
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <SortableList
        items={items}
        onReorder={(ids) => reorder.mutate(ids)}
        renderItem={(entry) => (
          <ProjectRow
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
        onClick={() => create.mutate({ name: '', content: '', technologies: [], sort_order: items.length })}
      >
        <Plus className="size-3.5" /> Add project
      </Button>
    </div>
  )
}
