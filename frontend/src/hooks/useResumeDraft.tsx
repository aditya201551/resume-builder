import { createContext, useCallback, useContext, useEffect, useReducer, useRef, useState, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { apiGet } from '@/lib/http'
import { loadDraft, saveDraft } from '@/lib/resumeDraftStorage'
import { flushDraft, isDraftDirty } from '@/lib/resumeDraftSync'
import { createDraftAutosync, type SyncStatus } from '@/lib/resumeDraftAutosync'
import { createEntryIdentity, resolveEntryReferences } from '@/lib/resumeDraftIds'
import { draftReducer, type DraftAction, type DraftState } from '@/hooks/resumeDraftReducer'
import type { FullResume } from '@/types/resume'

interface ResumeDraftContextValue {
  state: DraftState
  dispatch: (action: DraftAction) => void
  resumeId: string
  syncStatus: SyncStatus
  flushNow: () => Promise<boolean>
  getEntryKey: (id: string) => string
  resolveEntryId: (id: string) => string
}

const ResumeDraftContext = createContext<ResumeDraftContextValue | null>(null)

export function ResumeDraftProvider({ resumeId, children }: { resumeId: string; children: ReactNode }) {
  return <ResumeDraftSession key={resumeId} resumeId={resumeId}>{children}</ResumeDraftSession>
}

function ResumeDraftSession({ resumeId, children }: { resumeId: string; children: ReactNode }) {
  const [state, reactDispatch] = useReducer(draftReducer, null as unknown as DraftState)
  const [syncStatus, setSyncStatus] = useState<SyncStatus>('idle')
  const stateRef = useRef<DraftState | null>(null)
  const syncRef = useRef<ReturnType<typeof createDraftAutosync> | null>(null)
  const [identity] = useState(createEntryIdentity)
  const queryClient = useQueryClient()

  const dispatch = useCallback((action: DraftAction) => {
    if (action.type === 'resolve_ids') {
      identity.register(action.idMap)
    } else {
      action = resolveEntryReferences(action, identity.resolveId)
    }
    // Update this ref synchronously so an in-flight sync sees edits from this event,
    // even before React commits the render. Persist before scheduling any network work.
    stateRef.current = draftReducer(stateRef.current as DraftState, action)
    saveDraft(resumeId, stateRef.current)
    reactDispatch(action)
    syncRef.current?.schedule()
  }, [resumeId, identity])

  useEffect(() => {
    const sync = createDraftAutosync({
      getState: () => stateRef.current,
      sync: (snapshot) => flushDraft(resumeId, snapshot, dispatch),
      onStatus: setSyncStatus,
      onSynced: () => {
        void queryClient.invalidateQueries({ queryKey: ['resumes'] })
      },
    })
    syncRef.current = sync
    sync.schedule()
    const flush = () => { void sync.flush() }
    const onVisibilityChange = () => {
      if (document.visibilityState === 'hidden') flush()
    }
    window.addEventListener('online', flush)
    window.addEventListener('blur', flush)
    document.addEventListener('visibilitychange', onVisibilityChange)
    return () => {
      sync.dispose()
      syncRef.current = null
      window.removeEventListener('online', flush)
      window.removeEventListener('blur', flush)
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  }, [resumeId, dispatch, queryClient])

  useEffect(() => {
    let cancelled = false
    const cached = loadDraft(resumeId)
    if (cached) {
      dispatch({ type: 'hydrate', data: cached.data, lastSynced: cached.lastSynced })
      return
    }
    apiGet<FullResume>(`/api/resumes/${resumeId}/full`).then((data) => {
      if (!cancelled) dispatch({ type: 'replace_all', data })
    })
    return () => { cancelled = true }
  }, [resumeId, dispatch])

  useEffect(() => {
    function handleBeforeUnload(e: BeforeUnloadEvent) {
      const current = stateRef.current
      if (current && isDraftDirty(current.data, current.lastSynced)) {
        e.preventDefault()
        e.returnValue = ''
      }
    }
    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [])

  const flushNow = useCallback(() => syncRef.current?.flush() ?? Promise.resolve(false), [])

  if (!state) return null

  return (
    <ResumeDraftContext.Provider value={{
      state, dispatch, resumeId, syncStatus, getEntryKey: identity.getKey, resolveEntryId: identity.resolveId,
      flushNow,
    }}>
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
