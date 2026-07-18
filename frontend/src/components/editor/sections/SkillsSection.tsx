import { useEffect, useState, type ReactNode } from 'react'
import { Layers, Plus, Trash2 } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import RowCard from '@/components/editor/RowCard'
import SortableList from '@/components/editor/SortableList'
import EmptyState from '@/components/editor/EmptyState'
import AutosaveStatus from '@/components/editor/AutosaveStatus'
import { useAutosave } from '@/hooks/useAutosave'
import { useEntityMutations } from '@/hooks/useResumeEditor'
import { useEntryPanel } from '@/hooks/useEntryPanel'
import { usePreviewOverride } from '@/hooks/usePreviewOverride'
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
  dragHandle,
  open,
  isNew,
  onOpenChange,
}: {
  resumeId: string
  group: SkillGroup
  onSaveGroup: (input: Partial<SkillGroup>) => Promise<unknown>
  onDeleteGroup: () => void
  dragHandle?: ReactNode
  open: boolean
  isNew: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [groupName, setGroupName] = useState(group.group_name)
  const isDirty = groupName !== group.group_name
  const items = useEntityMutations(
    resumeId,
    `/api/resumes/${resumeId}/skill-groups/${group.id}/items`,
  )

  const { setOverride, clearOverride } = usePreviewOverride()
  useEffect(() => {
    if (!open) return
    return () => clearOverride(group.id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, group.id])
  useEffect(() => {
    if (!open) return
    setOverride(group.id, { group_name: groupName })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, groupName, group.id])

  return (
    <RowCard
      onDone={() => onSaveGroup({ group_name: groupName, sort_order: group.sort_order })}
      onDiscard={() => setGroupName(group.group_name)}
      isDirty={isDirty}
      onDelete={onDeleteGroup}
      title={groupName}
      subtitle={group.items.length > 0 ? `${group.items.length} skill${group.items.length === 1 ? '' : 's'}` : undefined}
      dragHandle={dragHandle}
      editorTitle="Edit skill group"
      open={open}
      isNew={isNew}
      onOpenChange={onOpenChange}
    >
      <div className="flex flex-col gap-1.5">
        <Label>Group name</Label>
        <Input
          className="font-medium"
          value={groupName}
          onChange={(e) => setGroupName(e.target.value)}
          placeholder="Group name (e.g. Languages & Frameworks)"
        />
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
    </RowCard>
  )
}

export default function SkillsSection({ resumeId, items }: { resumeId: string; items: SkillGroup[] }) {
  const { create, update, remove, reorder } = useEntityMutations(resumeId, `/api/resumes/${resumeId}/skill-groups`)
  const { openNew, getPanelProps } = useEntryPanel(items.map((i) => i.id))

  if (items.length === 0) {
    return (
      <EmptyState
        icon={Layers}
        message="No skill groups added yet."
        ctaLabel="Add skill group"
        onClick={() => {
          const id = create.mutate({ group_name: '', sort_order: items.length })
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
        renderItem={(group, _index, dragHandle) => (
          <SkillGroupCard
            resumeId={resumeId}
            group={group}
            onSaveGroup={(input) => update.mutateAsync({ id: group.id, input })}
            onDeleteGroup={() => remove.mutate(group.id)}
            dragHandle={dragHandle}
            {...getPanelProps(group.id)}
          />
        )}
      />
      <Button
        type="button"
        variant="secondary"
        size="sm"
        className="w-fit"
        onClick={() => {
          const id = create.mutate({ group_name: '', sort_order: items.length })
          openNew(id)
        }}
      >
        <Plus className="size-3.5" /> Add skill group
      </Button>
    </div>
  )
}
