import { createContext, useContext, useEffect, useReducer, useRef, useState, type ReactNode } from 'react'
import { apiGet } from '@/lib/http'
import { loadDraft, saveDraft } from '@/lib/resumeDraftStorage'
import { flushDraft, isDraftDirty } from '@/lib/resumeDraftSync'
import { draftReducer, type DraftAction, type DraftState } from '@/hooks/resumeDraftReducer'
import type { FullResume } from '@/types/resume'

export type SyncStatus = 'idle' | 'dirty' | 'syncing' | 'error'

interface ResumeDraftContextValue {
  state: DraftState
  dispatch: (action: DraftAction) => void
  resumeId: string
  syncStatus: SyncStatus
  flushNow: () => Promise<void>
}

const ResumeDraftContext = createContext<ResumeDraftContextValue | null>(null)

const FLUSH_INTERVAL_MS = 5000

export function ResumeDraftProvider({ resumeId, children }: { resumeId: string; children: ReactNode }) {
  const [state, dispatch] = useReducer(draftReducer, null as unknown as DraftState)
  const [ready, setReady] = useState(false)
  const [syncStatus, setSyncStatus] = useState<SyncStatus>('idle')
  const stateRef = useRef(state)
  stateRef.current = state
  const flushingRef = useRef(false)

  // Hydrate once: prefer a local draft (may hold edits never flushed to the
  // server, e.g. the tab closed mid-session); only hit the server if this
  // resume has never been opened locally before.
  useEffect(() => {
    let cancelled = false
    setReady(false)
    const cached = loadDraft(resumeId)
    if (cached) {
      dispatch({ type: 'hydrate', data: cached.data, lastSynced: cached.lastSynced })
      setReady(true)
      return
    }
    apiGet<FullResume>(`/api/resumes/${resumeId}/full`).then((data) => {
      if (cancelled) return
      dispatch({ type: 'replace_all', data })
      setReady(true)
    })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resumeId])

  // Persist to localStorage on every change.
  useEffect(() => {
    if (!ready || !state) return
    saveDraft(resumeId, { data: state.data, lastSynced: state.lastSynced })
  }, [resumeId, ready, state])

  async function runFlush() {
    if (flushingRef.current || !stateRef.current) return
    if (!isDraftDirty(stateRef.current.data, stateRef.current.lastSynced)) {
      setSyncStatus('idle')
      return
    }
    flushingRef.current = true
    setSyncStatus('syncing')
    try {
      await flushDraft(resumeId, stateRef.current, dispatch)
      setSyncStatus('idle')
    } catch {
      setSyncStatus('error')
    } finally {
      flushingRef.current = false
    }
  }

  useEffect(() => {
    if (!ready) return
    const timer = setInterval(runFlush, FLUSH_INTERVAL_MS)
    return () => clearInterval(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ready, resumeId])

  // Best-effort: flush on navigating away from the editor so a page close
  // right before the next interval tick doesn't strand up to 5s of edits.
  useEffect(() => {
    return () => {
      void runFlush()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resumeId])

  if (!ready || !state) return null

  return (
    <ResumeDraftContext.Provider value={{ state, dispatch, resumeId, syncStatus, flushNow: runFlush }}>
      {children}
    </ResumeDraftContext.Provider>
  )
}

export function useResumeDraftContext() {
  const ctx = useContext(ResumeDraftContext)
  if (!ctx) throw new Error('useResumeDraftContext must be used within a ResumeDraftProvider')
  return ctx
}

/** Read-only accessor mirroring the old `useFullResume` shape for the editor. */
export function useResumeDraftData() {
  const ctx = useContext(ResumeDraftContext)
  if (!ctx) return { data: undefined, isLoading: true }
  return { data: ctx.state.data, isLoading: false }
}
