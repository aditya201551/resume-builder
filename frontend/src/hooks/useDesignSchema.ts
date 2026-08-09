import { useQuery } from '@tanstack/react-query'
import { apiGet } from '@/lib/http'
import type { FieldGroup } from '@/types/designSchema'

export function useDesignSchema() {
  return useQuery({
    queryKey: ['design-schema'],
    queryFn: () => apiGet<FieldGroup[]>('/api/design-schema'),
    staleTime: Infinity,
  })
}
