import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Undo2, Redo2, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import SortableList from '@/components/editor/SortableList'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import { useTemplates } from '@/hooks/useTemplates'
import { useDesignSchema } from '@/hooks/useDesignSchema'
import type { ResumeDesign, SectionRef } from '@/types/design'
import type { CustomSection } from '@/types/resume'
import type { FieldDef, FieldGroup } from '@/types/designSchema'
import type { Template } from '@/types/template'
import { getTemplate, selectTemplate, supportsTemplateField } from '@/templates/catalog'
import { resolveSectionRefs, sectionRefKey, SECTION_LABELS } from '@/lib/sectionOrder'
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
  const [inputValue, setInputValue] = useState(String(value))
  const focused = useRef(false)
  useEffect(() => { if (!focused.current) setInputValue(String(value)) }, [value])
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between gap-2">
        <Label>{label}</Label>
        {/* Let users finish typing a number before clamping; incomplete input
            never enters the synced document or PDF pipeline. */}
        <input
          type="number"
          aria-label={label}
          value={inputValue}
          min={min}
          max={max}
          step={step}
          onFocus={() => { focused.current = true }}
          onChange={(e) => {
            setInputValue(e.target.value)
            const n = Number(e.target.value)
            if (e.target.value !== '' && Number.isFinite(n) && n >= min && n <= max) onChange(n)
          }}
          onBlur={() => {
            focused.current = false
            const parsed = inputValue.trim() === '' ? value : Number(inputValue)
            const next = Number.isFinite(parsed) ? Math.min(max, Math.max(min, parsed)) : value
            setInputValue(String(next))
            if (next !== value) onChange(next)
          }}
          className="h-7 w-16 rounded bg-secondary px-1.5 text-right text-sm tabular-nums"
        />
      </div>
      <div className="flex items-center gap-2">
        <input
          type="range"
          aria-label={`Adjust ${label}`}
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
            className="flex size-6 items-center justify-center rounded text-muted-foreground hover:bg-secondary"
          >
            −
          </button>
          <button
            type="button"
            aria-label={`Increase ${label}`}
            onClick={() => onChange(stepValue(value, step, min, max, step))}
            className="flex size-6 items-center justify-center rounded text-muted-foreground hover:bg-secondary"
          >
            +
          </button>
        </div>
      </div>
    </div>
  )
}

function ColorField({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  const [inputValue, setInputValue] = useState(value)
  const focused = useRef(false)
  useEffect(() => { if (!focused.current) setInputValue(value) }, [value])
  return (
    <Field label={label}>
      <div className="flex items-center gap-2">
        <input
          type="color"
          aria-label={`Pick ${label}`}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="h-9 w-9 shrink-0 cursor-pointer rounded bg-transparent p-0.5"
        />
        <Input aria-label={`${label} hex color`} value={inputValue} onFocus={() => { focused.current = true }} onChange={(e) => {
          setInputValue(e.target.value)
          if (/^#[0-9a-f]{6}$/i.test(e.target.value)) onChange(e.target.value)
        }} onBlur={() => { focused.current = false; if (!/^#[0-9a-f]{6}$/i.test(inputValue)) setInputValue(value) }} className="font-mono text-xs" />
      </div>
    </Field>
  )
}

function BoolField({ label, value, onChange }: { label: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="flex items-center justify-between gap-3 rounded-lg bg-card px-3 py-2 text-sm">
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
    onChange(selectTemplate(design, t.id))
  }

  return (
    <div className="grid grid-cols-2 gap-3">
      {templates.map((t) => (
        <button
          key={t.id}
          type="button"
          onClick={() => applyTemplate(t)}
          aria-pressed={design.templateId === t.id}
          className={cn(
            'flex flex-col items-center gap-2 rounded-lg p-3 text-sm transition-colors',
            design.templateId === t.id ? 'bg-secondary text-foreground' : 'bg-card hover:bg-secondary/60',
          )}
        >
          <div aria-hidden className="flex h-24 w-16 flex-col gap-1 rounded-sm bg-white p-2 shadow-sm">
            <span className="mb-1 h-1.5 w-8 rounded-sm bg-slate-700" />
            {[0, 1, 2].map((row) => <div key={row} className="mt-1 flex flex-col gap-1"><span className="h-1 w-7 rounded-sm bg-violet-300" /><span className="h-0.5 w-full bg-slate-200" /><span className="h-0.5 w-4/5 bg-slate-200" /></div>)}
          </div>
          {t.name}
          <span className="text-[10px] text-muted-foreground">{t.description}</span>
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
          <div className="flex items-center gap-2 rounded-lg bg-card px-3 py-2">
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

function SectionsPanel() {
  const { state, dispatch } = useResumeDraftContext()
  const data = state.data


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
          <SelectTrigger aria-label={field.label}>
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
  { key: 'page', label: 'Page & presets' },
  { key: 'sections', label: 'Sections' },
] as const

const SCROLLSPY_ROOT_MARGIN = '0px 0px -75% 0px'
const CLICK_SCROLL_SUPPRESS_MS = 700

export default function DesignMode() {
  const { state, dispatch, undoDesign, redoDesign, canUndoDesign, canRedoDesign } = useResumeDraftContext()
  const design = state.data.design
  const { data: schema, isLoading: schemaLoading } = useDesignSchema()
  const [activeCategory, setActiveCategory] = useState<string>('templates')
  const sectionElsRef = useRef<Map<string, HTMLElement>>(new Map())
  const suppressObserverUntilRef = useRef(0)

  function update(patch: Partial<ResumeDesign>) {
    dispatch({ type: 'design_update', patch })
  }

  const activeTemplate = getTemplate(design.templateId)
  const visibleGroups = (schema ?? []).filter((group) => group.key !== 'paper').map((group) => ({
    ...group, fields: group.fields.filter((field) => supportsTemplateField(activeTemplate, field.key)),
  })).filter((group) => group.fields.length > 0)

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
    sectionElsRef.current.get(key)?.scrollIntoView({ behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth', block: 'start' })
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
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-xs text-muted-foreground">Changes update your PDF automatically.</span>
          <div className="flex flex-wrap items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => dispatch({ type: 'restore_design_defaults' })}
              title={`Restore ${activeTemplate.name}'s default formatting and layout. Keeps resume content, section visibility, and custom titles.`}
            >
              <RotateCcw className="size-3.5" aria-hidden />
              Restore defaults
            </Button>
            <div className="flex gap-1" role="group" aria-label="Formatting history">
              <button type="button" onClick={undoDesign} disabled={!canUndoDesign} aria-label="Undo formatting" title="Undo formatting" className="rounded-lg p-2 disabled:opacity-30"><Undo2 className="size-3.5" /></button>
              <button type="button" onClick={redoDesign} disabled={!canRedoDesign} aria-label="Redo formatting" title="Redo formatting" className="rounded-lg p-2 disabled:opacity-30"><Redo2 className="size-3.5" /></button>
            </div>
          </div>
        </div>
        <section ref={(el) => registerSection('templates', el)} data-nav-key="templates" className="flex flex-col gap-3">
          <h3 className="text-sm font-semibold text-foreground">Templates</h3>
          <TemplatePicker design={design} onChange={update} />
        </section>

        <section ref={(el) => registerSection('page', el)} data-nav-key="page" className="flex flex-col gap-4">
          <h3 className="text-sm font-semibold">Page & formatting presets</h3>
          <Field label="Paper size">
            <Select value={design.page.format} onValueChange={(format) => update({ page: { ...design.page, format: format as 'Letter' | 'A4' } })}>
              <SelectTrigger aria-label="Paper size"><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="Letter">US Letter · 8.5 × 11 in</SelectItem><SelectItem value="A4">A4 · 210 × 297 mm</SelectItem></SelectContent>
            </Select>
          </Field>
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
            {[
              { label: 'Compact', description: 'More room for experience', lineHeight: 1.3, sectionGap: 12, entryGap: 7, bulletGap: 2 },
              { label: 'Balanced', description: 'Comfortable and clear', lineHeight: 1.5, sectionGap: 18, entryGap: 10, bulletGap: 4 },
              { label: 'Spacious', description: 'A little more breathing room', lineHeight: 1.65, sectionGap: 24, entryGap: 14, bulletGap: 6 },
            ].map((preset) => <button key={preset.label} type="button" className="format-preset" onClick={() => update({ typography: { ...design.typography, lineHeight: preset.lineHeight }, spacing: { sectionGap: preset.sectionGap, entryGap: preset.entryGap, bulletGap: preset.bulletGap } })}><strong>{preset.label}</strong><span className="text-[10px] text-muted-foreground">{preset.description}</span></button>)}
          </div>
          <p className="text-[11px] leading-relaxed text-muted-foreground">Fonts are embedded in your PDF. Georgia uses Gelasio; Times New Roman uses Tinos; Arial and Helvetica use Arimo. Other listed fonts use their own bundled files.</p>
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
