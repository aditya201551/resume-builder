import { useEffect, useRef, useState } from 'react'

export type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'

/**
 * Debounces `value` changes into a save call. Skips the mount-time value
 * (it's already persisted) and no-ops when the value round-trips back to
 * whatever was last saved, so typing-then-undoing doesn't fire a request.
 *
 * `save` now writes to the local draft (instant, in-memory) rather than the
 * network — the network write happens later, batched, via the draft's own
 * periodic flush. This debounce just bundles rapid keystrokes into fewer
 * dispatches/re-renders, so the delay is short relative to the old
 * network-latency-driven default.
 */
export function useAutosave<T>(value: T, save: (value: T) => Promise<unknown>, delay = 250): SaveStatus {
  const [status, setStatus] = useState<SaveStatus>('idle')
  const savedRef = useRef(value)
  const isFirstRef = useRef(true)

  useEffect(() => {
    if (isFirstRef.current) {
      isFirstRef.current = false
      savedRef.current = value
      return
    }
    if (JSON.stringify(value) === JSON.stringify(savedRef.current)) return

    setStatus('saving')
    const timer = setTimeout(() => {
      save(value)
        .then(() => {
          savedRef.current = value
          setStatus('saved')
        })
        .catch(() => setStatus('error'))
    }, delay)

    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value])

  return status
}
