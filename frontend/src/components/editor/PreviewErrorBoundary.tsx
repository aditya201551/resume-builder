import { Component, type ErrorInfo, type ReactNode } from 'react'
import { AlertTriangle } from 'lucide-react'

interface Props {
  children: ReactNode
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
