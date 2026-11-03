import { createContext } from 'react'

export type FeatureKey = 'jobs.enabled'
export type FeatureFlags = Partial<Record<FeatureKey, boolean>>

interface FeatureState {
  flags: FeatureFlags
  loading: boolean
  unavailable: boolean
}

export const FeatureContext = createContext<FeatureState>({
  flags: {}, loading: false, unavailable: true,
})
