import { useCallback, useEffect, useRef, useState } from 'react'
import { apiPost } from '@/lib/http'
import { readJobIntent, type JobRefinement, type SearchIntent } from '@/lib/jobSearch'

export interface JobChatMessage {
  id: number
  role: 'user' | 'assistant'
  text: string
  applied?: boolean
}

export function useJobAssistant(intent: SearchIntent, saved: boolean, onApply: (intent: SearchIntent) => void) {
  const [messages, setMessages] = useState<JobChatMessage[]>([])
  const [thinking, setThinking] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const messagesRef = useRef<JobChatMessage[]>([])
  const requestRef = useRef<AbortController | null>(null)
  const nextId = useRef(0)
  const lastRequest = useRef('')
  const context = useRef({ intent, saved, onApply })
  context.current = { intent, saved, onApply }

  const append = useCallback((message: Omit<JobChatMessage, 'id'>) => {
    const next = [...messagesRef.current, { ...message, id: nextId.current++ }]
    messagesRef.current = next
    setMessages(next)
  }, [])

  useEffect(() => () => requestRef.current?.abort(), [])
  useEffect(() => {
    requestRef.current?.abort()
    requestRef.current = null
    messagesRef.current = []
    lastRequest.current = ''
    setMessages([])
    setError(null)
    setThinking(false)
  }, [saved])

  const send = useCallback(async (raw: string, retry = false) => {
    const text = raw.trim()
    if (!text || text.length > 2000 || requestRef.current) return
    const start = context.current
    const version = JSON.stringify(start.intent)
    const controller = new AbortController()
    requestRef.current = controller
    lastRequest.current = text
    const history = messagesRef.current.slice(0, retry ? -1 : undefined).slice(-4).map(({ role, text }) => ({ role, text }))
    if (!retry) append({ role: 'user', text })
    setError(null)
    setThinking(true)
    try {
      const response = await apiPost<JobRefinement>('/api/jobs/assist', { intent: start.intent, message: text, history }, { signal: controller.signal })
      if (controller.signal.aborted || requestRef.current !== controller) return
      if (context.current.saved !== start.saved || JSON.stringify(context.current.intent) !== version) {
        append({ role: 'assistant', text: 'Your search criteria changed while I was working. I kept the latest filters. Send your refinement again to use them.' })
        return
      }
      if (typeof response.message !== 'string' || typeof response.apply !== 'boolean') throw new Error('Invalid assistant response')
      if (response.apply) {
        const next = readJobIntent(JSON.stringify(response.intent))
        // Never turn a malformed response into the default, broader search.
        if (JSON.stringify(next) !== JSON.stringify(response.intent)) throw new Error('Invalid search criteria')
        context.current.onApply(next)
      }
      append({ role: 'assistant', text: response.message, applied: response.apply })
    } catch {
      if (!controller.signal.aborted && requestRef.current === controller) setError('The assistant could not complete this request. Your search criteria are unchanged. Try again or use Manual search.')
    } finally {
      if (requestRef.current === controller) {
        requestRef.current = null
        setThinking(false)
      }
    }
  }, [append])

  const cancel = useCallback(() => {
    if (!requestRef.current) return
    requestRef.current.abort()
    requestRef.current = null
    setThinking(false)
    setError(null)
    append({ role: 'assistant', text: 'Stopped. Your search criteria are unchanged.' })
  }, [append])

  const clear = useCallback(() => {
    requestRef.current?.abort()
    requestRef.current = null
    messagesRef.current = []
    lastRequest.current = ''
    setMessages([])
    setThinking(false)
    setError(null)
  }, [])

  const retry = useCallback(() => send(lastRequest.current, true), [send])
  return { messages, thinking, error, send, cancel, clear, retry }
}

export type JobAssistantController = ReturnType<typeof useJobAssistant>
