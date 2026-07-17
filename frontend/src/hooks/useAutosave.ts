import { useEffect, useRef, useState } from 'react'

export type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'

/**
 * Debounces `value` changes into a save call. Skips the mount-time value
 * (it's already persisted) and no-ops when the value round-trips back to
 * whatever was last saved, so typing-then-undoing doesn't fire a request.
 */
export function useAutosave<T>(value: T, save: (value: T) => Promise<unknown>, delay = 800): SaveStatus {
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
