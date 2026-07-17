import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiGet, apiPatch, apiPost, apiPut } from '@/lib/http'
import type { FullResume } from '@/types/resume'

export function fullResumeKey(resumeId: string) {
  return ['resumes', resumeId, 'full'] as const
}

export function useFullResume(resumeId: string) {
  return useQuery({
    queryKey: fullResumeKey(resumeId),
    queryFn: () => apiGet<FullResume>(`/api/resumes/${resumeId}/full`),
    enabled: Boolean(resumeId),
  })
}

/**
 * Every child entity (flat or nested) is edited through the same
 * create/update/delete/reorder shape — this wraps that against a given
 * base path and invalidates the aggregate `full` query on success, since
 * that's the single source of truth the editor renders from.
 */
export function useEntityMutations(resumeId: string, basePath: string) {
  const queryClient = useQueryClient()
  const invalidate = () => queryClient.invalidateQueries({ queryKey: fullResumeKey(resumeId) })

  const create = useMutation({
    mutationFn: (input: unknown) => apiPost(basePath, input),
    onSuccess: invalidate,
  })

  const update = useMutation({
    mutationFn: ({ id, input }: { id: string; input: unknown }) => apiPatch(`${basePath}/${id}`, input),
    onSuccess: invalidate,
  })

  const remove = useMutation({
    mutationFn: (id: string) => apiDelete(`${basePath}/${id}`),
    onSuccess: invalidate,
  })

  const reorder = useMutation({
    mutationFn: (orderedIds: string[]) => apiPut(`${basePath}/reorder`, { ordered_ids: orderedIds }),
    onSuccess: invalidate,
  })

  return { create, update, remove, reorder }
}
