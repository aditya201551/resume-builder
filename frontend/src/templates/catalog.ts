import catalog from '../../../backend/internal/design/templates.json'
import type { Template } from '@/types/template'
import type { ResumeDesign, SectionRef } from '@/types/design'
import { defaultResumeDesign } from '@/lib/defaultResumeDesign'

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
  const base = defaultResumeDesign
  const template = getTemplate(raw.templateId)
  const design: ResumeDesign = {
    ...base, ...raw, templateId: template.id,
    layout: { ...base.layout, ...raw.layout },
    sectionOrder: {
      one: { sections: raw.sectionOrder?.one?.sections ?? [] },
      two: { left: raw.sectionOrder?.two?.left ?? [], right: raw.sectionOrder?.two?.right ?? [] },
      mix: { bands: raw.sectionOrder?.mix?.bands ?? [] },
    },
    page: { ...base.page, ...raw.page },
    typography: { ...base.typography, ...raw.typography },
    colors: { ...base.colors, ...raw.colors, applyAccent: { ...base.colors.applyAccent, ...raw.colors?.applyAccent } },
    heading: { ...base.heading, ...raw.heading },
    header: { ...base.header, ...raw.header },
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
  return { ...design, layout: { ...design.layout, mode: template.supported_modes[0] as ResumeDesign['layout']['mode'] }, sectionOrder: { ...order, one: { sections } } }
}

export function selectTemplate(design: ResumeDesign, templateId: string): ResumeDesign {
  if (!templates.some((template) => template.id === templateId)) throw new Error(`Unknown template: ${templateId}`)
  // Only template identity/layout compatibility changes. Never apply a formatting preset.
  return normalizeTemplateDesign({ ...design, templateId })
}
