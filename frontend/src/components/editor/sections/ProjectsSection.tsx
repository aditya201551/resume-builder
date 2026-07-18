import { useEffect, useState, type ReactNode } from 'react'
import { FolderGit2, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import RichTextEditor from '@/components/editor/RichTextEditor'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import { useEntryPanel } from '@/hooks/useEntryPanel'
import { usePreviewOverride } from '@/hooks/usePreviewOverride'
import type { Project } from '@/types/resume'

function toDateInput(v: string | null) {
  return v ? v.slice(0, 10) : ''
}

function formFromEntry(entry: Project) {
  return {
    name: entry.name,
    role: entry.role ?? '',
    url: entry.url ?? '',
    start_date: toDateInput(entry.start_date),
    end_date: toDateInput(entry.end_date),
    content: entry.content,
    technologies: entry.technologies.join(', '),
  }
}

function toPatch(form: ReturnType<typeof formFromEntry>) {
  return {
    name: form.name,
    role: form.role || null,
    url: form.url || null,
    start_date: form.start_date ? new Date(form.start_date).toISOString() : null,
    end_date: form.end_date ? new Date(form.end_date).toISOString() : null,
    content: form.content,
    technologies: form.technologies
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean),
  }
}

function ProjectRow({
  entry,
  onSave,
  onDelete,
  dragHandle,
  open,
  isNew,
  onOpenChange,
}: {
  entry: Project
  onSave: (input: Partial<Project>) => Promise<unknown>
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
      subtitle={form.role}
      dragHandle={dragHandle}
      editorTitle="Edit project"
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
  const { openNew, getPanelProps } = useEntryPanel(items.map((i) => i.id))

  if (items.length === 0) {
    return (
      <EmptyState
        icon={FolderGit2}
        message="No projects added yet."
        ctaLabel="Add project"
        onClick={() => {
          const id = create.mutate({ name: '', content: '', technologies: [], sort_order: items.length })
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
          <ProjectRow
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
          const id = create.mutate({ name: '', content: '', technologies: [], sort_order: items.length })
          openNew(id)
        }}
      >
        <Plus className="size-3.5" /> Add project
      </Button>
    </div>
  )
}
