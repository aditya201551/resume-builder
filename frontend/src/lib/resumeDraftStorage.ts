import type { FullResume } from '@/types/resume'

export interface StoredDraft {
  data: FullResume
  lastSynced: FullResume
}

const DRAFT_SCHEMA_VERSION = 2

interface StoredEnvelope extends StoredDraft {
  version: number
}

function storageKey(resumeId: string) {
  return `resume-draft:${resumeId}`
}

export function loadDraft(resumeId: string): StoredDraft | null {
  try {
    const raw = localStorage.getItem(storageKey(resumeId))
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<StoredEnvelope>
    if (parsed.version !== DRAFT_SCHEMA_VERSION) return null
    if (!parsed.data || !parsed.lastSynced) return null
    return { data: parsed.data, lastSynced: parsed.lastSynced }
  } catch {
    return null
  }
}

export function saveDraft(resumeId: string, draft: StoredDraft) {
  try {
    const envelope: StoredEnvelope = { version: DRAFT_SCHEMA_VERSION, ...draft }
    localStorage.setItem(storageKey(resumeId), JSON.stringify(envelope))
  } catch {
  }
}

export function clearDraft(resumeId: string) {
  try {
    localStorage.removeItem(storageKey(resumeId))
  } catch {
  }
}
