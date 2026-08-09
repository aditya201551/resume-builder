import { useCallback, useEffect, useRef, useState } from 'react'

export function useEntryPanel(ids: string[]) {
  const [openId, setOpenId] = useState<string | null>(null)
  const [pendingNewIds, setPendingNewIds] = useState<Set<string>>(new Set())
  const prevOpenId = useRef<string | null>(null)
  const prevIds = useRef<string[]>(ids)

  useEffect(() => {
    const before = prevIds.current
    prevIds.current = ids
    const appeared = ids.filter((id) => !before.includes(id))
    const disappeared = before.filter((id) => !ids.includes(id))
    if (appeared.length === 1 && disappeared.length === 1) {
      const [oldId] = disappeared
      const [newId] = appeared
      setOpenId((cur) => (cur === oldId ? newId : cur))
      setPendingNewIds((cur) => {
        if (!cur.has(oldId)) return cur
        const next = new Set(cur)
        next.delete(oldId)
        next.add(newId)
        return next
      })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ids])

  useEffect(() => {
    const prev = prevOpenId.current
    prevOpenId.current = openId
    if (prev && prev !== openId) {
      setPendingNewIds((ids) => {
        if (!ids.has(prev)) return ids
        const next = new Set(ids)
        next.delete(prev)
        return next
      })
    }
  }, [openId])

  const openNew = useCallback((id: string) => {
    setPendingNewIds((ids) => new Set(ids).add(id))
    setOpenId(id)
  }, [])

  const getPanelProps = useCallback(
    (id: string) => ({
      open: openId === id,
      isNew: pendingNewIds.has(id),
      onOpenChange: (open: boolean) => setOpenId(open ? id : null),
    }),
    [openId, pendingNewIds],
  )

  return { openNew, getPanelProps }
}
