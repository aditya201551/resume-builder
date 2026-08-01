import { Component, type ErrorInfo, type ReactNode } from 'react'
import { AlertTriangle } from 'lucide-react'

/**
 * Last line of defence around the live preview.
 *
 * The preview renders whatever the draft holds, and the draft is not fully
 * trusted data: it takes agent proposals, and it rehydrates from localStorage
 * written by possibly-older versions of the app. A single malformed entity
 * used to throw during render, which — with no boundary anywhere above it —
 * unmounted the whole editor and lost the user's unflushed edits along with
 * it. Containing the failure here keeps the editor and its draft alive, so
 * the user can undo or fix whatever produced the bad entity.
 *
 * Class component because React only exposes error boundaries via
 * componentDidCatch/getDerivedStateFromError; there is no hook equivalent.
 */
interface Props {
  children: ReactNode
  /**
   * The data being previewed. Compared by identity to decide when to retry:
   * a new draft is a new chance to render, so an edit that fixes the
   * offending entity brings the preview back on its own. Deliberately not
   * `children` — that's a fresh element on every parent render, which would
   * retry (and re-throw) on renders that changed nothing.
   */
  resetKey: unknown
}

interface State {
  error: Error | null
  lastResetKey: unknown
}

export default class PreviewErrorBoundary extends Component<Props, State> {
  state: State = { error: null, lastResetKey: this.props.resetKey }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { error }
  }

  static getDerivedStateFromProps(props: Props, state: State): Partial<State> | null {
    if (props.resetKey === state.lastResetKey) return null
    return { error: null, lastResetKey: props.resetKey }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Live preview failed to render:', error, info.componentStack)
  }

  render() {
    if (!this.state.error) return this.props.children

    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 p-8 text-center">
        <AlertTriangle className="size-8 text-muted-foreground" />
        <p className="text-sm font-medium text-foreground">This preview couldn&apos;t be rendered</p>
        <p className="max-w-xs text-xs text-muted-foreground">
          One of the entries on this resume is malformed. Your work is still saved — edit or remove the most recent change and the
          preview will come back.
        </p>
        <p className="max-w-xs font-mono text-[11px] text-muted-foreground/70">{this.state.error.message}</p>
      </div>
    )
  }
}
