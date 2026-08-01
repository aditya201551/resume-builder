import { useEffect, useMemo, useRef, useState } from 'react'
import ChatMarkdown from '@/components/agent/ChatMarkdown'
import styles from './StreamingMarkdown.module.css'

function useSmoothText(target: string, active: boolean) {
  const [displayed, setDisplayed] = useState(target)
  const displayedRef = useRef(displayed)
  displayedRef.current = displayed

  useEffect(() => {
    if (!active) {
      setDisplayed(target)
      return
    }

    if (target.length < displayedRef.current.length) {
      setDisplayed(target)
      return
    }

    let frame = 0
    const tick = () => {
      const current = displayedRef.current
      if (current.length >= target.length) return

      const remaining = target.length - current.length
      const step = Math.min(Math.max(Math.ceil(remaining / 18), 1), 8)
      setDisplayed(target.slice(0, current.length + step))
      frame = window.requestAnimationFrame(tick)
    }

    frame = window.requestAnimationFrame(tick)
    return () => window.cancelAnimationFrame(frame)
  }, [target, active])

  return displayed
}

function splitStableMarkdown(text: string, streaming: boolean) {
  if (!streaming) return { stable: text, tail: '' }

  const lastBreak = text.lastIndexOf('\n')
  if (lastBreak === -1) return { stable: '', tail: text }

  return {
    stable: text.slice(0, lastBreak + 1),
    tail: text.slice(lastBreak + 1),
  }
}

export default function StreamingMarkdown({ text, streaming }: { text: string; streaming: boolean }) {
  const displayed = useSmoothText(text, streaming)
  const { stable, tail } = useMemo(() => splitStableMarkdown(displayed, streaming), [displayed, streaming])
  const showCursor = streaming && displayed.length > 0

  return (
    <div className={styles.response}>
      {stable && <ChatMarkdown text={stable} />}
      {(tail || showCursor) && (
        <span className={styles.tail}>
          {tail}
          {showCursor && <span className={styles.cursor} aria-hidden />}
        </span>
      )}
    </div>
  )
}
