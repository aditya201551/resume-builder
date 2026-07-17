import type { FullResume } from '@/types/resume'

export interface StoredDraft {
  data: FullResume
  lastSynced: FullResume
}

function storageKey(resumeId: string) {
  return `resume-draft:${resumeId}`
}

export function loadDraft(resumeId: string): StoredDraft | null {
  try {
    const raw = localStorage.getItem(storageKey(resumeId))
    if (!raw) return null
    return JSON.parse(raw) as StoredDraft
  } catch {
    return null
  }
}

export function saveDraft(resumeId: string, draft: StoredDraft) {
  try {
    localStorage.setItem(storageKey(resumeId), JSON.stringify(draft))
  } catch {
    // localStorage can throw (quota, private mode) — local-only persistence
    // is a nice-to-have, not worth crashing the editor over.
  }
}

export function clearDraft(resumeId: string) {
  try {
    localStorage.removeItem(storageKey(resumeId))
  } catch {
    // ignore
  }
}
