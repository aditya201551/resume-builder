import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiGet, apiPut } from '@/lib/http'
import { applySaved } from '@/lib/jobs'
import { useAuth } from '@/hooks/useAuth'
import type { Job, JobCountry, JobPlace, JobSearchResult } from '@/types/job'

export function useJobs(query: string) {
  const { user } = useAuth()
  return useQuery({
    queryKey: ['jobs', user?.id, query],
    queryFn: ({ signal }) => apiGet<JobSearchResult>(`/api/jobs?${query}`, { signal }),
    placeholderData: (previous, previousQuery) => previousQuery?.queryKey[1] === user?.id ? previous : undefined,
    enabled: Boolean(user),
    staleTime: 60_000,
    refetchInterval: query => query.state.data?.refreshing || query.state.data?.sources.some(s => s.status === 'refreshing') ? 5_000 : false,
    retry: 1,
  })
}

export function useJob(id: string | null) {
  const { user } = useAuth()
  return useQuery({ queryKey: ['job-detail', user?.id, id], queryFn: () => apiGet<Job>(`/api/jobs/${id}`), enabled: Boolean(id && user), staleTime: 60_000, retry: 1 })
}

export function useJobCountries() {
  return useQuery({ queryKey: ['job-countries'], queryFn: () => apiGet<JobCountry[]>('/api/jobs/countries'), staleTime: Infinity, retry: 1 })
}

export function useJobPlaces(query: string) {
  return useQuery({ queryKey: ['job-places', query], queryFn: () => apiGet<JobPlace[]>(`/api/jobs/places?q=${encodeURIComponent(query)}`), enabled: query.trim().length >= 2, staleTime: 24 * 60 * 60 * 1000, retry: 1 })
}

export function useSaveJob() {
  const client = useQueryClient()
  const { user } = useAuth()
  return useMutation({
    mutationFn: ({ id, saved }: { id: string; saved: boolean }) => saved ? apiPut<void>(`/api/jobs/${id}/saved`) : apiDelete(`/api/jobs/${id}/saved`),
    onSuccess: (_, { id, saved }) => {
      client.setQueriesData<JobSearchResult>({ queryKey: ['jobs', user?.id] }, result => result ? applySaved(result, id, saved) : result)
      client.setQueryData<Job>(['job-detail', user?.id, id], job => job ? { ...job, saved } : job)
      void client.invalidateQueries({ queryKey: ['jobs', user?.id] })
    },
  })
}
