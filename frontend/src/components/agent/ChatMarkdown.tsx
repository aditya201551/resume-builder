import { useMemo, useRef } from 'react'
import { markdownToHtml } from '@/lib/markdown'
import { useWordReveal } from '@/hooks/useWordReveal'
import styles from './ChatMarkdown.module.css'

export default function ChatMarkdown({ text, animateWords = false, streaming = false }: { text: string; animateWords?: boolean; streaming?: boolean }) {
  const html = useMemo(() => markdownToHtml(text), [text])
  const rootRef = useRef<HTMLDivElement>(null)
  useWordReveal(rootRef, html, animateWords)
  if (!html) return null
  return <div ref={rootRef} className={styles.richText} data-streaming={streaming || undefined} dangerouslySetInnerHTML={{ __html: html }} />
}
