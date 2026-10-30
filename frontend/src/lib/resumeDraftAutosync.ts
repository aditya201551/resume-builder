import type { DraftState } from '@/hooks/resumeDraftReducer'
import { isDraftDirty } from '@/lib/resumeDraftSync'

export type SyncStatus = 'idle' | 'dirty' | 'syncing' | 'error'

export function createDraftAutosync(options: {
  getState: () => DraftState | null
  sync: (snapshot: DraftState) => Promise<void>
  onStatus: (status: SyncStatus) => void
  onSynced: () => void
}) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let inFlight: Promise<boolean> | undefined
  let disposed = false
  let firstChangeAt: number | undefined
  let retryDelay = 2000
  const dirty = () => {
    const state = options.getState()
    return state !== null && isDraftDirty(state.data, state.lastSynced)
  }
  function clearTimer() {
    clearTimeout(timer)
    timer = undefined
  }
  function flush(): Promise<boolean> {
    clearTimer()
    firstChangeAt = undefined
    if (inFlight) return inFlight
    if (disposed) return Promise.resolve(false)
    if (!dirty()) {
      options.onStatus('idle')
      return Promise.resolve(true)
    }

    options.onStatus('syncing')
    inFlight = Promise.resolve().then(async () => {
      try {
        // Changes made during a request stay dirty and are sent by the next serialized pass.
        while (!disposed && dirty()) await options.sync(options.getState()!)
        retryDelay = 2000
        if (!disposed) {
          options.onStatus('idle')
          options.onSynced()
        }
        return !dirty()
      } catch {
        if (!disposed) {
          options.onStatus('error')
          timer = setTimeout(() => void flush(), retryDelay)
          retryDelay = Math.min(retryDelay * 2, 30000)
        }
        return false
      } finally {
        inFlight = undefined
      }
    })
    return inFlight
  }
  return {
    flush,
    schedule() {
      if (disposed || inFlight) return
      if (!dirty()) {
        clearTimer()
        firstChangeAt = undefined
        options.onStatus('idle')
        return
      }
      options.onStatus('dirty')
      firstChangeAt ??= Date.now()
      clearTimer()
      // Debounce typing, but sync at least once a second during continuous edits.
      timer = setTimeout(() => void flush(), Math.min(300, Math.max(0, 1000 - (Date.now() - firstChangeAt))))
    },
    dispose() {
      disposed = true
      clearTimer()
    },
  }
}
