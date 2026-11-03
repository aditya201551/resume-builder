import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useAuth } from '@/hooks/useAuth'
import { apiGet } from '@/lib/http'
import { FeatureContext, type FeatureFlags } from './context'

export function FeatureProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const query = useQuery({
    queryKey: ['features', user?.id],
    queryFn: ({ signal }) => apiGet<FeatureFlags>('/api/features', { signal }),
    enabled: Boolean(user),
    staleTime: 0,
    gcTime: 0,
    refetchInterval: 10_000,
    refetchOnWindowFocus: 'always',
    retry: false,
  })
  // Don't display old enabled decisions when returning to the application or
  // when a refresh fails. Core resume screens do not depend on this request.
  const fresh = Boolean(user) && query.isFetchedAfterMount && !query.isError
  return (
    <FeatureContext value={{
      flags: fresh ? query.data ?? {} : {},
      loading: Boolean(user) && !query.isFetchedAfterMount,
      unavailable: query.isError,
    }}>
      {children}
    </FeatureContext>
  )
}
