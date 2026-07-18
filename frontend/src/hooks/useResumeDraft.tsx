import { createContext, useCallback, useContext, useEffect, useReducer, useRef, useState, type ReactNode } from 'react'
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
  /** Resolves true on success/no-op, false if the flush failed — callers
   * that need to know whether it's actually safe to proceed (e.g. "Save &
   * leave") should check this instead of assuming success. */
  flushNow: () => Promise<boolean>
  /** True while any entry editor panel has uncommitted (not-yet-Done) form
   * edits — a RowCard registers its commit function via `registerPanel`
   * while open and dirty. Folded into the app-wide "unsaved changes" signal
   * (see EditorNavbar's SyncStatus) since those edits are real unsaved work
   * even though they haven't reached `state.data` yet. */
  hasDirtyPanel: boolean
  registerPanel: (key: string, commit: (() => void) | null) => void
  /** Commits every currently-open dirty panel's form into the draft — the
   * same thing Ctrl+S does for open panels, exposed so other "save before
   * leaving" actions (see EditorNavbar's BackButton) can do the same thing. */
  commitDirtyPanels: () => void
}

const ResumeDraftContext = createContext<ResumeDraftContextValue | null>(null)

export function ResumeDraftProvider({ resumeId, children }: { resumeId: string; children: ReactNode }) {
  const [state, dispatch] = useReducer(draftReducer, null as unknown as DraftState)
  const [ready, setReady] = useState(false)
  const [syncStatus, setSyncStatus] = useState<SyncStatus>('idle')
  const stateRef = useRef(state)
  stateRef.current = state
  const flushingRef = useRef(false)

  const [dirtyPanels, setDirtyPanels] = useState<Map<string, () => void>>(new Map())
  const dirtyPanelsRef = useRef(dirtyPanels)
  dirtyPanelsRef.current = dirtyPanels
  const registerPanel = useCallback((key: string, commit: (() => void) | null) => {
    setDirtyPanels((prev) => {
      if (commit === null) {
        if (!prev.has(key)) return prev
        const next = new Map(prev)
        next.delete(key)
        return next
      }
      const next = new Map(prev)
      next.set(key, commit)
      return next
    })
  }, [])
  function commitDirtyPanels() {
    for (const commit of dirtyPanelsRef.current.values()) commit()
  }

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

  async function runFlush(): Promise<boolean> {
    if (flushingRef.current || !stateRef.current) return true
    if (!isDraftDirty(stateRef.current.data, stateRef.current.lastSynced)) {
      setSyncStatus('idle')
      return true
    }
    flushingRef.current = true
    setSyncStatus('syncing')
    try {
      await flushDraft(resumeId, stateRef.current, dispatch)
      setSyncStatus('idle')
      return true
    } catch {
      setSyncStatus('error')
      return false
    } finally {
      flushingRef.current = false
    }
  }

  // No autosave: the backend is only ever written to via an explicit user
  // action — Ctrl/Cmd+S (handled in EditPane, which also commits any open
  // entry editor first) or the "Save & leave" choice in the unsaved-changes
  // guard below. Everything else just lives in local state + localStorage
  // (see the persist effect above) until the user asks to save.
  useEffect(() => {
    function handleBeforeUnload(e: BeforeUnloadEvent) {
      if (!stateRef.current) return
      const dirty = isDraftDirty(stateRef.current.data, stateRef.current.lastSynced) || dirtyPanelsRef.current.size > 0
      if (dirty) {
        e.preventDefault()
        e.returnValue = ''
      }
    }
    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [])

  if (!ready || !state) return null

  return (
    <ResumeDraftContext.Provider
      value={{
        state,
        dispatch,
        resumeId,
        syncStatus,
        flushNow: runFlush,
        hasDirtyPanel: dirtyPanels.size > 0,
        registerPanel,
        commitDirtyPanels,
      }}
    >
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
