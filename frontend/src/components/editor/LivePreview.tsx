import type { FullResume } from '@/types/resume'
import { resolveResumeContent } from '@/lib/resolveResumeContent'
import { defaultResumeDesign } from '@/lib/defaultResumeDesign'
import { resolveTemplateComponent } from '@/templates/registry'
import { useTemplates } from '@/hooks/useTemplates'

export default function LivePreview({ data, onReady }: { data: FullResume; onReady?: () => void }) {
  const content = resolveResumeContent(data)
  const tokens = data.design ?? defaultResumeDesign
  const { data: templates } = useTemplates()
  const rendererKey = templates?.find((t) => t.id === tokens.templateId)?.renderer_key
  const Template = resolveTemplateComponent(rendererKey)
  return <Template content={content} tokens={tokens} onReady={onReady} />
}
