import { useRef, useState, type SetStateAction } from 'react'

// Keep raw form text (e.g. a trailing comma in technologies) while committing its
// normalized value immediately. External changes, including AI edits, refresh the form.
export function useDraftForm<T, P>(source: T, toPatch: (value: T) => P, commit: (patch: P) => unknown) {
  const sourceKey = JSON.stringify(toPatch(source))
  const [local, setLocal] = useState({ sourceKey, value: source })
  let value = local.value
  if (sourceKey !== local.sourceKey) {
    value = sourceKey === JSON.stringify(toPatch(local.value)) ? local.value : source
    setLocal({ sourceKey, value })
  }
  const valueRef = useRef(value)
  valueRef.current = value

  function setValue(update: SetStateAction<T>) {
    const next = typeof update === 'function' ? (update as (value: T) => T)(valueRef.current) : update
    valueRef.current = next
    setLocal({ sourceKey: JSON.stringify(toPatch(next)), value: next })
    commit(toPatch(next))
  }
  return [value, setValue] as const
}
