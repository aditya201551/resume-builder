import { useState } from 'react'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'

export function useEntryPanel() {
  const [openKey, setOpenKey] = useState<string | null>(null)
  const { getEntryKey } = useResumeDraftContext()

  return {
    openNew: (id: string) => setOpenKey(getEntryKey(id)),
    getPanelProps: (id: string) => {
      const key = getEntryKey(id)
      return {
        open: openKey === key,
        onOpenChange: (open: boolean) => setOpenKey(open ? key : null),
      }
    },
  }
}
