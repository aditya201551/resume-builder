import { createContext, useContext } from 'react'

export interface OpenPanelHandle {
  /** Closes the panel (used by Escape). */
  close: () => void
  /** Commits its current field values without closing (used by Ctrl/Cmd+S). */
  commit?: () => void
}

/**
 * DOM node that entry editors portal into (lives in EditorPage's content
 * pane, layered on top of the section list so editing replaces the list in
 * place while the live preview stays visible), plus a stack of open-panel
 * handles so Escape only dismisses the innermost editor when panels are
 * nested (e.g. a custom-section entry opened from within its already-open
 * section editor), and Ctrl/Cmd+S can commit every currently open editor.
 */
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
