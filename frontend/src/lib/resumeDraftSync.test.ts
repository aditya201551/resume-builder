import { afterEach, describe, expect, it, vi } from 'vitest'
import { draftReducer, type DraftAction } from '@/hooks/resumeDraftReducer'
import { defaultResumeDesign } from '@/lib/defaultResumeDesign'
import type { FullResume } from '@/types/resume'
import { flushDraft, isDraftDirty } from './resumeDraftSync'
import { createDraftAutosync } from './resumeDraftAutosync'
import { createEntryIdentity, resolveEntryReferences } from './resumeDraftIds'

function emptyResume(): FullResume {
  return {
    resume: {
      id: 'r1', user_id: 'u1', label: 'Resume', full_name: '', headline: null,
      email: null, phone: null, location: null, photo_url: null, summary: null,
      links: [], created_at: '', updated_at: '', last_exported_at: null,
    },
    design: structuredClone(defaultResumeDesign), work_experiences: [], educations: [],
    skill_groups: [], projects: [], certifications: [], languages: [], misc_entries: [],
    custom_sections: [], section_configs: [],
  }
}

function harness() {
  const data = emptyResume()
  let state = draftReducer(null!, { type: 'replace_all', data })
  const dispatch = (action: DraftAction) => { state = draftReducer(state, action) }
  return { getState: () => state, dispatch, flush: () => flushDraft('r1', state, dispatch) }
}

function deferred() {
  let resolve!: () => void
  const promise = new Promise<void>((done) => { resolve = done })
  return { promise, resolve }
}

function mockServer(handler?: (url: string, method: string, body: Record<string, unknown>) => Promise<Response> | Response) {
  let nextId = 0
  const fetch = vi.fn(async (url: string, init: RequestInit) => {
    const body = init.body ? JSON.parse(String(init.body)) : {}
    if (handler) return handler(url, init.method!, body)
    return Response.json({ ...body, id: `server-${++nextId}` })
  })
  vi.stubGlobal('fetch', fetch)
  return fetch
}

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

it('restores formatting defaults locally without changing resume content or the synced snapshot', () => {
  const h = harness()
  h.dispatch({ type: 'update_meta', patch: { full_name: 'My name', summary: 'Unsynced summary' } })
  h.dispatch({ type: 'design_update', patch: { typography: { ...defaultResumeDesign.typography, fontFamily: 'Roboto' } } })
  const before = h.getState()
  h.dispatch({ type: 'restore_design_defaults' })
  const after = h.getState()
  expect(after.data.design.typography).toEqual(defaultResumeDesign.typography)
  expect(after.data.resume).toBe(before.data.resume)
  expect(after.data.work_experiences).toBe(before.data.work_experiences)
  expect(after.data.custom_sections).toBe(before.data.custom_sections)
  expect(after.lastSynced).toBe(before.lastSynced)
  expect(isDraftDirty(after.data, after.lastSynced)).toBe(true)
})

describe('automatic draft syncing', () => {
  it('debounces edits, with a bounded wait during continuous typing', async () => {
    vi.useFakeTimers()
    const h = harness()
    const fetch = mockServer()
    const status = vi.fn()
    const sync = createDraftAutosync({ ...h, sync: h.flush, onStatus: status, onSynced: vi.fn() })
    for (let i = 0; i < 5; i++) {
      h.dispatch({ type: 'update_meta', patch: { full_name: `Name ${i}` } })
      sync.schedule()
      await vi.advanceTimersByTimeAsync(200)
    }
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(JSON.parse(fetch.mock.calls[0][1].body as string).full_name).toBe('Name 4')
    expect(isDraftDirty(h.getState().data, h.getState().lastSynced)).toBe(false)
    expect(status).toHaveBeenLastCalledWith('idle')
    sync.dispose()
  })

  it('serializes edits during a request and makes concurrent flush callers await the newest changes', async () => {
    const gate = deferred()
    const started = deferred()
    const names: unknown[] = []
    mockServer(async (_url, _method, body) => {
      names.push(body.full_name)
      if (names.length === 1) { started.resolve(); await gate.promise }
      return Response.json({})
    })
    const h = harness()
    h.dispatch({ type: 'update_meta', patch: { full_name: 'First' } })
    const sync = createDraftAutosync({ ...h, sync: h.flush, onStatus: vi.fn(), onSynced: vi.fn() })
    const first = sync.flush()
    await started.promise
    h.dispatch({ type: 'update_meta', patch: { full_name: 'Latest' } })
    sync.schedule()
    const second = sync.flush()
    expect(first).toBe(second)
    expect(names).toEqual(['First'])
    gate.resolve()
    expect(await second).toBe(true)
    expect(names).toEqual(['First', 'Latest'])
    expect(h.getState().lastSynced.resume.full_name).toBe('Latest')
    sync.dispose()
  })

  it('retries failures automatically without another edit or save event', async () => {
    vi.useFakeTimers()
    vi.spyOn(console, 'error').mockImplementation(() => {})
    let attempts = 0
    mockServer(() => ++attempts === 1 ? new Response('offline', { status: 503 }) : Response.json({}))
    const h = harness()
    h.dispatch({ type: 'update_meta', patch: { full_name: 'Retry me' } })
    const onStatus = vi.fn()
    const sync = createDraftAutosync({ ...h, sync: h.flush, onStatus, onSynced: vi.fn() })
    expect(await sync.flush()).toBe(false)
    expect(onStatus).toHaveBeenLastCalledWith('error')
    expect(isDraftDirty(h.getState().data, h.getState().lastSynced)).toBe(true)
    await vi.advanceTimersByTimeAsync(2000)
    expect(attempts).toBe(2)
    expect(onStatus).toHaveBeenLastCalledWith('idle')
    expect(isDraftDirty(h.getState().data, h.getState().lastSynced)).toBe(false)
    sync.dispose()
  })

  it('resumes recovered dirty drafts and cancels queued requests when disposed', async () => {
    vi.useFakeTimers()
    const h = harness()
    h.dispatch({ type: 'update_meta', patch: { summary: 'Recovered edit' } })
    const fetch = mockServer()
    const sync = createDraftAutosync({ ...h, sync: h.flush, onStatus: vi.fn(), onSynced: vi.fn() })
    sync.schedule()
    await vi.advanceTimersByTimeAsync(300)
    expect(fetch).toHaveBeenCalledTimes(1)
    h.dispatch({ type: 'update_meta', patch: { summary: 'Next edit' } })
    sync.schedule()
    sync.dispose()
    await vi.advanceTimersByTimeAsync(30000)
    expect(fetch).toHaveBeenCalledTimes(1)
  })
})

describe('sync acknowledgments', () => {
  it('keeps newer field edits dirty when an older PATCH completes', async () => {
    const gate = deferred()
    const started = deferred()
    mockServer(async () => { started.resolve(); await gate.promise; return Response.json({}) })
    const h = harness()
    h.dispatch({ type: 'update_meta', patch: { full_name: 'Submitted' } })
    const flushing = h.flush()
    await started.promise
    h.dispatch({ type: 'update_meta', patch: { full_name: 'Still typing' } })
    gate.resolve()
    await flushing
    expect(h.getState().data.resume.full_name).toBe('Still typing')
    expect(h.getState().lastSynced.resume.full_name).toBe('Submitted')
  })

  it('does not recreate an entry when a later create in the same collection fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    let failed = false
    const fetch = mockServer((_url, _method, body) => {
      if (body.name === 'Second' && !failed) { failed = true; return new Response('retry', { status: 503 }) }
      return Response.json({ ...body, id: body.name === 'First' ? 'first' : 'second' })
    })
    const h = harness()
    h.dispatch({ type: 'flat_create', entity: 'languages', tempId: 'temp-first', fields: { name: 'First', sort_order: 0 } })
    h.dispatch({ type: 'flat_create', entity: 'languages', tempId: 'temp-second', fields: { name: 'Second', sort_order: 1 } })
    await expect(h.flush()).rejects.toThrow()
    expect(h.getState().lastSynced.languages.map((item) => item.id)).toEqual(['first'])
    await h.flush()
    expect(fetch.mock.calls.filter(([, init]) => JSON.parse(init.body as string).name === 'First')).toHaveLength(1)
    expect(isDraftDirty(h.getState().data, h.getState().lastSynced)).toBe(false)
  })

  it('creates nested items using resolved parent IDs and acknowledges a reordered list', async () => {
    const fetch = mockServer()
    const h = harness()
    h.dispatch({ type: 'skill_group_create', tempId: 'temp-group', fields: { group_name: 'Skills', sort_order: 0 } })
    h.dispatch({ type: 'skill_item_create', groupId: 'temp-group', tempId: 'temp-item', fields: { name: 'Go', sort_order: 0 } })
    h.dispatch({ type: 'custom_section_create', tempId: 'temp-section', fields: { title: 'Extras', sort_order: 0 } })
    h.dispatch({ type: 'custom_entry_create', sectionId: 'temp-section', tempId: 'temp-entry', fields: { title: 'Example', sort_order: 0 } })
    await h.flush()
    expect(fetch.mock.calls.map(([url]) => url)).toEqual([
      '/api/resumes/r1/skill-groups', '/api/resumes/r1/skill-groups/server-1/items',
      '/api/resumes/r1/custom-sections', '/api/resumes/r1/custom-sections/server-3/entries',
    ])
    expect(h.getState().data.skill_groups[0].items[0].skill_group_id).toBe('server-1')
    expect(h.getState().data.custom_sections[0].entries[0].custom_section_id).toBe('server-3')
    expect(isDraftDirty(h.getState().data, h.getState().lastSynced)).toBe(false)
    h.dispatch({ type: 'flat_create', entity: 'languages', tempId: 'temp-a', fields: { name: 'A', sort_order: 0 } })
    h.dispatch({ type: 'flat_create', entity: 'languages', tempId: 'temp-b', fields: { name: 'B', sort_order: 1 } })
    await h.flush()
    h.dispatch({ type: 'flat_reorder', entity: 'languages', orderedIds: ['server-6', 'server-5'] })
    await h.flush()
    expect(h.getState().lastSynced.languages.map((item) => item.name)).toEqual(['B', 'A'])
    expect(isDraftDirty(h.getState().data, h.getState().lastSynced)).toBe(false)
  })

  it('deletes a row removed locally while its create was in flight', async () => {
    const gate = deferred()
    const started = deferred()
    const fetch = mockServer(async (_url, method) => {
      if (method === 'POST') { started.resolve(); await gate.promise; return Response.json({ id: 'created' }) }
      return new Response(null, { status: 204 })
    })
    const h = harness()
    h.dispatch({ type: 'flat_create', entity: 'languages', tempId: 'temp-language', fields: { name: 'Go', sort_order: 0 } })
    const flushing = h.flush()
    await started.promise
    h.dispatch({ type: 'flat_delete', entity: 'languages', id: 'temp-language' })
    gate.resolve()
    await flushing
    await h.flush()
    expect(fetch.mock.calls[1][0]).toBe('/api/resumes/r1/languages/created')
    expect(fetch.mock.calls[1][1].method).toBe('DELETE')
    expect(isDraftDirty(h.getState().data, h.getState().lastSynced)).toBe(false)
  })

  it('patches newer edits after an idempotent create returns previously stored content', async () => {
    const fetch = mockServer((_url, method, body) => Response.json(method === 'POST'
      ? { ...body, id: 'created', name: 'Previously created' } : {}))
    const h = harness()
    h.dispatch({ type: 'flat_create', entity: 'languages', tempId: 'temp-language', fields: { name: 'Latest', sort_order: 0 } })
    await h.flush()
    expect(h.getState().lastSynced.languages[0].name).toBe('Previously created')
    await h.flush()
    expect(fetch.mock.calls[1][1].method).toBe('PATCH')
    expect(JSON.parse(fetch.mock.calls[1][1].body as string).name).toBe('Latest')
    expect(isDraftDirty(h.getState().data, h.getState().lastSynced)).toBe(false)
  })
})

describe('entry identities', () => {
  it('keeps React keys and open panel identities stable while resolving late events and proposals', () => {
    const identity = createEntryIdentity()
    identity.register({ 'temp-group': 'group', 'temp-item': 'item' })
    expect(identity.getKey('group')).toBe('temp-group')
    expect(identity.getKey('temp-group')).toBe('temp-group')
    expect(resolveEntryReferences({ type: 'skill_item_update', groupId: 'temp-group', id: 'temp-item' }, identity.resolveId))
      .toEqual({ type: 'skill_item_update', groupId: 'group', id: 'item' })
    expect(resolveEntryReferences({ orderedIds: ['temp-item'], patch: { name: 'temp-item' } }, identity.resolveId))
      .toEqual({ orderedIds: ['item'], patch: { name: 'temp-item' } })
  })
})
