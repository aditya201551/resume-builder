import { useQuery } from '@tanstack/react-query'
import { apiGet } from '@/lib/http'
import type { Template } from '@/types/template'

export function useTemplates() {
  return useQuery({
    queryKey: ['templates'],
    queryFn: () => apiGet<Template[]>('/api/templates'),
  })
}
