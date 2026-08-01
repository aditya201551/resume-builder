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

/**
 * Drives the resume_chat SSE endpoint and owns the conversation's client
 * state — history, in-progress streaming parts, and proposal accept/reject.
 * History is intentionally in-memory only (not persisted): the backend is
 * stateless per request (see backend/internal/agent/chat.go's package
 * comment), and this hook mirrors that on the client rather than adding
 * chat persistence that wasn't asked for.
 */
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
  // Proposals apply against the draft as it stood when the turn started —
  // reading the live ref instead of state avoids constructing every request
  // body from a stale render's snapshot.
  const draftRef = useRef(state.data)
  draftRef.current = state.data
  // The authoritative accumulator for the in-progress message. React defers
  // running setState *updater functions* until it flushes a batch — it does
  // NOT run them synchronously at the setState() call site. If several SSE
  // events (a few `token`s plus the trailing `done`) arrive in the same
  // network chunk, they're all dispatched synchronously back-to-back before
  // React has flushed anything, so a ref only mutated *inside* a
  // setStreamingParts updater would still read stale on `done`. Mutating
  // this ref directly and immediately — then using setStreamingParts only
  // to mirror it for rendering — makes it correct regardless of batching.
  const streamingPartsRef = useRef<AssistantPart[]>([])

  // tempIds of parent rows (skill groups, custom sections) created by
  // proposals accepted during this session. dispatch() doesn't update
  // draftRef synchronously — React hasn't re-rendered yet — so when a group
  // and its items are auto-accepted back-to-back from one stream, the items
  // would be judged against a draft that doesn't contain the group yet.
  // Recording the parent here closes that window.
  const createdParentIdsRef = useRef(new Set<string>())

  /**
   * Applies a proposal if the draft can actually accommodate it, and reports
   * whether it did. A child whose parent is missing is NOT applied: the
   * reducer would silently drop it (its `map` finds no matching parent) and
   * the user would see an accepted change that never happened.
   */
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

  /**
   * Re-attempts every still-pending proposal in the in-progress message,
   * looping until a pass applies nothing new. Only auto-accept uses this:
   * it makes the stream order-independent, so a child that arrived before
   * its parent still lands instead of being silently skipped. Anything that
   * remains pending stays on screen for the user to accept by hand once its
   * parent exists.
   */
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
      // Only meaningful within a turn: past turns' parents have long since
      // landed in the draft (or been rejected), and keeping their ids around
      // would let a child apply against a parent the user has since deleted.
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
            // Applying one proposal can unblock earlier ones (a group
            // arriving after an item that referenced it), so sweep again.
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
              // Stays pending if it can't be applied — the card explains
              // which parent is still missing, and the button re-enables on
              // its own once that parent is accepted.
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
