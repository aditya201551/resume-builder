import type { ResumeDesign } from '@/types/design'
import type { ResumeContent } from './contract'
import ClassicTemplate from './classic/ClassicTemplate'

export interface TemplateComponentProps {
  content: ResumeContent
  tokens: ResumeDesign
  onReady?: () => void
}

export type TemplateComponent = (props: TemplateComponentProps) => React.JSX.Element

// Keyed by the templates table's renderer_key, not its id — the id is a
// per-database UUID, renderer_key is the stable string this registry and
// the seeded migration both agree on. Only "classic" exists today; once
// GET /api/templates is consumed by a template picker (later phase), this
// map grows one entry per new renderer without touching call sites.
export const templateRegistry: Record<string, TemplateComponent> = {
  classic: ClassicTemplate,
}

const DEFAULT_RENDERER_KEY = 'classic'

export function resolveTemplateComponent(rendererKey?: string): TemplateComponent {
  return (rendererKey && templateRegistry[rendererKey]) || templateRegistry[DEFAULT_RENDERER_KEY]
}
