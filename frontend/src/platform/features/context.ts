import { createContext } from 'react'

// Feature names are owned by the backend catalog.
export type FeatureKey = string
export type FeatureFlags = Partial<Record<FeatureKey, boolean>>

interface FeatureState {
  flags: FeatureFlags
  loading: boolean
  unavailable: boolean
}

export const FeatureContext = createContext<FeatureState>({
  flags: {}, loading: false, unavailable: true,
})
