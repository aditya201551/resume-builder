import { Component, useCallback, useState, type ReactNode } from 'react'
import { Renderer, type OpenUIError } from '@openuidev/react-lang'
import { assistantLibrary } from './library'
import { UIInteractionContext } from './context'
import type { AssistantUI, UIFormState, UIReviewAction } from '@/lib/assistantUI'

function Fallback({ text }: { text: string }) {
  return <div className="rounded-2xl bg-secondary/30 p-4 text-sm"><p className="font-medium">This interactive card couldn't be displayed.</p><p className="mt-2 whitespace-pre-wrap break-words text-muted-foreground">{text}</p><p className="mt-2 text-xs text-muted-foreground">Ask the assistant to try again, or continue in Content.</p></div>
}

class UIErrorBoundary extends Component<{ children: ReactNode; fallback: string }, { failed: boolean }> {
  state = { failed: false }
  static getDerivedStateFromError() { return { failed: true } }
  render() { return this.state.failed ? <Fallback text={this.props.fallback} /> : this.props.children }
}

export default function AssistantUIMessage({ ui, formState, disabled, reviewStatus, error, onStateChange, onReview }: {
  ui: AssistantUI
  formState: UIFormState
  disabled: boolean
  reviewStatus?: 'pending' | 'accepted' | 'rejected'
  error?: string
  onStateChange: (state: UIFormState) => void
  onReview: (action: UIReviewAction) => void
}) {
  const [failed, setFailed] = useState(false)
  const onError = useCallback((errors: OpenUIError[]) => {
    if (errors.some((error) => error.severity !== 'warning')) setFailed(true)
  }, [])
  const submitted = reviewStatus === 'pending' || reviewStatus === 'accepted'
  return <UIErrorBoundary fallback={ui.fallback}>
    {failed ? <Fallback text={ui.fallback} /> : <UIInteractionContext.Provider value={{ ui, formState, disabled: disabled || submitted, onStateChange, onReview }}>
      {/* No toolProvider, MCP client, model-selected callbacks or external URLs. */}
      <Renderer library={assistantLibrary} response={ui.code} isStreaming={false} onError={onError} publishObservability={false} />
      {error && <p role="alert" className="mt-2 rounded-xl bg-destructive/10 p-3 text-xs text-destructive">{error}</p>}
      {submitted && <p role="status" className="mt-2 text-xs text-muted-foreground">{reviewStatus === 'accepted' ? 'Applied to your resume.' : 'Your proposed change is ready to review below.'}</p>}
    </UIInteractionContext.Provider>}
  </UIErrorBoundary>
}
