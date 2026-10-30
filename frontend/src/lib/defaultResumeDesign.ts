import type { ResumeDesign } from '@/types/design'
import type { Template } from '@/types/template'
import catalog from '../../../backend/internal/design/templates.json'

export function createDefaultResumeDesign(templateId = catalog.defaultTemplateId): ResumeDesign {
  const templates = catalog.templates as Template[]
  const template = templates.find((entry) => entry.id === templateId)
    ?? templates.find((entry) => entry.id === catalog.defaultTemplateId)
  if (!template) throw new Error('The default resume template is not registered')
  return { templateId: template.id, ...structuredClone(template.defaults) }
}

export const defaultResumeDesign = createDefaultResumeDesign()
