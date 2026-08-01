import { useQuery } from '@tanstack/react-query'
import { apiGet } from '@/lib/http'
import type { Template } from '@/types/template'

/** The template catalog rarely changes and isn't resume-scoped, so this is
 * a plain read with no invalidation wiring beyond react-query's defaults. */
export function useTemplates() {
  return useQuery({
    queryKey: ['templates'],
    queryFn: () => apiGet<Template[]>('/api/templates'),
  })
}
