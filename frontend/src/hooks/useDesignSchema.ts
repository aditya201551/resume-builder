import { useQuery } from '@tanstack/react-query'
import { apiGet } from '@/lib/http'
import type { FieldGroup } from '@/types/designSchema'

/** The field schema is static (code, not per-resume data) and identical for
 * every template — see backend design.Schema()'s comment. */
export function useDesignSchema() {
  return useQuery({
    queryKey: ['design-schema'],
    queryFn: () => apiGet<FieldGroup[]>('/api/design-schema'),
    staleTime: Infinity,
  })
}
