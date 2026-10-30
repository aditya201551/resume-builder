import catalog from '../../../backend/internal/design/templates.json'
import type { Template } from '@/types/template'
import type { ResumeDesign, SectionRef } from '@/types/design'
import { createDefaultResumeDesign } from '@/lib/defaultResumeDesign'

export const DEFAULT_TEMPLATE_ID = catalog.defaultTemplateId
export const templates: readonly Template[] = catalog.templates as Template[]

export function getTemplate(id: string): Template {
  return templates.find((template) => template.id === id) ?? templates.find((template) => template.id === DEFAULT_TEMPLATE_ID)!
}

export function supportsTemplateField(template: Template, key: string) {
  return template.supported_fields.includes(key)
}

// Read-time compatibility for saved DB UUIDs and pre-registry local drafts. Shared
// formatting survives. Dormant legacy layout settings are kept in the saved design.
export function normalizeTemplateDesign(raw: ResumeDesign): ResumeDesign {
  const template = getTemplate(raw.templateId)
  const base = createDefaultResumeDesign(template.id)
  const design: ResumeDesign = {
    ...base, ...raw, templateId: template.id,
    layout: { ...base.layout, ...raw.layout, columnWidths: { ...base.layout.columnWidths, ...raw.layout?.columnWidths } },
    sectionOrder: {
      one: { sections: raw.sectionOrder?.one?.sections ?? base.sectionOrder.one.sections },
      two: { left: raw.sectionOrder?.two?.left ?? base.sectionOrder.two.left, right: raw.sectionOrder?.two?.right ?? base.sectionOrder.two.right },
      mix: { bands: raw.sectionOrder?.mix?.bands ?? base.sectionOrder.mix.bands },
    },
    page: { ...base.page, ...raw.page },
    typography: { ...base.typography, ...raw.typography },
    colors: { ...base.colors, ...raw.colors, applyAccent: { ...base.colors.applyAccent, ...raw.colors?.applyAccent } },
    heading: { ...base.heading, ...raw.heading },
    header: { ...base.header, ...raw.header, photo: { ...base.header.photo, ...raw.header?.photo } },
    entryLayout: { ...base.entryLayout, ...raw.entryLayout },
    sectionDisplay: { ...base.sectionDisplay, ...raw.sectionDisplay },
    spacing: { ...base.spacing, ...raw.spacing },
    linkStyle: { ...base.linkStyle, ...raw.linkStyle },
    footer: { ...base.footer, ...raw.footer },
  }
  if (template.supported_modes.includes(design.layout.mode)) return design
  const order = design.sectionOrder
  const active = design.layout.mode === 'two'
    ? [...(order.two?.right ?? []), ...(order.two?.left ?? [])]
    : design.layout.mode === 'mix'
      ? (order.mix?.bands ?? []).flatMap((band) => [...(band.full ?? []), ...(band.left ?? []), ...(band.right ?? [])])
      : []
  const seen = new Set<string>()
  const sections = [...active, ...(order.one?.sections ?? [])].filter((ref: SectionRef) => {
    const key = ref.sectionType === 'custom' ? `custom-${ref.customSectionId}` : ref.sectionType
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
  const mode = base.layout.mode
  return {
    ...design, layout: { ...design.layout, mode },
    sectionOrder: {
      ...order,
      ...(mode === 'one' ? { one: { sections } }
        : mode === 'two' ? { two: { left: [], right: sections } }
          : { mix: { bands: [{ full: sections }] } }),
    },
  }
}

export function selectTemplate(design: ResumeDesign, templateId: string): ResumeDesign {
  if (!templates.some((template) => template.id === templateId)) throw new Error(`Unknown template: ${templateId}`)
  if (design.templateId === templateId) return normalizeTemplateDesign(design)
  return applyTemplateDefaults(design, templateId)
}

export function restoreTemplateDefaults(design: ResumeDesign): ResumeDesign {
  return applyTemplateDefaults(design, getTemplate(design.templateId).id)
}

function applyTemplateDefaults(design: ResumeDesign, templateId: string): ResumeDesign {
  const selected = createDefaultResumeDesign(templateId)
  const refs = structuredClone(activeSectionRefs(design))
  const key = (ref: SectionRef) => ref.sectionType === 'custom' ? `custom:${ref.customSectionId}` : ref.sectionType
  const saved = new Map(refs.map((ref) => [key(ref), ref]))
  const seen = new Set<string>()
  const apply = (defaults: SectionRef[]) => defaults.map((ref) => {
    seen.add(key(ref))
    return saved.get(key(ref)) ?? ref
  })
  const order = selected.sectionOrder
  const mode = selected.layout.mode
  if (mode === 'one') order.one.sections = apply(order.one.sections)
  else if (mode === 'two') {
    order.two.left = apply(order.two.left)
    order.two.right = apply(order.two.right)
  } else {
    order.mix.bands = order.mix.bands.map((band) => ({
      full: apply(band.full ?? []), left: apply(band.left ?? []), right: apply(band.right ?? []),
    }))
  }
  const remaining = refs.filter((ref) => {
    if (seen.has(key(ref))) return false
    seen.add(key(ref))
    return true
  })
  if (mode === 'one') order.one.sections.push(...remaining)
  else if (mode === 'two') order.two.right.push(...remaining)
  else if (remaining.length) order.mix.bands.push({ full: remaining })
  return selected
}

function activeSectionRefs(design: ResumeDesign): SectionRef[] {
  if (design.layout.mode === 'two') return [...design.sectionOrder.two.right, ...design.sectionOrder.two.left]
  if (design.layout.mode === 'mix') return design.sectionOrder.mix.bands.flatMap((band) => [...(band.full ?? []), ...(band.left ?? []), ...(band.right ?? [])])
  return design.sectionOrder.one.sections
}
