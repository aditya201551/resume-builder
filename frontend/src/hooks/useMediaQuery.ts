import { useCallback, useSyncExternalStore } from 'react'

export function useMediaQuery(query: string, serverMatch = false) {
  const subscribe = useCallback((onChange: () => void) => {
    const media = window.matchMedia(query)
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [query])
  const getSnapshot = useCallback(() => window.matchMedia(query).matches, [query])
  const getServerSnapshot = useCallback(() => serverMatch, [serverMatch])
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
}
