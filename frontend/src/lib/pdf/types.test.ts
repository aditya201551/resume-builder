import { describe, expect, it } from 'vitest'
import { pdfSourceKey, type PdfSource } from './types'
import { defaultResumeDesign } from '@/lib/defaultResumeDesign'
import type { FullResume } from '@/types/resume'

describe('PDF revision identity', () => {
  it('ignores resume labels and synchronization metadata but tracks rendered edits', () => {
    const data: FullResume = {
      resume: { id: 'r1', user_id: 'u1', label: 'My resume', full_name: 'Alex', headline: null, email: null, phone: null, location: null, photo_url: null, summary: 'Original', links: [], created_at: '', updated_at: '', last_exported_at: null },
      design: structuredClone(defaultResumeDesign), work_experiences: [], educations: [], skill_groups: [], projects: [], certifications: [], languages: [], misc_entries: [], custom_sections: [], section_configs: [],
    }
    const source: PdfSource = { data, rendererKey: 'classic' }
    const original = pdfSourceKey(source)
    data.resume.label = 'Renamed document'
    data.resume.updated_at = '2026-10-30T10:00:00Z'
    data.resume.last_exported_at = '2026-10-30T11:00:00Z'
    expect(pdfSourceKey(source)).toBe(original)
    data.resume.summary = 'Changed content'
    expect(pdfSourceKey(source)).not.toBe(original)
  })
})
