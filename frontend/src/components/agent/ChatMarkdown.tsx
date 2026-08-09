import { useMemo } from 'react'
import { markdownToHtml } from '@/lib/markdown'
import styles from './ChatMarkdown.module.css'

export default function ChatMarkdown({ text }: { text: string }) {
  const html = useMemo(() => markdownToHtml(text), [text])
  if (!html) return null
  return <div className={styles.richText} dangerouslySetInnerHTML={{ __html: html }} />
}
