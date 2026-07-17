import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import type { SectionConfig } from '@/types/resume'

/**
 * Reordering just reassigns each config's own sort_order — applied locally
 * and instantly; the actual PATCH-per-row network writes happen later via
 * the draft's periodic flush (there's no bulk reorder endpoint for section
 * configs, unlike every other child entity, so the flush still sends one
 * PATCH per changed row, just batched on a timer instead of per drag).
 */
export function useSectionConfigReorder(resumeId: string) {
  void resumeId
  const { dispatch } = useResumeDraftContext()

  return {
    mutate: (ordered: SectionConfig[]) => {
      dispatch({ type: 'section_config_reorder', ordered: ordered.map((c, index) => ({ ...c, sort_order: index })) })
    },
  }
}
