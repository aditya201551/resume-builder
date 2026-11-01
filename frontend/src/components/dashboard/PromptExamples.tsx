import { Fragment, useEffect, useState } from 'react'
import { AnimatePresence, type Variants } from 'motion/react'
import * as m from 'motion/react-m'
import { usePrefersReducedMotion } from '@/hooks/usePrefersReducedMotion'
import { wordReveal } from '@/lib/textMotion'

const examples = [
  'Make my experience sound more impactful…',
  'Review my resume and suggest improvements…',
  'Tailor my resume to a software engineer role…',
  'Help me write a compelling summary…',
]

const phraseMotion: Variants = {
  hidden: { opacity: 0, y: 10 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.3,
      ease: [0.22, 1, 0.36, 1],
      delayChildren: 0.025,
      staggerChildren: wordReveal.stagger,
    },
  },
  exit: { opacity: 0, y: -8, transition: { duration: 0.16, ease: 'easeIn' } },
}

const wordMotion: Variants = {
  hidden: { opacity: 0, y: wordReveal.distance },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: wordReveal.duration, ease: wordReveal.ease },
  },
}

// Mounted only while the field is empty, so typing removes the hint immediately.
export default function PromptExamples() {
  const [index, setIndex] = useState(0)
  const reducedMotion = usePrefersReducedMotion()

  useEffect(() => {
    if (reducedMotion) return
    const interval = window.setInterval(() => {
      if (!document.hidden) setIndex((current) => (current + 1) % examples.length)
    }, 3000)
    return () => window.clearInterval(interval)
  }, [reducedMotion])

  return (
    <div className="home-prompt-examples" aria-hidden="true">
      {reducedMotion ? (
        <span>{examples[index]}</span>
      ) : (
        <AnimatePresence initial={false} mode="wait">
          <m.span
            key={index}
            className="home-prompt-example"
            variants={phraseMotion}
            initial="hidden"
            animate="visible"
            exit="exit"
          >
            {examples[index].split(' ').map((word, wordIndex) => (
              <Fragment key={`${index}-${wordIndex}`}>
                {wordIndex > 0 && ' '}
                <m.span className="home-prompt-word" variants={wordMotion}>{word}</m.span>
              </Fragment>
            ))}
          </m.span>
        </AnimatePresence>
      )}
    </div>
  )
}
