import type { FullResume } from '@/types/resume'
import { resolveResumeContent } from '@/lib/resolveResumeContent'
import { defaultResumeDesign } from '@/lib/defaultResumeDesign'
import { resolveTemplateComponent } from '@/templates/registry'
import { getTemplate, normalizeTemplateDesign } from '@/templates/catalog'

export default function LivePreview({ data, onReady }: { data: FullResume; onReady?: () => void }) {
  const tokens = normalizeTemplateDesign(data.design ?? defaultResumeDesign)
  const content = resolveResumeContent({ ...data, design: tokens })
  const rendererKey = getTemplate(tokens.templateId).renderer_key
  const Template = resolveTemplateComponent(rendererKey)
  return <Template content={content} tokens={tokens} onReady={onReady} />
}
