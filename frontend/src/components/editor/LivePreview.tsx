import type { FullResume } from '@/types/resume'
import { resolveResumeContent } from '@/lib/resolveResumeContent'
import { defaultResumeDesign } from '@/lib/defaultResumeDesign'
import { resolveTemplateComponent } from '@/templates/registry'

// Thin wrapper around the template-engine: resolve content (data ->
// ResumeContent, decoupled from the draft/sync system), look up the
// renderer, hand off. This component itself no longer knows anything about
// resume sections, pagination, or CSS — that all lives in templates/*.
//
// Template selection is hardcoded to "classic" for now: there is exactly
// one template in the catalog, and mapping data.design.templateId (a
// database UUID) to a renderer_key requires the template catalog fetched
// from GET /api/templates, which the future template-picker UI will own.
export default function LivePreview({ data, onReady }: { data: FullResume; onReady?: () => void }) {
  const content = resolveResumeContent(data)
  const Template = resolveTemplateComponent('classic')
  // data.design can be missing on a draft cached in localStorage before this
  // field existed on the backend — see defaultResumeDesign's comment.
  return <Template content={content} tokens={data.design ?? defaultResumeDesign} onReady={onReady} />
}
