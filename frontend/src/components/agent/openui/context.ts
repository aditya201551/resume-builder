import { createContext, useContext } from 'react'
import type { AssistantUI, UIFormState, UIReviewAction } from '@/lib/assistantUI'

export interface UIInteraction {
  ui: AssistantUI
  disabled: boolean
  formState: UIFormState
  onStateChange: (state: UIFormState) => void
  onReview: (action: UIReviewAction) => void
}
export const UIInteractionContext = createContext<UIInteraction | null>(null)
export function useUIInteraction() {
  const context = useContext(UIInteractionContext)
  if (!context) throw new Error('Interactive resume card is missing its context.')
  return context
}
