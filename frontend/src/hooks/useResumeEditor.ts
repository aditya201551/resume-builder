import { useQuery } from '@tanstack/react-query'
import { apiGet } from '@/lib/http'
import { tempId } from '@/lib/tempId'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import type { FlatKind } from '@/hooks/resumeDraftReducer'
import type { FullResume } from '@/types/resume'

export function fullResumeKey(resumeId: string) {
  return ['resumes', resumeId, 'full'] as const
}

/** Read-only fetch — used by the dashboard's resume thumbnails, which aren't editing. */
export function useFullResume(resumeId: string) {
  return useQuery({
    queryKey: fullResumeKey(resumeId),
    queryFn: () => apiGet<FullResume>(`/api/resumes/${resumeId}/full`),
    enabled: Boolean(resumeId),
  })
}

type ParsedPath =
  | { kind: 'flat'; entity: FlatKind }
  | { kind: 'skill_groups' }
  | { kind: 'skill_items'; groupId: string }
  | { kind: 'custom_sections' }
  | { kind: 'custom_entries'; sectionId: string }

const FLAT_SEGMENTS: Record<string, FlatKind> = {
  'work-experiences': 'work_experiences',
  educations: 'educations',
  projects: 'projects',
  certifications: 'certifications',
  languages: 'languages',
  'misc-entries': 'misc_entries',
}

function parseBasePath(basePath: string): ParsedPath {
  const match = basePath.match(/^\/api\/resumes\/[^/]+\/(.+)$/)
  const segments = match?.[1].split('/') ?? []

  if (segments.length === 1) {
    if (segments[0] === 'skill-groups') return { kind: 'skill_groups' }
    if (segments[0] === 'custom-sections') return { kind: 'custom_sections' }
    const entity = FLAT_SEGMENTS[segments[0]]
    if (entity) return { kind: 'flat', entity }
  }
  if (segments.length === 3 && segments[0] === 'skill-groups' && segments[2] === 'items') {
    return { kind: 'skill_items', groupId: segments[1] }
  }
  if (segments.length === 3 && segments[0] === 'custom-sections' && segments[2] === 'entries') {
    return { kind: 'custom_entries', sectionId: segments[1] }
  }
  throw new Error(`useEntityMutations: cannot parse basePath "${basePath}"`)
}

function localMutation<TArg, TResult = void>(fn: (arg: TArg) => TResult) {
  return {
    mutate: (arg: TArg) => fn(arg),
    mutateAsync: async (arg: TArg) => fn(arg),
    isPending: false,
  }
}

/**
 * Every child entity (flat or nested) is edited through the same
 * create/update/delete/reorder shape. This used to hit the network directly
 * (via react-query mutations) and invalidate the aggregate query on success;
 * it now applies the change to the local draft instantly (no debounce, no
 * round trip) — the actual network write happens later, batched, via the
 * draft's periodic flush. Callers are unchanged: same `.mutate`/`.mutateAsync`
 * shape as before.
 */
export function useEntityMutations(resumeId: string, basePath: string) {
  void resumeId
  const { dispatch } = useResumeDraftContext()
  const parsed = parseBasePath(basePath)

  const create = localMutation((fields: Record<string, unknown>) => {
    const id = tempId()
    switch (parsed.kind) {
      case 'flat':
        dispatch({ type: 'flat_create', entity: parsed.entity, tempId: id, fields })
        break
      case 'skill_groups':
        dispatch({ type: 'skill_group_create', tempId: id, fields })
        break
      case 'skill_items':
        dispatch({ type: 'skill_item_create', groupId: parsed.groupId, tempId: id, fields })
        break
      case 'custom_sections':
        dispatch({ type: 'custom_section_create', tempId: id, fields })
        break
      case 'custom_entries':
        dispatch({ type: 'custom_entry_create', sectionId: parsed.sectionId, tempId: id, fields })
        break
    }
    return id
  })

  const update = localMutation(({ id, input }: { id: string; input: Record<string, unknown> }) => {
    switch (parsed.kind) {
      case 'flat':
        dispatch({ type: 'flat_update', entity: parsed.entity, id, patch: input })
        break
      case 'skill_groups':
        dispatch({ type: 'skill_group_update', id, patch: input })
        break
      case 'skill_items':
        dispatch({ type: 'skill_item_update', groupId: parsed.groupId, id, patch: input })
        break
      case 'custom_sections':
        dispatch({ type: 'custom_section_update', id, patch: input })
        break
      case 'custom_entries':
        dispatch({ type: 'custom_entry_update', sectionId: parsed.sectionId, id, patch: input })
        break
    }
  })

  const remove = localMutation((id: string) => {
    switch (parsed.kind) {
      case 'flat':
        dispatch({ type: 'flat_delete', entity: parsed.entity, id })
        break
      case 'skill_groups':
        dispatch({ type: 'skill_group_delete', id })
        break
      case 'skill_items':
        dispatch({ type: 'skill_item_delete', groupId: parsed.groupId, id })
        break
      case 'custom_sections':
        dispatch({ type: 'custom_section_delete', id })
        break
      case 'custom_entries':
        dispatch({ type: 'custom_entry_delete', sectionId: parsed.sectionId, id })
        break
    }
  })

  const reorder = localMutation((orderedIds: string[]) => {
    switch (parsed.kind) {
      case 'flat':
        dispatch({ type: 'flat_reorder', entity: parsed.entity, orderedIds })
        break
      case 'skill_groups':
        dispatch({ type: 'skill_group_reorder', orderedIds })
        break
      case 'skill_items':
        dispatch({ type: 'skill_item_reorder', groupId: parsed.groupId, orderedIds })
        break
      case 'custom_sections':
        dispatch({ type: 'custom_section_reorder', orderedIds })
        break
      case 'custom_entries':
        dispatch({ type: 'custom_entry_reorder', sectionId: parsed.sectionId, orderedIds })
        break
    }
  })

  return { create, update, remove, reorder }
}
