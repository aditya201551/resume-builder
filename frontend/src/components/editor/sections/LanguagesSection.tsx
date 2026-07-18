import { useEffect, useState, type ReactNode } from 'react'
import { Languages as LanguagesIcon, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import { useEntryPanel } from '@/hooks/useEntryPanel'
import { usePreviewOverride } from '@/hooks/usePreviewOverride'
import type { Language } from '@/types/resume'

function formFromEntry(entry: Language) {
  return { name: entry.name, proficiency: entry.proficiency ?? '' }
}

function toPatch(form: ReturnType<typeof formFromEntry>) {
  return { name: form.name, proficiency: form.proficiency || null }
}

function LanguageRow({
  entry,
  onSave,
  onDelete,
  dragHandle,
  open,
  isNew,
  onOpenChange,
}: {
  entry: Language
  onSave: (input: Partial<Language>) => Promise<unknown>
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
      subtitle={form.proficiency}
      dragHandle={dragHandle}
      editorTitle="Edit language"
      open={open}
      isNew={isNew}
      onOpenChange={onOpenChange}
    >
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label>Language</Label>
          <Input value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Proficiency</Label>
          <Input
            value={form.proficiency}
            onChange={(e) => setForm((f) => ({ ...f, proficiency: e.target.value }))}
            placeholder="Native, Fluent, Conversational…"
          />
        </div>
      </div>
    </RowCard>
  )
}

export default function LanguagesSection({ resumeId, items }: { resumeId: string; items: Language[] }) {
  const { create, update, remove, reorder } = useEntityMutations(resumeId, `/api/resumes/${resumeId}/languages`)
  const { openNew, getPanelProps } = useEntryPanel(items.map((i) => i.id))

  if (items.length === 0) {
    return (
      <EmptyState
        icon={LanguagesIcon}
        message="No languages added yet."
        ctaLabel="Add language"
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
          <LanguageRow
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
        <Plus className="size-3.5" /> Add language
      </Button>
    </div>
  )
}
