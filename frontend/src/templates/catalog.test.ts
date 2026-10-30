import { describe, expect, it, vi, afterEach } from 'vitest'
import { createDefaultResumeDesign, defaultResumeDesign } from '@/lib/defaultResumeDesign'
import { DEFAULT_TEMPLATE_ID, getTemplate, normalizeTemplateDesign, restoreTemplateDefaults, selectTemplate, supportsTemplateField, templates } from './catalog'
import { pdfTemplateRegistry } from './pdfRegistry'
import { loadDraft } from '@/lib/resumeDraftStorage'
import type { FullResume } from '@/types/resume'
import type { ResumeDesign } from '@/types/design'
import type { Template } from '@/types/template'

const addedTemplates: Template[] = []
afterEach(() => {
  vi.unstubAllGlobals()
  for (const template of addedTemplates.splice(0)) {
    const index = templates.indexOf(template)
    if (index >= 0) (templates as Template[]).splice(index, 1)
  }
})

function addTemplate(mode: ResumeDesign['layout']['mode'] = 'one') {
  const template = structuredClone(getTemplate(DEFAULT_TEMPLATE_ID))
  template.id = `test-${mode}`
  template.supported_modes = [mode]
  template.defaults.layout.mode = mode
  template.defaults.typography.fontFamily = 'Arial'
  template.defaults.page.marginLeft = 24
  template.defaults.spacing.sectionGap = 24
  template.defaults.colors.applyAccent.name = true
  template.defaults.header.photo.size = 'l'
  const refs = [
    { sectionType: 'skills', customSectionId: null, isVisible: true, titleOverride: null },
    { sectionType: 'summary', customSectionId: null, isVisible: true, titleOverride: null },
  ]
  template.defaults.sectionOrder.one.sections = refs
  template.defaults.sectionOrder.two = { left: refs.slice(0, 1), right: refs.slice(1) }
  template.defaults.sectionOrder.mix.bands = [{ full: refs.slice(1), left: refs.slice(0, 1) }]
  const mutableTemplates = templates as Template[]
  mutableTemplates.push(template)
  addedTemplates.push(template)
  return template
}
describe('template module', () => {
  it('publishes one vertical renderer with explicit capabilities and a cache revision', () => {
    expect(templates.map((t) => t.id)).toEqual(['vertical'])
    for (const template of templates) {
      expect(pdfTemplateRegistry[template.renderer_key]).toBeDefined()
      expect(template.revision).toBeGreaterThan(0)
      expect(template.capabilities.columns).toBe(1)
      expect(template.supported_modes).toEqual(['one'])
      expect(supportsTemplateField(template, 'typography.fontFamily')).toBe(true)
      expect(supportsTemplateField(template, 'layout.mode')).toBe(false)
    }
  })
  it('reselects the current template without replacing user formatting or sections', () => {
    const original = structuredClone(defaultResumeDesign)
    original.typography.fontFamily = 'Roboto'
    original.colors.accent = '#112233'
    original.spacing.sectionGap = 25
    original.sectionOrder.one.sections = [{ sectionType: 'skills', customSectionId: null, isVisible: false, titleOverride: 'Technical skills' }]
    const selected = selectTemplate(original, DEFAULT_TEMPLATE_ID)
    expect(selected).toEqual(original)
    expect(() => selectTemplate(original, 'not-installed')).toThrow('Unknown template')
  })
  it('creates independent defaults for the selected template', () => {
    const template = addTemplate()
    const design = createDefaultResumeDesign(template.id)
    expect(design.typography.fontFamily).toBe('Arial')
    expect(design.page.marginLeft).toBe(24)
    design.sectionOrder.one.sections[0].titleOverride = 'Changed'
    design.colors.applyAccent.name = false
    expect(createDefaultResumeDesign(template.id).sectionOrder.one.sections[0].titleOverride).toBeNull()
    expect(createDefaultResumeDesign(template.id).colors.applyAccent.name).toBe(true)
    expect(createDefaultResumeDesign('retired-id').templateId).toBe(DEFAULT_TEMPLATE_ID)
  })
  it('fills missing saved fields from the selected template, keeping explicit zero, false and empty values', () => {
    const template = addTemplate()
    const saved = {
      templateId: template.id,
      page: { marginLeft: 0 },
      typography: { baseFontSizePt: 16 },
      colors: { applyAccent: { name: false } },
      header: { photo: { show: false } },
      sectionOrder: { one: { sections: [] } },
    } as unknown as ResumeDesign
    const normalized = normalizeTemplateDesign(saved)
    expect(normalized.typography.fontFamily).toBe('Arial')
    expect(normalized.typography.baseFontSizePt).toBe(16)
    expect(normalized.page.marginLeft).toBe(0)
    expect(normalized.spacing.sectionGap).toBe(24)
    expect(normalized.colors.applyAccent.name).toBe(false)
    expect(normalized.header.photo).toEqual({ show: false, size: 'l' })
    expect(normalized.sectionOrder.one.sections).toEqual([])
  })
  it('explicitly restores the current template defaults while keeping section preferences', () => {
    const template = addTemplate()
    const current = createDefaultResumeDesign(template.id)
    current.typography.fontFamily = 'Roboto'
    current.page.marginLeft = 0
    current.spacing.sectionGap = 7
    current.sectionOrder.one.sections = [
      { sectionType: 'skills', customSectionId: null, isVisible: false, titleOverride: 'Toolkit' },
      { sectionType: 'custom', customSectionId: 'c1', isVisible: true, titleOverride: 'Selected work' },
    ]
    const before = structuredClone(current)
    const restored = restoreTemplateDefaults(current)
    expect(restored.templateId).toBe(template.id)
    expect(restored.typography).toEqual(template.defaults.typography)
    expect(restored.page).toEqual(template.defaults.page)
    expect(restored.spacing).toEqual(template.defaults.spacing)
    expect(restored.sectionOrder.one.sections).toContainEqual(current.sectionOrder.one.sections[0])
    expect(restored.sectionOrder.one.sections).toContainEqual(current.sectionOrder.one.sections[1])
    expect(current).toEqual(before)
    expect(restoreTemplateDefaults(restored)).toEqual(restored)
  })
  it.each(['one', 'two', 'mix'] as const)('applies template defaults and retains section preferences in %s layout', (mode) => {
    const template = addTemplate(mode)
    const original = structuredClone(defaultResumeDesign)
    original.typography.fontFamily = 'Roboto'
    original.sectionOrder.one.sections = [
      { sectionType: 'skills', customSectionId: null, isVisible: false, titleOverride: 'Toolkit' },
      { sectionType: 'custom', customSectionId: 'custom-one', isVisible: true, titleOverride: 'Selected work' },
    ]
    const before = structuredClone(original)
    const selected = selectTemplate(original, template.id)
    expect(selected.layout.mode).toBe(mode)
    expect(selected.typography).toEqual(template.defaults.typography)
    expect(selected.page).toEqual(template.defaults.page)
    const order = selected.sectionOrder
    const refs = mode === 'one' ? order.one.sections : mode === 'two' ? [...order.two.left, ...order.two.right]
      : order.mix.bands.flatMap((band) => [...(band.full ?? []), ...(band.left ?? []), ...(band.right ?? [])])
    expect(refs).toHaveLength(3)
    expect(refs).toContainEqual(original.sectionOrder.one.sections[0])
    expect(refs).toContainEqual(original.sectionOrder.one.sections[1])
    expect(original).toEqual(before)
    selected.typography.fontFamily = 'Inter'
    expect(selectTemplate(selected, template.id).typography.fontFamily).toBe('Inter')
  })
  it('converts retired split drafts without losing hidden sections, custom sections, or common styling', () => {
    const original = structuredClone(defaultResumeDesign)
    original.templateId = 'old-database-uuid'
    original.layout.mode = 'two'
    original.typography.fontFamily = 'Inter'
    original.sectionOrder.two.right = [
      { sectionType: 'work_experience', customSectionId: null, isVisible: true, titleOverride: 'Career' },
      { sectionType: 'custom', customSectionId: 'c1', isVisible: true, titleOverride: 'Selected work' },
    ]
    original.sectionOrder.two.left = [{ sectionType: 'skills', customSectionId: null, isVisible: false, titleOverride: 'Toolkit' }]
    original.sectionOrder.one.sections = [{ sectionType: 'skills', customSectionId: null, isVisible: true, titleOverride: null }]
    const normalized = normalizeTemplateDesign(original)
    expect(normalized.templateId).toBe('vertical')
    expect(normalized.layout.mode).toBe('one')
    expect(normalized.sectionOrder.one.sections.map((s) => s.sectionType)).toEqual(['work_experience', 'custom', 'skills'])
    expect(normalized.sectionOrder.one.sections[2]).toEqual(original.sectionOrder.two.left[0])
    expect(normalized.typography).toEqual(original.typography)
    expect(original.layout.mode).toBe('two')
    expect(normalizeTemplateDesign(normalized)).toEqual(normalized)
    expect(getTemplate('old-database-uuid').id).toBe('vertical')
  })
  it('recovers unsynced edits from v2 local drafts rather than invalidating their storage', () => {
    const design = structuredClone(defaultResumeDesign)
    design.templateId = 'legacy-id'
    design.layout.mode = 'two'
    const synced = { design, resume: { summary: 'Server summary' } } as FullResume
    const data = { ...synced, resume: { ...synced.resume, summary: 'Unsynced local summary' }, design: { ...design, colors: { ...design.colors, accent: '#123456' } } }
    vi.stubGlobal('localStorage', { getItem: () => JSON.stringify({ version: 2, data, lastSynced: synced }) })
    const cached = loadDraft('r1')!
    expect(cached.data.resume.summary).toBe('Unsynced local summary')
    expect(cached.data.design.colors.accent).toBe('#123456')
    expect(cached.lastSynced.resume.summary).toBe('Server summary')
    expect(cached.data.design.templateId).toBe('vertical')
    expect(cached.data.design.layout.mode).toBe('one')
  })
})
