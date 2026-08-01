import type { ReactNode } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import SortableList from '@/components/editor/SortableList'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import { useTemplates } from '@/hooks/useTemplates'
import { FONT_FAMILIES, HEADING_STYLES, CAPITALIZATIONS, type ResumeDesign, type SectionRef } from '@/types/design'
import { resolveSectionRefs, sectionRefKey, SECTION_LABELS } from '@/lib/sectionOrder'
import { tempId } from '@/lib/tempId'
import { cn } from '@/lib/utils'

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}

function NumberField({
  label,
  value,
  onChange,
  step = 1,
  min,
}: {
  label: string
  value: number
  onChange: (v: number) => void
  step?: number
  min?: number
}) {
  return (
    <Field label={label}>
      <Input
        type="number"
        value={value}
        step={step}
        min={min}
        onChange={(e) => {
          const n = Number(e.target.value)
          if (!Number.isNaN(n)) onChange(n)
        }}
      />
    </Field>
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

function TemplatePicker({ design, onChange }: { design: ResumeDesign; onChange: (patch: Partial<ResumeDesign>) => void }) {
  const { data: templates, isLoading } = useTemplates()
  if (isLoading) return <p className="text-sm text-muted-foreground">Loading templates…</p>
  if (!templates || templates.length === 0) return null

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
      {templates.map((t) => (
        <button
          key={t.id}
          type="button"
          onClick={() => onChange({ templateId: t.id })}
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

function SectionsPanel() {
  const { state, dispatch } = useResumeDraftContext()
  const data = state.data
  const refs = resolveSectionRefs(data)

  function persist(newRefs: SectionRef[]) {
    dispatch({
      type: 'design_update',
      patch: { sectionOrder: { ...data.design.sectionOrder, one: { sections: newRefs } } },
    })
  }

  function toggleVisible(key: string) {
    persist(refs.map((r) => (sectionRefKey(r) === key ? { ...r, isVisible: !r.isVisible } : r)))
  }

  function setTitleOverride(key: string, value: string) {
    persist(refs.map((r) => (sectionRefKey(r) === key ? { ...r, titleOverride: value || null } : r)))
  }

  function addCustomSection() {
    const id = tempId()
    dispatch({ type: 'custom_section_create', tempId: id, fields: { title: 'New section', sort_order: data.custom_sections.length } })
    persist([...refs, { sectionType: 'custom', customSectionId: id, isVisible: true, titleOverride: null }])
  }

  function removeCustomSection(customSectionId: string) {
    dispatch({ type: 'custom_section_delete', id: customSectionId })
  }

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-foreground">Sections</h3>
        <Button type="button" variant="secondary" size="sm" onClick={addCustomSection}>
          <Plus className="size-3.5" /> Add custom section
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">
        Order, show/hide, and rename the sections that appear on your resume. A default section (Experience, Skills,
        etc.) only shows up here once it actually has content — add it in Content first.
      </p>
      {refs.length === 0 ? (
        <p className="text-sm text-muted-foreground">Nothing to order yet — add some content first.</p>
      ) : (
        <SortableList
          dragHandlePlacement="inline"
          items={refs.map((r) => ({ id: sectionRefKey(r) }))}
          onReorder={(orderedIds) => {
            const byKey = new Map(refs.map((r) => [sectionRefKey(r), r]))
            persist(orderedIds.map((id) => byKey.get(id)!))
          }}
          renderItem={(item, _index, dragHandle) => {
            const ref = refs.find((r) => sectionRefKey(r) === item.id)!
            const isCustom = ref.sectionType === 'custom'
            const customSection = isCustom ? data.custom_sections.find((s) => s.id === ref.customSectionId) : undefined
            const defaultLabel = isCustom ? customSection?.title || 'Untitled section' : (SECTION_LABELS[ref.sectionType] ?? ref.sectionType)
            return (
              <div className="flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2">
                {dragHandle}
                <Input
                  value={ref.titleOverride ?? ''}
                  placeholder={defaultLabel}
                  onChange={(e) => setTitleOverride(item.id, e.target.value)}
                  className="h-8 flex-1"
                />
                <Switch checked={ref.isVisible} onCheckedChange={() => toggleVisible(item.id)} />
                {isCustom && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-8 shrink-0 text-muted-foreground hover:text-destructive"
                    onClick={() => removeCustomSection(ref.customSectionId!)}
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                )}
              </div>
            )
          }}
        />
      )}
    </section>
  )
}

export default function DesignMode() {
  const { state, dispatch } = useResumeDraftContext()
  const design = state.data.design

  function update(patch: Partial<ResumeDesign>) {
    dispatch({ type: 'design_update', patch })
  }

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold text-foreground">Template</h3>
        <TemplatePicker design={design} onChange={update} />
      </section>

      <SectionsPanel />

      <section className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold text-foreground">Typography</h3>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Font family">
            <Select
              value={design.typography.fontFamily}
              onValueChange={(v) => update({ typography: { ...design.typography, fontFamily: v } })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FONT_FAMILIES.map((f) => (
                  <SelectItem key={f} value={f}>
                    {f}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <NumberField
            label="Base size (px)"
            value={design.typography.baseFontSizePt}
            min={8}
            onChange={(v) => update({ typography: { ...design.typography, baseFontSizePt: v } })}
          />
          <NumberField
            label="Line height"
            value={design.typography.lineHeight}
            step={0.05}
            min={1}
            onChange={(v) => update({ typography: { ...design.typography, lineHeight: v } })}
          />
          <NumberField
            label="Name size (px)"
            value={design.typography.nameFontSizePt}
            min={8}
            onChange={(v) => update({ typography: { ...design.typography, nameFontSizePt: v } })}
          />
          <NumberField
            label="Section heading size (px)"
            value={design.typography.sectionHeadingFontSizePt}
            min={8}
            onChange={(v) => update({ typography: { ...design.typography, sectionHeadingFontSizePt: v } })}
          />
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold text-foreground">Colors</h3>
        <div className="grid grid-cols-3 gap-3">
          <ColorField label="Text" value={design.colors.text} onChange={(v) => update({ colors: { ...design.colors, text: v } })} />
          <ColorField label="Accent" value={design.colors.accent} onChange={(v) => update({ colors: { ...design.colors, accent: v } })} />
          <ColorField
            label="Background"
            value={design.colors.background}
            onChange={(v) => update({ colors: { ...design.colors, background: v } })}
          />
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold text-foreground">Section headings</h3>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Style">
            <Select
              value={design.heading.style}
              onValueChange={(v) => update({ heading: { ...design.heading, style: v as ResumeDesign['heading']['style'] } })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {HEADING_STYLES.map((s) => (
                  <SelectItem key={s} value={s}>
                    {s}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field label="Capitalization">
            <Select
              value={design.heading.capitalization}
              onValueChange={(v) =>
                update({ heading: { ...design.heading, capitalization: v as ResumeDesign['heading']['capitalization'] } })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CAPITALIZATIONS.map((c) => (
                  <SelectItem key={c} value={c}>
                    {c}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold text-foreground">Spacing</h3>
        <div className="grid grid-cols-3 gap-3">
          <NumberField
            label="Section gap (px)"
            value={design.spacing.sectionGap}
            min={0}
            onChange={(v) => update({ spacing: { ...design.spacing, sectionGap: v } })}
          />
          <NumberField
            label="Entry gap (px)"
            value={design.spacing.entryGap}
            min={0}
            onChange={(v) => update({ spacing: { ...design.spacing, entryGap: v } })}
          />
          <NumberField
            label="Bullet gap (px)"
            value={design.spacing.bulletGap}
            min={0}
            onChange={(v) => update({ spacing: { ...design.spacing, bulletGap: v } })}
          />
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold text-foreground">Page margins</h3>
        <p className="text-xs text-muted-foreground">
          Page size (A4/Letter) isn&apos;t wired up to rendering yet — margins already apply live.
        </p>
        <div className="grid grid-cols-2 gap-3">
          <NumberField
            label="Top (px)"
            value={design.page.marginTop}
            min={0}
            onChange={(v) => update({ page: { ...design.page, marginTop: v } })}
          />
          <NumberField
            label="Bottom (px)"
            value={design.page.marginBottom}
            min={0}
            onChange={(v) => update({ page: { ...design.page, marginBottom: v } })}
          />
          <NumberField
            label="Left (px)"
            value={design.page.marginLeft}
            min={0}
            onChange={(v) => update({ page: { ...design.page, marginLeft: v } })}
          />
          <NumberField
            label="Right (px)"
            value={design.page.marginRight}
            min={0}
            onChange={(v) => update({ page: { ...design.page, marginRight: v } })}
          />
        </div>
      </section>
    </div>
  )
}
