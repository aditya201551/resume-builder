import { useMemo } from 'react'
import { markdownToHtml } from '@/lib/markdown'
import styles from './ChatMarkdown.module.css'

/**
 * Renders finalized assistant text as Markdown (bold, lists, links) via the
 * same tiptap-markdown pipeline the resume content editor/preview uses, so
 * output styling stays visually consistent with the rest of the app. Only
 * meant for text that's done streaming — re-parsing partial Markdown on
 * every token would both be wasteful (a full headless Tiptap editor spins
 * up per call) and visually noisy (a stray "**" mid-stream renders oddly
 * until its closing pair arrives). Plain-text rendering during an
 * in-progress stream is handled by the caller instead.
 */
export default function ChatMarkdown({ text }: { text: string }) {
  const html = useMemo(() => markdownToHtml(text), [text])
  if (!html) return null
  return <div className={styles.richText} dangerouslySetInnerHTML={{ __html: html }} />
}
