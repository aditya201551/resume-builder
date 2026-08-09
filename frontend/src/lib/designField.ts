import type { ResumeDesign } from '@/types/design'

export function getFieldValue(design: ResumeDesign, key: string): unknown {
  const parts = key.split('.')
  let cur: unknown = design
  for (const part of parts) {
    cur = (cur as Record<string, unknown>)[part]
  }
  return cur
}

function buildPatch(obj: Record<string, unknown>, parts: string[], value: unknown): Record<string, unknown> {
  const [head, ...rest] = parts
  if (rest.length === 0) return { [head]: value }
  return { [head]: { ...(obj[head] as Record<string, unknown>), ...buildPatch(obj[head] as Record<string, unknown>, rest, value) } }
}

export function buildFieldPatch(design: ResumeDesign, key: string, value: unknown): Partial<ResumeDesign> {
  return buildPatch(design as unknown as Record<string, unknown>, key.split('.'), value) as Partial<ResumeDesign>
}
