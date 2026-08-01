import type { FullResume } from '@/types/resume'

export interface StoredDraft {
  data: FullResume
  lastSynced: FullResume
}

// Bump this whenever FullResume's shape changes in a way an old cached
// draft can't safely stand in for (e.g. adding the `design` field — an old
// draft simply has no `design` key at all, not a default one). loadDraft
// discards anything that doesn't match, forcing a fresh GET /full instead
// of the app having to guess at a migration for arbitrarily-shaped stale
// local state.
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
