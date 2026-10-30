export class SupersededRenderError extends Error {
  constructor() { super('A newer resume revision replaced this render') }
}

/** One running job plus one latest pending job. Keeps typing from building a render backlog. */
export function createRenderQueue<T, R>(render: (input: T) => Promise<R>, delay = 200) {
  type Job = { key: string; input: T; promise: Promise<R>; resolve: (result: R) => void; reject: (error: unknown) => void }
  let active: Job | undefined
  let pending: Job | undefined
  let timer: ReturnType<typeof setTimeout> | undefined
  let firstPendingAt = 0
  let disposed = false
  const cache = new Map<string, R>()

  function arm(immediate: boolean) {
    clearTimeout(timer)
    if (active || !pending || disposed) return
    const remaining = Math.max(0, 800 - (Date.now() - firstPendingAt))
    timer = setTimeout(run, immediate ? 0 : Math.min(delay, remaining))
  }
  async function run() {
    if (active || !pending || disposed) return
    const job = pending
    active = job
    pending = undefined
    firstPendingAt = 0
    try {
      const result = await render(job.input)
      if (!disposed) {
        cache.set(job.key, result)
        if (cache.size > 2) cache.delete(cache.keys().next().value!)
        job.resolve(result)
      }
    } catch (error) {
      job.reject(error)
    } finally {
      active = undefined
      arm(true)
    }
  }
  return {
    request(key: string, input: T, immediate = false): Promise<R> {
      if (disposed) return Promise.reject(new Error('PDF workspace closed'))
      if (pending && pending.key !== key) {
        pending.reject(new SupersededRenderError())
        pending = undefined
      }
      if (cache.has(key)) return Promise.resolve(cache.get(key)!)
      if (active?.key === key) return active.promise
      if (pending?.key === key) { if (immediate) arm(true); return pending.promise }
      let resolve!: Job['resolve']
      let reject!: Job['reject']
      const promise = new Promise<R>((yes, no) => { resolve = yes; reject = no })
      pending = { key, input, promise, resolve, reject }
      if (!firstPendingAt) firstPendingAt = Date.now()
      arm(immediate)
      return promise
    },
    dispose() {
      disposed = true
      clearTimeout(timer)
      pending?.reject(new SupersededRenderError())
      active?.reject(new SupersededRenderError())
      pending = undefined
      cache.clear()
    },
  }
}
