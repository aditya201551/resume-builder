import { afterEach, describe, expect, it, vi } from 'vitest'
import { generateResumePdf } from './session'
import { defaultResumeDesign } from '@/lib/defaultResumeDesign'
import type { FullResume } from '@/types/resume'
import type { PdfWorkerRequest, PdfWorkerResponse } from './types'

const data: FullResume = {
  resume: { id: 'r1', user_id: 'u1', label: 'Test', full_name: 'Alex', headline: null, email: null, phone: null, location: null, photo_url: null, summary: null, links: [], created_at: '', updated_at: '', last_exported_at: null },
  design: defaultResumeDesign, work_experiences: [], educations: [], skill_groups: [], projects: [], certifications: [], languages: [], misc_entries: [], custom_sections: [], section_configs: [],
}

afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals() })

describe('on-demand library PDF export', () => {
  function worker(fail: boolean) {
    const terminate = vi.fn()
    class FakeWorker {
      onmessage?: (event: { data: PdfWorkerResponse }) => void
      terminate = terminate
      postMessage(request: PdfWorkerRequest) {
        this.onmessage?.({ data: fail
          ? { id: request.id, error: 'Invalid font' }
          : { id: request.id, result: { bytes: new Uint8Array([37, 80, 68, 70]), pages: [], regions: [], warnings: [], durationMs: 1 } } })
      }
    }
    const constructor = vi.fn(function () { return new FakeWorker() })
    vi.stubGlobal('Worker', constructor)
    return { constructor, terminate }
  }

  it('starts only on request and releases the worker after producing a PDF', async () => {
    vi.useFakeTimers()
    const mocked = worker(false)
    expect(mocked.constructor).not.toHaveBeenCalled()
    const pending = generateResumePdf(data)
    await vi.advanceTimersByTimeAsync(0)
    const blob = await pending
    expect(blob.type).toBe('application/pdf')
    expect(await blob.text()).toBe('%PDF')
    expect(mocked.constructor).toHaveBeenCalledTimes(1)
    expect(mocked.terminate).toHaveBeenCalledTimes(1)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('releases the worker and rejects failed exports instead of downloading a broken file', async () => {
    vi.useFakeTimers()
    const mocked = worker(true)
    const pending = generateResumePdf(data).catch((error: Error) => error.message)
    await vi.advanceTimersByTimeAsync(0)
    expect(await pending).toBe('Invalid font')
    expect(mocked.terminate).toHaveBeenCalledTimes(1)
    expect(vi.getTimerCount()).toBe(0)
  })
})
