import { useEffect, useRef, useState, type ReactNode } from 'react'
import {
  DndContext,
  DragOverlay,
  closestCorners,
  PointerSensor,
  useSensor,
  useSensors,
  useDroppable,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { SortableContext, useSortable, verticalListSortingStrategy, arrayMove } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical } from 'lucide-react'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import SortableList from '@/components/editor/SortableList'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import { useTemplates } from '@/hooks/useTemplates'
import { useDesignSchema } from '@/hooks/useDesignSchema'
import type { ResumeDesign, SectionRef } from '@/types/design'
import type { CustomSection, FullResume } from '@/types/resume'
import type { FieldDef, FieldGroup } from '@/types/designSchema'
import type { Template } from '@/types/template'
import { resolveSectionRefs, resolveTwoColumnSectionRefs, sectionRefKey, SECTION_LABELS } from '@/lib/sectionOrder'
import { getFieldValue, buildFieldPatch } from '@/lib/designField'
import { cn } from '@/lib/utils'

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}

function stepValue(value: number, delta: number, min: number, max: number, step: number) {
  const precision = step < 1 ? String(step).split('.')[1]?.length ?? 0 : 0
  const next = Number((value + delta).toFixed(precision))
  return Math.min(max, Math.max(min, next))
}

function NumberField({
  label,
  value,
  onChange,
  min = 0,
  max = 100,
  step = 1,
}: {
  label: string
  value: number
  onChange: (v: number) => void
  min?: number
  max?: number
  step?: number
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between gap-2">
        <Label>{label}</Label>
        {/* Slider is bounded to [min, max], but typing an exact value here
            isn't clamped to that range — the range is a sensible default,
            not a hard backend limit. */}
        <input
          type="number"
          value={value}
          step={step}
          onChange={(e) => {
            const n = Number(e.target.value)
            if (!Number.isNaN(n)) onChange(n)
          }}
          className="h-7 w-16 rounded border border-input bg-transparent px-1.5 text-right text-sm tabular-nums"
        />
      </div>
      <div className="flex items-center gap-2">
        <input
          type="range"
          min={min}
          max={max}
          step={step}
          value={Math.min(max, Math.max(min, value))}
          onChange={(e) => onChange(Number(e.target.value))}
          className="h-1.5 flex-1 cursor-pointer accent-accent"
        />
        <div className="flex shrink-0 gap-1">
          <button
            type="button"
            aria-label={`Decrease ${label}`}
            onClick={() => onChange(stepValue(value, -step, min, max, step))}
            className="flex size-6 items-center justify-center rounded border border-input text-muted-foreground hover:bg-secondary"
          >
            −
          </button>
          <button
            type="button"
            aria-label={`Increase ${label}`}
            onClick={() => onChange(stepValue(value, step, min, max, step))}
            className="flex size-6 items-center justify-center rounded border border-input text-muted-foreground hover:bg-secondary"
          >
            +
          </button>
        </div>
      </div>
    </div>
  )
}

function ColorField({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <Field label={label}>
      <div className="flex items-center gap-2">
        <input
          type="color"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="h-9 w-9 shrink-0 cursor-pointer rounded border border-border bg-transparent p-0.5"
        />
        <Input value={value} onChange={(e) => onChange(e.target.value)} className="font-mono text-xs" />
      </div>
    </Field>
  )
}

function BoolField({ label, value, onChange }: { label: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="flex items-center justify-between gap-3 rounded-lg border border-border bg-card px-3 py-2 text-sm">
      {label}
      <Switch checked={value} onCheckedChange={onChange} />
    </label>
  )
}

function TemplatePicker({ design, onChange }: { design: ResumeDesign; onChange: (patch: Partial<ResumeDesign>) => void }) {
  const { data: templates, isLoading } = useTemplates()
  if (isLoading) return <p className="text-sm text-muted-foreground">Loading templates…</p>
  if (!templates || templates.length === 0) return null

  function applyTemplate(t: Template) {
    onChange({ ...t.default_design, templateId: t.id, sectionOrder: design.sectionOrder })
  }

  return (
    <div className="grid grid-cols-2 gap-3">
      {templates.map((t) => (
        <button
          key={t.id}
          type="button"
          onClick={() => applyTemplate(t)}
          className={cn(
            'flex flex-col items-center gap-2 rounded-lg border p-3 text-sm transition-colors',
            design.templateId === t.id ? 'border-accent ring-1 ring-accent' : 'border-border hover:border-foreground/30',
          )}
        >
          <div className="flex h-16 w-12 items-center justify-center rounded bg-secondary text-[10px] text-muted-foreground">Aa</div>
          {t.name}
        </button>
      ))}
    </div>
  )
}

function SectionRefList({
  refs,
  customSections,
  onReorder,
  onToggleVisible,
  onSetTitleOverride,
}: {
  refs: SectionRef[]
  customSections: CustomSection[]
  onReorder: (orderedIds: string[]) => void
  onToggleVisible: (key: string) => void
  onSetTitleOverride: (key: string, value: string) => void
}) {
  if (refs.length === 0) {
    return <p className="text-sm text-muted-foreground">Nothing here yet.</p>
  }
  return (
    <SortableList
      dragHandlePlacement="inline"
      items={refs.map((r) => ({ id: sectionRefKey(r) }))}
      onReorder={onReorder}
      renderItem={(item, _index, dragHandle) => {
        const ref = refs.find((r) => sectionRefKey(r) === item.id)!
        const isCustom = ref.sectionType === 'custom'
        const customSection = isCustom ? customSections.find((s) => s.id === ref.customSectionId) : undefined
        const defaultLabel = isCustom ? customSection?.title || 'Untitled section' : (SECTION_LABELS[ref.sectionType] ?? ref.sectionType)
        return (
          <div className="flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2">
            {dragHandle}
            <Input
              value={ref.titleOverride ?? ''}
              placeholder={defaultLabel}
              onChange={(e) => onSetTitleOverride(item.id, e.target.value)}
              className="h-8 flex-1"
            />
            <Switch checked={ref.isVisible} onCheckedChange={() => onToggleVisible(item.id)} />
          </div>
        )
      }}
    />
  )
}

function sectionRefListHandlers(refs: SectionRef[], persist: (newRefs: SectionRef[]) => void) {
  return {
    onReorder: (orderedIds: string[]) => {
      const byKey = new Map(refs.map((r) => [sectionRefKey(r), r]))
      persist(orderedIds.map((id) => byKey.get(id)!))
    },
    onToggleVisible: (key: string) => persist(refs.map((r) => (sectionRefKey(r) === key ? { ...r, isVisible: !r.isVisible } : r))),
    onSetTitleOverride: (key: string, value: string) =>
      persist(refs.map((r) => (sectionRefKey(r) === key ? { ...r, titleOverride: value || null } : r))),
  }
}

function sectionDefaultLabel(ref: SectionRef, customSections: CustomSection[]): string {
  if (ref.sectionType !== 'custom') return SECTION_LABELS[ref.sectionType] ?? ref.sectionType
  return customSections.find((s) => s.id === ref.customSectionId)?.title || 'Untitled section'
}

function SectionRowContent({
  dragHandle,
  ref,
  defaultLabel,
  onSetTitleOverride,
  onToggleVisible,
  overlay,
}: {
  dragHandle: ReactNode
  ref: SectionRef
  defaultLabel: string
  onSetTitleOverride?: (key: string, value: string) => void
  onToggleVisible?: (key: string) => void
  overlay?: boolean
}) {
  const key = sectionRefKey(ref)
  return (
    <div className={cn('flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2', overlay && 'shadow-lg')}>
      {dragHandle}
      <Input
        value={ref.titleOverride ?? ''}
        placeholder={defaultLabel}
        onChange={(e) => onSetTitleOverride?.(key, e.target.value)}
        readOnly={overlay}
        className="h-8 flex-1"
      />
      <Switch checked={ref.isVisible} onCheckedChange={() => onToggleVisible?.(key)} disabled={overlay} />
    </div>
  )
}

function DragHandle(props: React.ComponentProps<'button'>) {
  return (
    <button
      type="button"
      aria-label="Drag to move"
      onClick={(e) => e.stopPropagation()}
      className="-my-1 flex size-6 shrink-0 cursor-grab items-center justify-center rounded text-muted-foreground/50 hover:bg-muted hover:text-muted-foreground active:cursor-grabbing"
      {...props}
    >
      <GripVertical className="size-4" />
    </button>
  )
}

function SortableSectionRow({
  sectionRef,
  defaultLabel,
  onSetTitleOverride,
  onToggleVisible,
}: {
  sectionRef: SectionRef
  defaultLabel: string
  onSetTitleOverride: (key: string, value: string) => void
  onToggleVisible: (key: string) => void
}) {
  const id = sectionRefKey(sectionRef)
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id })
  return (
    <div ref={setNodeRef} style={{ transform: CSS.Transform.toString(transform), transition }} className={cn(isDragging && 'opacity-30')}>
      <SectionRowContent
        dragHandle={<DragHandle {...attributes} {...listeners} />}
        ref={sectionRef}
        defaultLabel={defaultLabel}
        onSetTitleOverride={onSetTitleOverride}
        onToggleVisible={onToggleVisible}
      />
    </div>
  )
}

function SectionColumn({
  id,
  label,
  refs,
  customSections,
  onSetTitleOverride,
  onToggleVisible,
}: {
  id: 'left' | 'right'
  label: string
  refs: SectionRef[]
  customSections: CustomSection[]
  onSetTitleOverride: (key: string, value: string) => void
  onToggleVisible: (key: string) => void
}) {
  const { setNodeRef } = useDroppable({ id })
  return (
    <div className="flex flex-col gap-2">
      <h4 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">{label}</h4>
      <SortableContext items={refs.map(sectionRefKey)} strategy={verticalListSortingStrategy}>
        <div ref={setNodeRef} className="flex min-h-16 flex-col gap-2 rounded-lg border border-dashed border-transparent p-0.5">
          {refs.length === 0 ? (
            <p className="rounded-lg border border-dashed border-border px-3 py-4 text-center text-xs text-muted-foreground">
              Drop a section here
            </p>
          ) : (
            refs.map((ref) => (
              <SortableSectionRow
                key={sectionRefKey(ref)}
                sectionRef={ref}
                defaultLabel={sectionDefaultLabel(ref, customSections)}
                onSetTitleOverride={onSetTitleOverride}
                onToggleVisible={onToggleVisible}
              />
            ))
          )}
        </div>
      </SortableContext>
    </div>
  )
}

type Columns = { left: SectionRef[]; right: SectionRef[] }

function findColumn(id: string, columns: Columns): 'left' | 'right' | undefined {
  if (id === 'left' || id === 'right') return id
  if (columns.left.some((r) => sectionRefKey(r) === id)) return 'left'
  if (columns.right.some((r) => sectionRefKey(r) === id)) return 'right'
  return undefined
}

function TwoColumnSectionsPanel({ data, dispatch }: { data: FullResume; dispatch: (action: { type: 'design_update'; patch: Partial<ResumeDesign> }) => void }) {
  const [dragColumns, setDragColumns] = useState<Columns | null>(null)
  const [activeId, setActiveId] = useState<string | null>(null)
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 4 } }))

  const columns = dragColumns ?? resolveTwoColumnSectionRefs(data)

  function persist(final: Columns) {
    dispatch({
      type: 'design_update',
      patch: { sectionOrder: { ...data.design.sectionOrder, two: { left: final.left, right: final.right } } },
    })
  }

  function handleDragStart(event: DragStartEvent) {
    setActiveId(String(event.active.id))
    setDragColumns(resolveTwoColumnSectionRefs(data))
  }

  function handleDragOver(event: DragOverEvent) {
    const { active, over } = event
    if (!over) return
    setDragColumns((prev) => {
      if (!prev) return prev
      const activeId = String(active.id)
      const overId = String(over.id)
      const activeColumn = findColumn(activeId, prev)
      const overColumn = findColumn(overId, prev)
      if (!activeColumn || !overColumn || activeColumn === overColumn) return prev

      const activeItems = prev[activeColumn]
      const overItems = prev[overColumn]
      const activeIndex = activeItems.findIndex((r) => sectionRefKey(r) === activeId)
      if (activeIndex === -1) return prev
      const overIndex = overItems.findIndex((r) => sectionRefKey(r) === overId)
      const insertAt = overIndex >= 0 ? overIndex : overItems.length
      const moved = activeItems[activeIndex]
      return {
        ...prev,
        [activeColumn]: activeItems.filter((_, i) => i !== activeIndex),
        [overColumn]: [...overItems.slice(0, insertAt), moved, ...overItems.slice(insertAt)],
      }
    })
  }

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    let final = dragColumns ?? resolveTwoColumnSectionRefs(data)

    if (over) {
      const activeId = String(active.id)
      const overId = String(over.id)
      const column = findColumn(activeId, final)
      const overColumn = findColumn(overId, final)
      if (column && overColumn === column) {
        const items = final[column]
        const activeIndex = items.findIndex((r) => sectionRefKey(r) === activeId)
        const overIndex = items.findIndex((r) => sectionRefKey(r) === overId)
        if (activeIndex !== -1 && overIndex !== -1 && activeIndex !== overIndex) {
          final = { ...final, [column]: arrayMove(items, activeIndex, overIndex) }
        }
      }
    }

    persist(final)
    setDragColumns(null)
    setActiveId(null)
  }

  function toggleVisible(key: string) {
    const cur = resolveTwoColumnSectionRefs(data)
    const column = findColumn(key, cur)
    if (!column) return
    persist({ ...cur, [column]: cur[column].map((r) => (sectionRefKey(r) === key ? { ...r, isVisible: !r.isVisible } : r)) })
  }

  function setTitleOverride(key: string, value: string) {
    const cur = resolveTwoColumnSectionRefs(data)
    const column = findColumn(key, cur)
    if (!column) return
    persist({ ...cur, [column]: cur[column].map((r) => (sectionRefKey(r) === key ? { ...r, titleOverride: value || null } : r)) })
  }

  const activeRef = activeId ? [...columns.left, ...columns.right].find((r) => sectionRefKey(r) === activeId) : undefined

  return (
    <div className="flex flex-col gap-5">
      <p className="text-xs text-muted-foreground">
        This template has two columns — drag sections between them, reorder within a column, and show/hide or rename
        each. A section only shows up here once it actually has content — add or remove sections in Content.
      </p>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragOver={handleDragOver}
        onDragEnd={handleDragEnd}
      >
        <div className="grid grid-cols-2 gap-4">
          <SectionColumn
            id="left"
            label="Sidebar"
            refs={columns.left}
            customSections={data.custom_sections}
            onSetTitleOverride={setTitleOverride}
            onToggleVisible={toggleVisible}
          />
          <SectionColumn
            id="right"
            label="Main"
            refs={columns.right}
            customSections={data.custom_sections}
            onSetTitleOverride={setTitleOverride}
            onToggleVisible={toggleVisible}
          />
        </div>
        <DragOverlay>
          {activeRef && (
            <SectionRowContent
              dragHandle={<DragHandle />}
              ref={activeRef}
              defaultLabel={sectionDefaultLabel(activeRef, data.custom_sections)}
              overlay
            />
          )}
        </DragOverlay>
      </DndContext>
    </div>
  )
}

function SectionsPanel() {
  const { state, dispatch } = useResumeDraftContext()
  const data = state.data

  if (data.design.layout.mode === 'two') {
    return <TwoColumnSectionsPanel data={data} dispatch={dispatch} />
  }

  const refs = resolveSectionRefs(data)
  const persist = (newRefs: SectionRef[]) =>
    dispatch({
      type: 'design_update',
      patch: { sectionOrder: { ...data.design.sectionOrder, one: { sections: newRefs } } },
    })

  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-muted-foreground">
        Order, show/hide, and rename the sections that appear on your resume. A section only shows up here once it
        actually has content — add or remove sections in Content.
      </p>
      <SectionRefList refs={refs} customSections={data.custom_sections} {...sectionRefListHandlers(refs, persist)} />
    </div>
  )
}

function SchemaField({
  field,
  design,
  onChange,
  options,
}: {
  field: FieldDef
  design: ResumeDesign
  onChange: (patch: Partial<ResumeDesign>) => void
  options?: string[]
}) {
  const value = getFieldValue(design, field.key)

  if (field.type === 'bool') {
    return <BoolField label={field.label} value={value as boolean} onChange={(v) => onChange(buildFieldPatch(design, field.key, v))} />
  }

  if (field.type === 'enum') {
    return (
      <Field label={field.label}>
        <Select value={value as string} onValueChange={(v) => onChange(buildFieldPatch(design, field.key, v))}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {(options ?? field.options ?? []).map((o) => (
              <SelectItem key={o} value={o}>
                {o}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>
    )
  }

  if (field.type === 'color') {
    return <ColorField label={field.label} value={value as string} onChange={(v) => onChange(buildFieldPatch(design, field.key, v))} />
  }

  return (
    <NumberField
      label={field.label}
      value={value as number}
      min={field.min}
      max={field.max}
      step={field.step}
      onChange={(v) => onChange(buildFieldPatch(design, field.key, v))}
    />
  )
}

function SchemaGroupPanel({
  group,
  design,
  onChange,
  layoutModeOptions,
}: {
  group: FieldGroup
  design: ResumeDesign
  onChange: (patch: Partial<ResumeDesign>) => void
  layoutModeOptions?: string[]
}) {
  const boolFields = group.fields.filter((f) => f.type === 'bool')
  const otherFields = group.fields.filter((f) => f.type !== 'bool')

  return (
    <div className="flex flex-col gap-4">
      {otherFields.length > 0 && (
        <div className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
          {otherFields.map((field) => (
            <SchemaField
              key={field.key}
              field={field}
              design={design}
              onChange={onChange}
              options={field.key === 'layout.mode' ? layoutModeOptions : undefined}
            />
          ))}
        </div>
      )}
      {boolFields.length > 0 && (
        <div className="flex flex-col gap-2">
          {boolFields.map((field) => (
            <SchemaField key={field.key} field={field} design={design} onChange={onChange} />
          ))}
        </div>
      )}
    </div>
  )
}

const PINNED_ITEMS = [
  { key: 'templates', label: 'Templates' },
  { key: 'sections', label: 'Sections' },
] as const

const SCROLLSPY_ROOT_MARGIN = '0px 0px -75% 0px'
const CLICK_SCROLL_SUPPRESS_MS = 700

export default function DesignMode() {
  const { state, dispatch } = useResumeDraftContext()
  const design = state.data.design
  const { data: schema, isLoading: schemaLoading } = useDesignSchema()
  const { data: templates } = useTemplates()
  const [activeCategory, setActiveCategory] = useState<string>('templates')
  const sectionElsRef = useRef<Map<string, HTMLElement>>(new Map())
  const suppressObserverUntilRef = useRef(0)

  function update(patch: Partial<ResumeDesign>) {
    dispatch({ type: 'design_update', patch })
  }

  const activeTemplate = templates?.find((t) => t.id === design.templateId)
  const visibleGroups = (schema ?? []).filter((group) => {
    if (!activeTemplate) return true
    if (!activeTemplate.supported_groups.includes(group.key)) return false
    if (group.key === 'layout' && activeTemplate.supported_modes.length <= 1) return false
    return true
  })

  const navItems = [...PINNED_ITEMS, ...visibleGroups.map((g) => ({ key: g.key, label: g.label }))]
  const navKeys = navItems.map((n) => n.key).join(',')

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (Date.now() < suppressObserverUntilRef.current) return
        const visible = entries.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
        const top = visible[0]?.target.getAttribute('data-nav-key')
        if (top) setActiveCategory(top)
      },
      { rootMargin: SCROLLSPY_ROOT_MARGIN, threshold: 0 },
    )
    for (const el of sectionElsRef.current.values()) observer.observe(el)
    return () => observer.disconnect()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [navKeys])

  function registerSection(key: string, el: HTMLElement | null) {
    if (el) sectionElsRef.current.set(key, el)
    else sectionElsRef.current.delete(key)
  }

  function goToSection(key: string) {
    setActiveCategory(key)
    suppressObserverUntilRef.current = Date.now() + CLICK_SCROLL_SUPPRESS_MS
    sectionElsRef.current.get(key)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  return (
    <div className="flex gap-6">
      <nav className="sticky top-0 flex max-h-svh w-36 shrink-0 flex-col gap-0.5 self-start overflow-y-auto">
        {navItems.map((item) => (
          <button
            key={item.key}
            type="button"
            onClick={() => goToSection(item.key)}
            className={cn(
              'rounded-md px-3 py-2 text-left text-sm transition-colors',
              activeCategory === item.key ? 'bg-secondary font-medium text-foreground' : 'text-muted-foreground hover:bg-secondary/60',
            )}
          >
            {item.label}
          </button>
        ))}
      </nav>

      <div className="flex min-w-0 flex-1 flex-col gap-8">
        <section ref={(el) => registerSection('templates', el)} data-nav-key="templates" className="flex flex-col gap-3">
          <h3 className="text-sm font-semibold text-foreground">Templates</h3>
          <TemplatePicker design={design} onChange={update} />
        </section>

        <section ref={(el) => registerSection('sections', el)} data-nav-key="sections" className="flex flex-col gap-3">
          <h3 className="text-sm font-semibold text-foreground">Sections</h3>
          <SectionsPanel />
        </section>

        {schemaLoading && <p className="text-sm text-muted-foreground">Loading design options…</p>}
        {visibleGroups.map((group) => (
          <section key={group.key} ref={(el) => registerSection(group.key, el)} data-nav-key={group.key} className="flex flex-col gap-3">
            <h3 className="text-sm font-semibold text-foreground">{group.label}</h3>
            <SchemaGroupPanel
              group={group}
              design={design}
              onChange={update}
              layoutModeOptions={group.key === 'layout' ? activeTemplate?.supported_modes : undefined}
            />
          </section>
        ))}
      </div>
    </div>
  )
}
