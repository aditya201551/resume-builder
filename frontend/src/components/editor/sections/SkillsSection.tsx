import { useState } from 'react'
import { Layers, Plus, Trash2 } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import AutosaveStatus from '@/components/editor/AutosaveStatus'
import { useAutosave } from '@/hooks/useAutosave'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import type { SkillGroup, SkillItem } from '@/types/resume'

function SkillItemRow({
  item,
  onSave,
  onDelete,
}: {
  item: SkillItem
  onSave: (input: Partial<SkillItem>) => Promise<unknown>
  onDelete: () => void
}) {
  const [form, setForm] = useState({ name: item.name, proficiency: item.proficiency ?? '' })
  const status = useAutosave(form, (value) =>
    onSave({ name: value.name, proficiency: value.proficiency || null, sort_order: item.sort_order }),
  )

  return (
    <div className="flex items-center gap-2">
      <Input
        className="h-8"
        value={form.name}
        onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
        placeholder="Skill"
      />
      <Input
        className="h-8 w-36"
        value={form.proficiency}
        onChange={(e) => setForm((f) => ({ ...f, proficiency: e.target.value }))}
        placeholder="Proficiency"
      />
      <AutosaveStatus status={status} compact className="shrink-0" />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-7 shrink-0 text-muted-foreground hover:text-destructive"
        onClick={onDelete}
      >
        <Trash2 className="size-3.5" />
      </Button>
    </div>
  )
}

function SkillGroupCard({
  resumeId,
  group,
  onSaveGroup,
  onDeleteGroup,
}: {
  resumeId: string
  group: SkillGroup
  onSaveGroup: (input: Partial<SkillGroup>) => Promise<unknown>
  onDeleteGroup: () => void
}) {
  const [groupName, setGroupName] = useState(group.group_name)
  const status = useAutosave(groupName, (value) => onSaveGroup({ group_name: value, sort_order: group.sort_order }))
  const items = useEntityMutations(
    resumeId,
    `/api/resumes/${resumeId}/skill-groups/${group.id}/items`,
  )

  return (
    <div className="rounded-md border border-border bg-card p-4">
      <div className="mb-3 flex items-center gap-2">
        <Input
          className="font-medium"
          value={groupName}
          onChange={(e) => setGroupName(e.target.value)}
          placeholder="Group name (e.g. Languages & Frameworks)"
        />
        <AutosaveStatus status={status} className="shrink-0" />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-8 shrink-0 text-muted-foreground hover:text-destructive"
          onClick={onDeleteGroup}
        >
          <Trash2 className="size-3.5" />
        </Button>
      </div>

      <div className="flex flex-col gap-2">
        <SortableList
          items={group.items}
          onReorder={(ids) => items.reorder.mutate(ids)}
          renderItem={(item) => (
            <SkillItemRow
              item={item}
              onSave={(input) => items.update.mutateAsync({ id: item.id, input })}
              onDelete={() => items.remove.mutate(item.id)}
            />
          )}
        />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="w-fit text-muted-foreground"
          onClick={() => items.create.mutate({ name: '', sort_order: group.items.length })}
        >
          <Plus className="size-3.5" /> Add skill
        </Button>
      </div>
    </div>
  )
}

export default function SkillsSection({ resumeId, items }: { resumeId: string; items: SkillGroup[] }) {
  const { create, update, remove, reorder } = useEntityMutations(resumeId, `/api/resumes/${resumeId}/skill-groups`)

  if (items.length === 0) {
    return (
      <EmptyState
        icon={Layers}
        message="No skill groups added yet."
        ctaLabel="Add skill group"
        onClick={() => create.mutate({ group_name: '', sort_order: items.length })}
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <SortableList
        items={items}
        onReorder={(ids) => reorder.mutate(ids)}
        renderItem={(group) => (
          <SkillGroupCard
            resumeId={resumeId}
            group={group}
            onSaveGroup={(input) => update.mutateAsync({ id: group.id, input })}
            onDeleteGroup={() => remove.mutate(group.id)}
          />
        )}
      />
      <Button
        type="button"
        variant="secondary"
        size="sm"
        className="w-fit"
        onClick={() => create.mutate({ group_name: '', sort_order: items.length })}
      >
        <Plus className="size-3.5" /> Add skill group
      </Button>
    </div>
  )
}
