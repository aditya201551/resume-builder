import type { FullResume } from '@/types/resume'
import { resolveResumeContent } from '@/lib/resolveResumeContent'
import { defaultResumeDesign } from '@/lib/defaultResumeDesign'
import { resolveTemplateComponent } from '@/templates/registry'
import { useTemplates } from '@/hooks/useTemplates'

// Thin wrapper around the template-engine: resolve content (data ->
// ResumeContent, decoupled from the draft/sync system), look up the
// renderer, hand off. This component itself no longer knows anything about
// resume sections, pagination, or CSS — that all lives in templates/*.
export default function LivePreview({ data, onReady }: { data: FullResume; onReady?: () => void }) {
  const content = resolveResumeContent(data)
  // data.design can be missing on a draft cached in localStorage before this
  // field existed on the backend — see defaultResumeDesign's comment.
  const tokens = data.design ?? defaultResumeDesign
  // Mapping design.templateId (a database UUID) to a renderer_key needs the
  // template catalog — fail open to "classic" while it's loading or if the
  // id isn't found, same pattern DesignMode's capability gating uses.
  const { data: templates } = useTemplates()
  const rendererKey = templates?.find((t) => t.id === tokens.templateId)?.renderer_key
  const Template = resolveTemplateComponent(rendererKey)
  return <Template content={content} tokens={tokens} onReady={onReady} />
}
