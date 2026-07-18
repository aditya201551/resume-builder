import { useEffect, useState, type ReactNode } from 'react'
import { LayoutList, Plus } from 'lucide-react'
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
import type { CustomSection, CustomSectionEntry } from '@/types/resume'

function toDateInput(v: string | null) {
  return v ? v.slice(0, 10) : ''
}

function formFromEntry(entry: CustomSectionEntry) {
  return {
    title: entry.title ?? '',
    description: entry.description ?? '',
    entry_date: toDateInput(entry.entry_date),
  }
}

function toPatch(form: ReturnType<typeof formFromEntry>) {
  return {
    title: form.title || null,
    description: form.description || null,
    entry_date: form.entry_date ? new Date(form.entry_date).toISOString() : null,
  }
}

function CustomEntryRow({
  entry,
  onSave,
  onDelete,
  dragHandle,
  open,
  isNew,
  onOpenChange,
}: {
  entry: CustomSectionEntry
  onSave: (input: Partial<CustomSectionEntry>) => Promise<unknown>
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
      subtitle={form.entry_date}
      dragHandle={dragHandle}
      editorTitle="Edit entry"
      open={open}
      isNew={isNew}
      onOpenChange={onOpenChange}
    >
      <div className="flex gap-2">
        <div className="flex flex-1 flex-col gap-1.5">
          <Label>Title</Label>
          <Input value={form.title} onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Date</Label>
          <Input
            className="w-40"
            type="date"
            value={form.entry_date}
            onChange={(e) => setForm((f) => ({ ...f, entry_date: e.target.value }))}
          />
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

function CustomSectionCard({
  resumeId,
  section,
  onSaveSection,
  onDeleteSection,
  dragHandle,
  open,
  isNew,
  onOpenChange,
}: {
  resumeId: string
  section: CustomSection
  onSaveSection: (input: Partial<CustomSection>) => Promise<unknown>
  onDeleteSection: () => void
  dragHandle?: ReactNode
  open: boolean
  isNew: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [title, setTitle] = useState(section.title)
  const isDirty = title !== section.title
  const entries = useEntityMutations(
    resumeId,
    `/api/resumes/${resumeId}/custom-sections/${section.id}/entries`,
  )
  const { openNew: openNewEntry, getPanelProps: getEntryPanelProps } = useEntryPanel(section.entries.map((e) => e.id))

  const { setOverride, clearOverride } = usePreviewOverride()
  useEffect(() => {
    if (!open) return
    return () => clearOverride(section.id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, section.id])
  useEffect(() => {
    if (!open) return
    setOverride(section.id, { title })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, title, section.id])

  return (
    <RowCard
      onDone={() => onSaveSection({ title, sort_order: section.sort_order })}
      onDiscard={() => setTitle(section.title)}
      isDirty={isDirty}
      onDelete={onDeleteSection}
      title={title}
      subtitle={section.entries.length > 0 ? `${section.entries.length} entr${section.entries.length === 1 ? 'y' : 'ies'}` : undefined}
      dragHandle={dragHandle}
      editorTitle="Edit custom section"
      open={open}
      isNew={isNew}
      onOpenChange={onOpenChange}
    >
      <div className="flex flex-col gap-1.5">
        <Label>Section title</Label>
        <Input
          className="font-medium"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Section title"
        />
      </div>

      <div className="flex flex-col gap-2">
        <SortableList
          items={section.entries}
          onReorder={(ids) => entries.reorder.mutate(ids)}
          dragHandlePlacement="inline"
          renderItem={(entry, _index, entryDragHandle) => (
            <CustomEntryRow
              entry={entry}
              onSave={(input) => entries.update.mutateAsync({ id: entry.id, input })}
              onDelete={() => entries.remove.mutate(entry.id)}
              dragHandle={entryDragHandle}
              {...getEntryPanelProps(entry.id)}
            />
          )}
        />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="w-fit text-muted-foreground"
          onClick={() => {
            const id = entries.create.mutate({ sort_order: section.entries.length })
            openNewEntry(id)
          }}
        >
          <Plus className="size-3.5" /> Add entry
        </Button>
      </div>
    </RowCard>
  )
}

export default function CustomSectionsSection({ resumeId, items }: { resumeId: string; items: CustomSection[] }) {
  const { create, update, remove, reorder } = useEntityMutations(resumeId, `/api/resumes/${resumeId}/custom-sections`)
  const { openNew, getPanelProps } = useEntryPanel(items.map((i) => i.id))

  if (items.length === 0) {
    return (
      <EmptyState
        icon={LayoutList}
        message="No custom sections added yet."
        ctaLabel="Add custom section"
        onClick={() => {
          const id = create.mutate({ title: '', sort_order: items.length })
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
        renderItem={(section, _index, dragHandle) => (
          <CustomSectionCard
            resumeId={resumeId}
            section={section}
            onSaveSection={(input) => update.mutateAsync({ id: section.id, input })}
            onDeleteSection={() => remove.mutate(section.id)}
            dragHandle={dragHandle}
            {...getPanelProps(section.id)}
          />
        )}
      />
      <Button
        type="button"
        variant="secondary"
        size="sm"
        className="w-fit"
        onClick={() => {
          const id = create.mutate({ title: '', sort_order: items.length })
          openNew(id)
        }}
      >
        <Plus className="size-3.5" /> Add custom section
      </Button>
    </div>
  )
}
