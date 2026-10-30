import type { FullResume } from '@/types/resume'
import { markdownToHtml } from './markdown'
import { resolveResumeContentData } from './resumeContent'

// Legacy HTML/print rendering. The PDF worker supplies its own DOM-free Markdown parser.
export function resolveResumeContent(data: FullResume) {
  return resolveResumeContentData(data, markdownToHtml)
}
