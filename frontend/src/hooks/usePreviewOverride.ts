import { createContext, useContext } from 'react'

/**
 * Lets an open entry editor push its in-progress (uncommitted) field values
 * so the live preview can reflect them instantly, without touching the
 * actual draft — the draft only updates when the user clicks Done. Keyed by
 * entity id; cleared when the editor closes (Done or otherwise), at which
 * point the real committed/reverted value in the draft is authoritative again.
 */
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
