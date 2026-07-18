import { useCallback, useEffect, useRef, useState } from 'react'

/**
 * Drives which entry's RowCard panel is open within a section, plus which
 * ones are "new" (created via the Add button and not yet closed once) so
 * RowCard can discard them on close if they were left blank — e.g. Add
 * Education then close without typing anything shouldn't leave an empty
 * "Untitled" row behind.
 *
 * `ids` must be the current list of entity ids in this section. A newly
 * created entry gets a client-side temp id that the background draft sync
 * later resolves to a real server id (rewriting `entry.id` app-wide) — if
 * that happens while its panel is open, `openId` would otherwise point at
 * an id that no longer exists and the panel would appear to slam shut. When
 * exactly one id disappears and exactly one new one appears between renders,
 * we treat it as that resolution and carry `openId`/pending-new status over
 * to the new id.
 *
 * `pendingNewIds` is only cleared for an entry the render *after* it stops
 * being the open one, so RowCard's own effect (a descendant, whose effects
 * always run before this hook's) gets one commit where it can see
 * `isNew && !open` together and run its discard check before we drop it.
 */
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
