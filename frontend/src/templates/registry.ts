import type { ResumeDesign } from '@/types/design'
import type { ResumeContent } from './contract'
import ClassicTemplate from './classic/ClassicTemplate'
import VerticalSplitTemplate from './verticalSplit/VerticalSplitTemplate'

export interface TemplateComponentProps {
  content: ResumeContent
  tokens: ResumeDesign
  onReady?: () => void
}

export type TemplateComponent = (props: TemplateComponentProps) => React.JSX.Element

export const templateRegistry: Record<string, TemplateComponent> = {
  classic: ClassicTemplate,
  'vertical-split': VerticalSplitTemplate,
}

const DEFAULT_RENDERER_KEY = 'classic'

export function resolveTemplateComponent(rendererKey?: string): TemplateComponent {
  return (rendererKey && templateRegistry[rendererKey]) || templateRegistry[DEFAULT_RENDERER_KEY]
}
