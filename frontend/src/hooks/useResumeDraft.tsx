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
  flushNow: () => Promise<boolean>
  hasDirtyPanel: boolean
  registerPanel: (key: string, commit: (() => void) | null) => void
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

export function useResumeDraftData() {
  const ctx = useContext(ResumeDraftContext)
  if (!ctx) return { data: undefined, isLoading: true }
  return { data: ctx.state.data, isLoading: false }
}
