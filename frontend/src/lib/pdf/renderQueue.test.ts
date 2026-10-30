import { afterEach, describe, expect, it, vi } from 'vitest'
import { createRenderQueue, SupersededRenderError } from './renderQueue'

afterEach(() => vi.useRealTimers())
describe('PDF render queue', () => {
  it('debounces typing and renders only the latest queued revision', async () => {
    vi.useFakeTimers()
    const render = vi.fn(async (n: number) => n * 2)
    const queue = createRenderQueue(render)
    const old = queue.request('a', 1).catch((e) => e)
    const latest = queue.request('b', 2)
    await vi.advanceTimersByTimeAsync(200)
    expect(await old).toBeInstanceOf(SupersededRenderError)
    expect(await latest).toBe(4)
    expect(render).toHaveBeenCalledTimes(1)
    queue.dispose()
  })
  it('serializes work, replaces pending changes, and reuses the generated download', async () => {
    vi.useFakeTimers()
    let finish!: (n: number) => void
    const render = vi.fn((n: number) => n === 1 ? new Promise<number>((r) => { finish = r }) : Promise.resolve(n))
    const queue = createRenderQueue(render)
    const first = queue.request('a', 1, true)
    await vi.advanceTimersByTimeAsync(0)
    const stale = queue.request('b', 2).catch((e) => e)
    const latest = queue.request('c', 3)
    expect(queue.request('a', 1, true)).toBe(first)
    // Requesting a different revision invalidates the pending one.
    expect(await stale).toBeInstanceOf(SupersededRenderError)
    await expect(latest).rejects.toBeInstanceOf(SupersededRenderError)
    const final = queue.request('c', 3)
    finish(1)
    await vi.advanceTimersByTimeAsync(1)
    expect(await final).toBe(3)
    expect(await queue.request('c', 3, true)).toBe(3)
    expect(render).toHaveBeenCalledTimes(2)
    queue.dispose()
  })
  it('bounds the wait during continuous typing and supports immediate download', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(1000)
    const render = vi.fn(async (n: number) => n)
    const queue = createRenderQueue(render)
    for (let n = 0; n < 8; n++) {
      void queue.request(String(n), n).catch(() => {})
      await vi.advanceTimersByTimeAsync(100)
    }
    expect(render).toHaveBeenCalledTimes(1)
    const download = queue.request('download', 9, true)
    await vi.advanceTimersByTimeAsync(0)
    expect(await download).toBe(9)
    queue.dispose()
  })
  it('recovers after errors and rejects waiters when closed', async () => {
    vi.useFakeTimers()
    const queue = createRenderQueue(async () => { throw new Error('bad font') })
    const failed = queue.request('a', 1).catch((e: Error) => e.message)
    await vi.advanceTimersByTimeAsync(200)
    expect(await failed).toBe('bad font')
    const pending = queue.request('b', 2).catch((e) => e)
    queue.dispose()
    expect(await pending).toBeInstanceOf(SupersededRenderError)
  })
})
