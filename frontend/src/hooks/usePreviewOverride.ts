import { createContext, useContext } from 'react'

export interface PreviewOverrideContextValue {
  overrides: Record<string, Record<string, unknown>>
  setOverride: (id: string, patch: Record<string, unknown>) => void
  clearOverride: (id: string) => void
}

export const PreviewOverrideContext = createContext<PreviewOverrideContextValue>({
  overrides: {},
  setOverride: () => {},
  clearOverride: () => {},
})

export function usePreviewOverride() {
  return useContext(PreviewOverrideContext)
}
