import { useContext } from 'react'
import { FeatureContext, type FeatureKey } from './context'

export function useFeatures() {
  const state = useContext(FeatureContext)
  return {
    ...state,
    enabled: (key: FeatureKey) => !state.loading && !state.unavailable && state.flags[key] === true,
  }
}
