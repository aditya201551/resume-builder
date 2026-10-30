import { afterEach, describe, expect, it, vi } from 'vitest'
import { importResume, MAX_RESUME_PDF_BYTES, resumeFileError } from './importResume'

afterEach(() => vi.unstubAllGlobals())

describe('PDF resume import', () => {
  it('rejects unsupported, empty and oversized files before upload', () => {
    expect(resumeFileError(new File(['resume'], 'resume.docx'))).toMatch(/PDF file/)
    expect(resumeFileError(new File([], 'resume.pdf'))).toMatch(/empty/)
    expect(resumeFileError({ name: 'resume.pdf', size: MAX_RESUME_PDF_BYTES + 1 } as File)).toMatch(/10 MB/)
    expect(resumeFileError(new File(['%PDF'], 'RESUME.PDF'))).toBeNull()
  })

  it('uploads multipart to the standalone import endpoint without chat calls', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ resume: { id: 'imported' }, warnings: [] }) })
    vi.stubGlobal('fetch', fetcher)
    const file = new File(['%PDF-1.7'], 'resume.pdf', { type: 'application/pdf' })
    const result = await importResume(file)
    expect(result.resume.id).toBe('imported')
    const [url, options] = fetcher.mock.calls[0]
    expect(url).toBe('/api/resumes/import')
    expect(options.credentials).toBe('include')
    expect(options.body.get('file').name).toBe('resume.pdf')
    expect(options.headers).toBeUndefined() // Browser supplies the multipart boundary.
    expect(fetcher).toHaveBeenCalledTimes(1)
  })

  it('shows a non-resume rejection without returning a created resume', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 422, json: async () => ({ code: 'not_resume', message: "This PDF doesn't appear to be a resume or CV." }) }))
    await expect(importResume(new File(['%PDF'], 'invoice.pdf'))).rejects.toThrow("doesn't appear to be a resume")
  })
})
