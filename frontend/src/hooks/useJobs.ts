import { useMutation, useQuery, useQueryClient, type InfiniteData } from '@tanstack/react-query'
import { apiDelete, apiGet, apiPut } from '@/lib/http'
import { useAuth } from '@/hooks/useAuth'
import type { Job, JobCountry } from '@/types/job'
import type { DiscoveryPage } from '@/lib/jobSearch'

export function useJob(id: string | null) {
  const { user } = useAuth()
  const client = useQueryClient()
  return useQuery({ queryKey: ['job-detail', user?.id, id], queryFn: async ({ signal }) => {
    const job = await apiGet<Job>(`/api/jobs/${id}`, { signal })
    if (job.closed) void client.invalidateQueries({ queryKey: ['jobs', user?.id] })
    return job
  }, enabled: Boolean(id && user), staleTime: 60_000, retry: 1 })
}

export function useJobCountries() {
  return useQuery({ queryKey: ['job-countries'], queryFn: () => apiGet<JobCountry[]>('/api/jobs/countries'), staleTime: Infinity, retry: 1 })
}

export function useSaveJob() {
  const client = useQueryClient()
  const { user } = useAuth()
  return useMutation({
    mutationFn: ({ id, saved }: { id: string; saved: boolean }) => saved ? apiPut<void>(`/api/jobs/${id}/saved`) : apiDelete(`/api/jobs/${id}/saved`),
    onSuccess: (_, { id, saved }) => {
      client.setQueriesData<InfiniteData<DiscoveryPage>>({ queryKey: ['jobs', user?.id] }, result => {
        if (!result) return result
        return { ...result, pages: result.pages.map(page => ({ ...page, jobs: page.jobs.map(job => job.id === id ? { ...job, saved } : job) })) }
      })
      client.setQueryData<Job>(['job-detail', user?.id, id], job => job ? { ...job, saved } : job)
      void client.invalidateQueries({ queryKey: ['jobs', user?.id] })
    },
  })
}
