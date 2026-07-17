import { useState } from 'react'
import { Languages as LanguagesIcon, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import { useAutosave } from '@/hooks/useAutosave'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import type { Language } from '@/types/resume'

function LanguageRow({
  entry,
  onSave,
  onDelete,
}: {
  entry: Language
  onSave: (input: Partial<Language>) => Promise<unknown>
  onDelete: () => void
}) {
  const [form, setForm] = useState({ name: entry.name, proficiency: entry.proficiency ?? '' })

  const status = useAutosave(form, (value) =>
    onSave({ name: value.name, proficiency: value.proficiency || null, sort_order: entry.sort_order }),
  )

  return (
    <RowCard status={status} onDelete={onDelete}>
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

  if (items.length === 0) {
    return (
      <EmptyState
        icon={LanguagesIcon}
        message="No languages added yet."
        ctaLabel="Add language"
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
          <LanguageRow
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
        <Plus className="size-3.5" /> Add language
      </Button>
    </div>
  )
}
