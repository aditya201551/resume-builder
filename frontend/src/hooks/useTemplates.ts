import { templates } from '@/templates/catalog'

export function useTemplates() {
  return { data: templates, isLoading: false }
}
