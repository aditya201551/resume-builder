import type { Resume } from '@/types/resume'

export const MAX_RESUME_PDF_BYTES = 10 * 1024 * 1024

export function resumeFileError(file: File): string | null {
  if (!file.name.toLowerCase().endsWith('.pdf')) return 'Choose a PDF file. Word documents and images are not supported yet.'
  if (file.size === 0) return 'This file is empty. Choose your resume PDF.'
  if (file.size > MAX_RESUME_PDF_BYTES) return 'Choose a PDF smaller than 10 MB.'
  return null
}

export interface ResumeImportResult { resume: Resume; warnings: string[] }

export async function importResume(file: File): Promise<ResumeImportResult> {
  const error = resumeFileError(file)
  if (error) throw new Error(error)
  const data = new FormData()
  data.append('file', file)
  const response = await fetch('/api/resumes/import', { method: 'POST', credentials: 'include', body: data })
  if (!response.ok) {
    const body = await response.json().catch(() => null) as { message?: string } | null
    throw new Error(body?.message ?? (response.status === 401 ? 'Your session expired. Sign in and try again.' : "Import couldn't be completed. Check your library before trying again."))
  }
  return response.json()
}
