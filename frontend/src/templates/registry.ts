import type { ResumeDesign } from '@/types/design'
import type { ResumeContent } from './contract'
import VerticalHtmlTemplate from './vertical/VerticalHtmlTemplate'

export interface TemplateComponentProps {
  content: ResumeContent
  tokens: ResumeDesign
  onReady?: () => void
}

export type TemplateComponent = (props: TemplateComponentProps) => React.JSX.Element

export const templateRegistry: Record<string, TemplateComponent> = {
  vertical: VerticalHtmlTemplate,
}

const DEFAULT_RENDERER_KEY = 'vertical'

export function resolveTemplateComponent(rendererKey?: string): TemplateComponent {
  const component = templateRegistry[rendererKey ?? DEFAULT_RENDERER_KEY]
  if (!component) throw new Error(`Legacy HTML renderer is not registered: ${rendererKey}`)
  return component
}
