import { useState } from 'react'
import { LayoutList, Plus, Trash2 } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Button } from '@/components/ui/button'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import AutosaveStatus from '@/components/editor/AutosaveStatus'
import { useAutosave } from '@/hooks/useAutosave'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import type { CustomSection, CustomSectionEntry } from '@/types/resume'

function toDateInput(v: string | null) {
  return v ? v.slice(0, 10) : ''
}

function CustomEntryRow({
  entry,
  onSave,
  onDelete,
}: {
  entry: CustomSectionEntry
  onSave: (input: Partial<CustomSectionEntry>) => Promise<unknown>
  onDelete: () => void
}) {
  const [form, setForm] = useState({
    title: entry.title ?? '',
    description: entry.description ?? '',
    entry_date: toDateInput(entry.entry_date),
  })
  const status = useAutosave(form, (value) =>
    onSave({
      title: value.title || null,
      description: value.description || null,
      entry_date: value.entry_date ? new Date(value.entry_date).toISOString() : null,
      sort_order: entry.sort_order,
    }),
  )

  return (
    <div className="rounded-md border border-border bg-card p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <AutosaveStatus status={status} compact />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-6 text-muted-foreground hover:text-destructive"
          onClick={onDelete}
        >
          <Trash2 className="size-3.5" />
        </Button>
      </div>
      <div className="flex flex-col gap-2">
        <div className="flex gap-2">
          <Input
            className="h-8"
            value={form.title}
            onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
            placeholder="Title"
          />
          <Input
            className="h-8 w-40"
            type="date"
            value={form.entry_date}
            onChange={(e) => setForm((f) => ({ ...f, entry_date: e.target.value }))}
          />
        </div>
        <Textarea
          rows={2}
          value={form.description}
          onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
          placeholder="Description"
        />
      </div>
    </div>
  )
}

function CustomSectionCard({
  resumeId,
  section,
  onSaveSection,
  onDeleteSection,
}: {
  resumeId: string
  section: CustomSection
  onSaveSection: (input: Partial<CustomSection>) => Promise<unknown>
  onDeleteSection: () => void
}) {
  const [title, setTitle] = useState(section.title)
  const status = useAutosave(title, (value) => onSaveSection({ title: value, sort_order: section.sort_order }))
  const entries = useEntityMutations(
    resumeId,
    `/api/resumes/${resumeId}/custom-sections/${section.id}/entries`,
  )

  return (
    <div className="rounded-md border border-border bg-card p-4">
      <div className="mb-3 flex items-center gap-2">
        <Input
          className="font-medium"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Section title"
        />
        <AutosaveStatus status={status} className="shrink-0" />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-8 shrink-0 text-muted-foreground hover:text-destructive"
          onClick={onDeleteSection}
        >
          <Trash2 className="size-3.5" />
        </Button>
      </div>

      <div className="flex flex-col gap-2">
        <SortableList
          items={section.entries}
          onReorder={(ids) => entries.reorder.mutate(ids)}
          renderItem={(entry) => (
            <CustomEntryRow
              entry={entry}
              onSave={(input) => entries.update.mutateAsync({ id: entry.id, input })}
              onDelete={() => entries.remove.mutate(entry.id)}
            />
          )}
        />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="w-fit text-muted-foreground"
          onClick={() => entries.create.mutate({ sort_order: section.entries.length })}
        >
          <Plus className="size-3.5" /> Add entry
        </Button>
      </div>
    </div>
  )
}

export default function CustomSectionsSection({ resumeId, items }: { resumeId: string; items: CustomSection[] }) {
  const { create, update, remove, reorder } = useEntityMutations(resumeId, `/api/resumes/${resumeId}/custom-sections`)

  if (items.length === 0) {
    return (
      <EmptyState
        icon={LayoutList}
        message="No custom sections added yet."
        ctaLabel="Add custom section"
        onClick={() => create.mutate({ title: '', sort_order: items.length })}
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <SortableList
        items={items}
        onReorder={(ids) => reorder.mutate(ids)}
        renderItem={(section) => (
          <CustomSectionCard
            resumeId={resumeId}
            section={section}
            onSaveSection={(input) => update.mutateAsync({ id: section.id, input })}
            onDeleteSection={() => remove.mutate(section.id)}
          />
        )}
      />
      <Button
        type="button"
        variant="secondary"
        size="sm"
        className="w-fit"
        onClick={() => create.mutate({ title: '', sort_order: items.length })}
      >
        <Plus className="size-3.5" /> Add custom section
      </Button>
    </div>
  )
}
