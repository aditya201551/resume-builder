import { describe, expect, it, vi, afterEach } from 'vitest'
import { defaultResumeDesign } from '@/lib/defaultResumeDesign'
import { DEFAULT_TEMPLATE_ID, getTemplate, normalizeTemplateDesign, selectTemplate, supportsTemplateField, templates } from './catalog'
import { pdfTemplateRegistry } from './pdfRegistry'
import { loadDraft } from '@/lib/resumeDraftStorage'
import type { FullResume } from '@/types/resume'

afterEach(() => vi.unstubAllGlobals())
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
  it('selects a template without replacing shared formatting or sections', () => {
    const original = structuredClone(defaultResumeDesign)
    original.typography.fontFamily = 'Roboto'
    original.colors.accent = '#112233'
    original.spacing.sectionGap = 25
    original.sectionOrder.one.sections = [{ sectionType: 'skills', customSectionId: null, isVisible: false, titleOverride: 'Technical skills' }]
    const selected = selectTemplate(original, DEFAULT_TEMPLATE_ID)
    expect(selected).toEqual(original)
    expect(() => selectTemplate(original, 'not-installed')).toThrow('Unknown template')
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
