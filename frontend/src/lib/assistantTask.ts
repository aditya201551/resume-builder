export interface AssistantTask {
  prompt: string
  jobDescription?: string
}

export function assistantMessage(
  prompt: string,
  jobDescription: string,
): string {
  if (!jobDescription.trim()) return prompt.trim()
  return `${prompt.trim()}\n\nUse this supplied job description as context when suggesting changes. Emphasize relevant experience already present in my resume. Ask me for missing facts; do not invent skills, responsibilities, or achievement metrics.\n\n<job_description>\n${jobDescription.trim()}\n</job_description>`
}

export const ASSISTANT_STARTERS = [
  {
    title: 'Strengthen my writing',
    description: 'Clearer words. Stronger experience.',
    prompt:
      'Review my resume and propose clearer, more concise wording for my summary and experience. Preserve the facts and ask me for missing details.',
  },
  {
    title: 'Review my resume',
    description: 'A fresh perspective on your story.',
    prompt:
      'Review my resume for clarity, grammar, and structure. Explain the most useful improvements and propose specific changes using only the information I have provided.',
  },
  {
    title: 'Tailor to a role',
    description: 'Bring your relevant experience forward.',
    prompt:
      'Help me tailor this resume to a role. If I have not supplied a job description, ask me for one first. Propose changes supported by my experience and keep the original facts.',
  },
] as const
