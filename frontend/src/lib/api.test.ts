import { afterEach, describe, expect, it, vi } from 'vitest'
import { renameResume } from './api'

afterEach(() => vi.unstubAllGlobals())

describe('rename resume', () => {
  it('preserves all current metadata when the replacement endpoint changes the label', async () => {
    const metadata = { full_name: 'Alex', headline: 'Designer', email: 'alex@example.com', phone: '123', location: 'London', photo_url: null, summary: 'Summary', links: [{ label: 'Portfolio', url: 'https://example.com' }] }
    const fetch = vi.fn()
      .mockResolvedValueOnce(Response.json({ id: 'r1', label: 'Old', ...metadata }))
      .mockResolvedValueOnce(Response.json({ id: 'r1', label: 'New', ...metadata }))
    vi.stubGlobal('fetch', fetch)
    await expect(renameResume('r1', 'New')).resolves.toMatchObject({ label: 'New' })
    expect(fetch.mock.calls[0][0]).toBe('/api/resumes/r1')
    expect(fetch.mock.calls[1][1].method).toBe('PATCH')
    expect(JSON.parse(fetch.mock.calls[1][1].body)).toEqual({ ...metadata, label: 'New' })
  })

  it('does not overwrite metadata when the fresh read fails', async () => {
    const fetch = vi.fn().mockResolvedValue(new Response('Unavailable', { status: 503 }))
    vi.stubGlobal('fetch', fetch)
    await expect(renameResume('r1', 'New')).rejects.toThrow('503')
    expect(fetch).toHaveBeenCalledTimes(1)
  })
})
