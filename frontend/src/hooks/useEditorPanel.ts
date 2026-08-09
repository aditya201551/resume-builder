import { createContext, useContext } from 'react'

export interface OpenPanelHandle {
  close: () => void
  commit?: () => void
}

export interface EditorPanelContextValue {
  slot: HTMLDivElement | null
  pushOpen: (handle: OpenPanelHandle) => () => void
}

export const EditorPanelContext = createContext<EditorPanelContextValue>({
  slot: null,
  pushOpen: () => () => {},
})

export function useEditorPanel() {
  return useContext(EditorPanelContext)
}
