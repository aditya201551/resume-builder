export function createEntryIdentity() {
  const resolved = new Map<string, string>()
  const keys = new Map<string, string>()
  const resolveId = (id: string): string => {
    const next = resolved.get(id)
    return next && next !== id ? resolveId(next) : id
  }
  const getKey = (id: string) => keys.get(resolveId(id)) ?? id
  return {
    resolveId,
    getKey,
    register(idMap: Record<string, string>) {
      for (const [oldId, newId] of Object.entries(idMap)) {
        const key = getKey(oldId)
        resolved.set(oldId, newId)
        keys.set(newId, key)
      }
    },
  }
}

// Event handlers and streamed proposals may still reference a temporary ID after a create completes.
export function resolveEntryReferences<T extends object>(value: T, resolveId: (id: string) => string): T {
  const result = { ...value } as Record<string, unknown>
  for (const key of ['id', 'groupId', 'sectionId', 'customSectionId']) {
    if (typeof result[key] === 'string') result[key] = resolveId(result[key])
  }
  if (Array.isArray(result.orderedIds)) result.orderedIds = result.orderedIds.map(resolveId)
  return result as T
}
