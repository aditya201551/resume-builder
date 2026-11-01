import { useInfiniteQuery } from '@tanstack/react-query'
import { useAuth } from '@/hooks/useAuth'
import { apiPost } from '@/lib/http'
import type { DiscoveryPage, SearchIntent } from '@/lib/jobSearch'

export function useJobDiscovery(intent: SearchIntent, savedOnly: boolean, enabled: boolean) {
  const { user } = useAuth()
  return useInfiniteQuery({
    queryKey: ['jobs', user?.id, 'discovery', intent, savedOnly],
    queryFn: ({ pageParam, signal }) => apiPost<DiscoveryPage>('/api/jobs/search', { intent, savedOnly, nextToken: pageParam }, { signal }),
    initialPageParam: '',
    getNextPageParam: page => page.nextToken || undefined,
    enabled: Boolean(user) && enabled,
    staleTime: 60_000,
    retry: false,
    refetchOnWindowFocus: false,
  })
}
