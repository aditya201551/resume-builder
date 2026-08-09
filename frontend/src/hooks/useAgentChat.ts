import { useCallback, useRef, useState } from 'react'
import { streamAgentChat, type AgentProposal, type AgentToolEvent } from '@/lib/agentChat'
import { canApplyProposal, proposalToDraftAction } from '@/lib/agentProposal'
import { loadAutoAccept, saveAutoAccept } from '@/lib/agentChatPrefs'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'

export type AssistantPart =
  | { kind: 'text'; text: string }
  | { kind: 'tool'; key: string; event: AgentToolEvent }
  | { kind: 'proposal'; key: string; proposal: AgentProposal; status: 'pending' | 'accepted' | 'rejected' }

export type ChatMessage = { role: 'user'; content: string } | { role: 'assistant'; parts: AssistantPart[] }

let proposalKeyCounter = 0
function nextProposalKey() {
  return `p${proposalKeyCounter++}`
}

let toolKeyCounter = 0
function toolKey(event: AgentToolEvent) {
  return event.id || `${event.name}-${toolKeyCounter++}`
}

function assistantToPlain(m: Extract<ChatMessage, { role: 'assistant' }>): { role: 'assistant'; content: string } {
  return { role: 'assistant', content: m.parts.filter((p) => p.kind === 'text').map((p) => p.text).join('') }
}

export function useAgentChat() {
  const { resumeId, state, dispatch } = useResumeDraftContext()

  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [streamingParts, setStreamingParts] = useState<AssistantPart[] | null>(null)
  const [isStreaming, setIsStreaming] = useState(false)
  const [autoAccept, setAutoAcceptState] = useState(loadAutoAccept)
  const [error, setError] = useState<string | null>(null)

  const setAutoAccept = useCallback((value: boolean) => {
    setAutoAcceptState(value)
    saveAutoAccept(value)
  }, [])

  const abortRef = useRef<AbortController | null>(null)
  const draftRef = useRef(state.data)
  draftRef.current = state.data
  const streamingPartsRef = useRef<AssistantPart[]>([])

  const createdParentIdsRef = useRef(new Set<string>())

  const applyProposal = useCallback(
    (proposal: AgentProposal): boolean => {
      const action = proposalToDraftAction(proposal)
      if (!action) return false

      const applicability = canApplyProposal(proposal, draftRef.current, createdParentIdsRef.current)
      if (!applicability.ok) return false

      dispatch(action)
      if (proposal.tempId && (proposal.type === 'skill_group_create' || proposal.type === 'custom_section_create')) {
        createdParentIdsRef.current.add(proposal.tempId)
      }
      return true
    },
    [dispatch],
  )

  const pushPart = useCallback((part: AssistantPart) => {
    streamingPartsRef.current = [...streamingPartsRef.current, part]
    setStreamingParts(streamingPartsRef.current)
  }, [])

  const appendStreamingText = useCallback((text: string) => {
    const parts = streamingPartsRef.current
    const last = parts[parts.length - 1]
    const next =
      last?.kind === 'text' ? [...parts.slice(0, -1), { kind: 'text' as const, text: last.text + text }] : [...parts, { kind: 'text' as const, text }]
    streamingPartsRef.current = next
    setStreamingParts(next)
  }, [])

  const upsertToolEvent = useCallback((event: AgentToolEvent) => {
    const key = toolKey(event)
    const parts = streamingPartsRef.current
    const existingIndex = parts.findIndex((p) => p.kind === 'tool' && (p.key === key || (!event.id && p.event.name === event.name)))
    const next =
      existingIndex === -1
        ? [...parts, { kind: 'tool' as const, key, event }]
        : parts.map((part, i) => (i === existingIndex && part.kind === 'tool' ? { ...part, event: { ...part.event, ...event } } : part))

    streamingPartsRef.current = next
    setStreamingParts(next)
  }, [])

  const retryPendingParts = useCallback(() => {
    let progressed = true
    while (progressed) {
      progressed = false
      const next = streamingPartsRef.current.map((part) => {
        if (part.kind !== 'proposal' || part.status !== 'pending') return part
        if (!applyProposal(part.proposal)) return part
        progressed = true
        return { ...part, status: 'accepted' as const }
      })
      if (progressed) {
        streamingPartsRef.current = next
        setStreamingParts(next)
      }
    }
  }, [applyProposal])

  const finishStreaming = useCallback(() => {
    const finishedParts = streamingPartsRef.current
    if (finishedParts.length > 0) {
      setMessages((prev) => [...prev, { role: 'assistant', parts: finishedParts }])
    }
    streamingPartsRef.current = []
    setStreamingParts(null)
    setIsStreaming(false)
  }, [])

  const sendMessage = useCallback(
    async (content: string) => {
      if (!content.trim() || isStreaming) return
      setError(null)

      const history = [
        ...messages.map((m) => (m.role === 'user' ? { role: 'user' as const, content: m.content } : assistantToPlain(m))),
        { role: 'user' as const, content },
      ]
      setMessages((prev) => [...prev, { role: 'user', content }])
      createdParentIdsRef.current = new Set()
      streamingPartsRef.current = []
      setStreamingParts(streamingPartsRef.current)
      setIsStreaming(true)

      const controller = new AbortController()
      abortRef.current = controller

      await streamAgentChat(
        resumeId,
        { messages: history, draft: draftRef.current },
        {
          onToken: appendStreamingText,
          onProposal: (proposal) => {
            const key = nextProposalKey()
            const applied = autoAccept && applyProposal(proposal)
            pushPart({ kind: 'proposal', key, proposal, status: applied ? 'accepted' : 'pending' })
            if (applied) retryPendingParts()
          },
          onTool: upsertToolEvent,
          onDone: finishStreaming,
          onError: (message) => {
            setError(message)
            finishStreaming()
          },
        },
        controller.signal,
      )
    },
    [
      messages,
      isStreaming,
      resumeId,
      autoAccept,
      appendStreamingText,
      applyProposal,
      pushPart,
      upsertToolEvent,
      finishStreaming,
      retryPendingParts,
    ],
  )

  const cancel = useCallback(() => {
    abortRef.current?.abort()
    finishStreaming()
  }, [finishStreaming])

  const resolveProposal = useCallback(
    (messageIndex: number, key: string, accept: boolean) => {
      setMessages((prev) =>
        prev.map((m, i) => {
          if (i !== messageIndex || m.role !== 'assistant') return m
          return {
            ...m,
            parts: m.parts.map((part) => {
              if (part.kind !== 'proposal' || part.key !== key || part.status !== 'pending') return part
              if (accept && !applyProposal(part.proposal)) return part
              return { ...part, status: accept ? 'accepted' : 'rejected' }
            }),
          }
        }),
      )
    },
    [applyProposal],
  )

  return {
    messages,
    streamingParts,
    isStreaming,
    autoAccept,
    setAutoAccept,
    error,
    sendMessage,
    cancel,
    resolveProposal,
  }
}
