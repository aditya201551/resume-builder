import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiPatch } from '@/lib/http'
import { sectionConfigUrl } from '@/lib/sectionConfig'
import { fullResumeKey } from '@/hooks/useResumeEditor'
import type { SectionConfig } from '@/types/resume'

/**
 * There's no bulk reorder endpoint for section configs (unlike every other
 * child entity) — sort_order is just one field on the per-(section_type,
 * custom_section_id) PATCH, so a reorder is N individual PATCH calls.
 */
export function useSectionConfigReorder(resumeId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (ordered: SectionConfig[]) =>
      Promise.all(
        ordered.map((c, index) =>
          apiPatch(sectionConfigUrl(resumeId, c), {
            region: c.region,
            is_visible: c.is_visible,
            display_title_override: c.display_title_override,
            sort_order: index,
          }),
        ),
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: fullResumeKey(resumeId) }),
  })
}
