import { useCallback, useEffect, useRef, useState } from 'react'
import {
  streamAgentChat,
  type AgentProposal,
  type AgentToolEvent,
} from '@/lib/agentChat'
import {
  canApplyProposal,
  editProposalText,
  proposalToDraftAction,
} from '@/lib/agentProposal'
import { loadAutoAccept, saveAutoAccept } from '@/lib/agentChatPrefs'
import { assistantMessage } from '@/lib/assistantTask'
import { resolveEntryReferences } from '@/lib/resumeDraftIds'
import { useResumeDraftContext } from '@/hooks/useResumeDraft'
import { tempId } from '@/lib/tempId'
import { uiReviewProposal, rewriteSource, type AssistantUI, type UIFormState, type UIReviewAction } from '@/lib/assistantUI'

export type AssistantPart =
  | { kind: 'text'; text: string }
  | { kind: 'tool'; key: string; event: AgentToolEvent }
  | { kind: 'ui'; key: string; ui: AssistantUI; formState: UIFormState; reviewKey?: string; error?: string }
  | {
      kind: 'proposal'
      key: string
      proposal: AgentProposal
      status: 'pending' | 'accepted' | 'rejected'
    }

export type ChatMessage =
  | { role: 'user'; content: string; modelContent?: string }
  | { role: 'assistant'; parts: AssistantPart[] }

let proposalKeyCounter = 0
function nextProposalKey() {
  return `p${proposalKeyCounter++}`
}

let toolKeyCounter = 0
function toolKey(event: AgentToolEvent) {
  return event.id || `${event.name}-${toolKeyCounter++}`
}

function assistantToPlain(m: Extract<ChatMessage, { role: 'assistant' }>): {
  role: 'assistant'
  content: string
} {
  return {
    role: 'assistant',
    content: m.parts
      .map((p) => p.kind === 'text' ? p.text : p.kind === 'ui' ? `[Interactive card: ${p.ui.fallback}]\n${p.ui.code}` : p.kind === 'proposal' ? `[Proposed resume change; ${p.status}: ${JSON.stringify(p.proposal)}]` : '')
      .join(''),
  }
}

export function useAgentChat() {
  const { resumeId, state, dispatch, resolveEntryId } = useResumeDraftContext()

  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [streamingParts, setStreamingParts] = useState<AssistantPart[] | null>(
    null,
  )
  const [isStreaming, setIsStreaming] = useState(false)
  const [autoAccept, setAutoAcceptState] = useState(loadAutoAccept)
  const [error, setError] = useState<string | null>(null)
  const autoAcceptRef = useRef(autoAccept)
  autoAcceptRef.current = autoAccept

  const setAutoAccept = useCallback((value: boolean) => {
    autoAcceptRef.current = value
    setAutoAcceptState(value)
    saveAutoAccept(value)
  }, [])

  const abortRef = useRef<AbortController | null>(null)
  const activeRef = useRef(false)
  const generationRef = useRef(0)
  const messagesRef = useRef(messages)
  messagesRef.current = messages
  useEffect(
    () => () => {
      generationRef.current++
      activeRef.current = false
      abortRef.current?.abort()
    },
    [],
  )
  const draftRef = useRef(state.data)
  draftRef.current = state.data
  const streamingPartsRef = useRef<AssistantPart[]>([])

  const createdParentIdsRef = useRef(new Set<string>())

  const applyProposal = useCallback(
    (proposal: AgentProposal): boolean => {
      if (proposal.sourceGuard && rewriteSource(proposal.sourceGuard, draftRef.current, resolveEntryId) !== proposal.sourceGuard.original) return false
      proposal = resolveEntryReferences(proposal, resolveEntryId)
      const action = proposalToDraftAction(proposal)
      if (!action) return false

      const applicability = canApplyProposal(
        proposal,
        draftRef.current,
        new Set([...createdParentIdsRef.current].map(resolveEntryId)),
      )
      if (!applicability.ok) return false

      dispatch(action)
      if (
        proposal.tempId &&
        (proposal.type === 'skill_group_create' ||
          proposal.type === 'custom_section_create')
      ) {
        createdParentIdsRef.current.add(proposal.tempId)
      }
      return true
    },
    [dispatch, resolveEntryId],
  )

  const pushPart = useCallback((part: AssistantPart) => {
    if (!activeRef.current) return
    streamingPartsRef.current = [...streamingPartsRef.current, part]
    setStreamingParts(streamingPartsRef.current)
  }, [])

  const appendStreamingText = useCallback((text: string) => {
    if (!activeRef.current) return
    const parts = streamingPartsRef.current
    const last = parts[parts.length - 1]
    const next =
      last?.kind === 'text'
        ? [
            ...parts.slice(0, -1),
            { kind: 'text' as const, text: last.text + text },
          ]
        : [...parts, { kind: 'text' as const, text }]
    streamingPartsRef.current = next
    setStreamingParts(next)
  }, [])

  const upsertToolEvent = useCallback((event: AgentToolEvent) => {
    if (!activeRef.current) return
    const key = toolKey(event)
    const parts = streamingPartsRef.current
    const existingIndex = parts.findIndex(
      (p) =>
        p.kind === 'tool' &&
        (p.key === key || (!event.id && p.event.name === event.name)),
    )
    const next =
      existingIndex === -1
        ? [...parts, { kind: 'tool' as const, key, event }]
        : parts.map((part, i) =>
            i === existingIndex && part.kind === 'tool'
              ? { ...part, event: { ...part.event, ...event } }
              : part,
          )

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
    if (!activeRef.current) return
    activeRef.current = false
    const finishedParts = streamingPartsRef.current
    if (finishedParts.length > 0) {
      const next: ChatMessage[] = [
        ...messagesRef.current,
        { role: 'assistant', parts: finishedParts },
      ]
      messagesRef.current = next
      setMessages(next)
    }
    streamingPartsRef.current = []
    setStreamingParts(null)
    setIsStreaming(false)
  }, [])

  const sendMessage = useCallback(
    async (content: string, jobDescription = '') => {
      if (!content.trim() || activeRef.current) return
      activeRef.current = true
      const generation = ++generationRef.current
      const current = () => generation === generationRef.current
      const modelContent = assistantMessage(content, jobDescription)
      setError(null)

      const history = [
        ...messagesRef.current.map((m) =>
          m.role === 'user'
            ? { role: 'user' as const, content: m.modelContent ?? m.content }
            : assistantToPlain(m),
        ),
        { role: 'user' as const, content: modelContent },
      ]
      const next: ChatMessage[] = [
        ...messagesRef.current,
        { role: 'user', content, modelContent },
      ]
      messagesRef.current = next
      setMessages(next)
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
          onToken: (text) => {
            if (current()) appendStreamingText(text)
          },
          onProposal: (proposal) => {
            if (!current()) return
            const key = nextProposalKey()
            const applied = autoAcceptRef.current && applyProposal(proposal)
            pushPart({
              kind: 'proposal',
              key,
              proposal,
              status: applied ? 'accepted' : 'pending',
            })
            if (applied) retryPendingParts()
          },
          onTool: (event) => {
            if (current()) upsertToolEvent(event)
          },
          onUI: (ui) => {
            if (!current() || streamingPartsRef.current.some((p) => p.kind === 'ui' && p.ui.id === ui.id)) return
            pushPart({ kind: 'ui', key: ui.id, ui, formState: {} })
          },
          onDone: () => {
            if (current()) finishStreaming()
          },
          onError: (message) => {
            if (!current()) return
            setError(message)
            finishStreaming()
          },
        },
        controller.signal,
      )
      if (current() && activeRef.current) {
        setError('The response ended before it was complete. Please try again.')
        finishStreaming()
      }
    },
    [
      resumeId,
      appendStreamingText,
      applyProposal,
      pushPart,
      upsertToolEvent,
      finishStreaming,
      retryPendingParts,
    ],
  )

  const cancel = useCallback(() => {
    generationRef.current++
    abortRef.current?.abort()
    finishStreaming()
  }, [finishStreaming])

  const resolveProposal = useCallback(
    (
      messageIndex: number,
      key: string,
      accept: boolean,
      edits: Record<string, string> = {},
    ) => {
      const message = messagesRef.current[messageIndex]
      const part =
        message?.role === 'assistant'
          ? message.parts.find(
              (part) => part.kind === 'proposal' && part.key === key,
            )
          : undefined
      if (!part || part.kind !== 'proposal' || part.status !== 'pending') return
      const proposal = editProposalText(part.proposal, edits)
      if (accept && !applyProposal(proposal)) return
      const next = messagesRef.current.map((m, i) =>
        i !== messageIndex || m.role !== 'assistant'
          ? m
          : {
              ...m,
              parts: m.parts.map((item) =>
                item.kind === 'proposal' && item.key === key
                  ? {
                      ...item,
                      proposal,
                      status: accept
                        ? ('accepted' as const)
                        : ('rejected' as const),
                    }
                  : item,
              ),
            },
      )
      messagesRef.current = next
      setMessages(next)
    },
    [applyProposal],
  )

  const updateUI = useCallback((messageIndex: number, key: string, update: (part: Extract<AssistantPart, { kind: 'ui' }>, parts: AssistantPart[]) => AssistantPart[]) => {
    const message = messagesRef.current[messageIndex]
    if (message?.role !== 'assistant') return
    const part = message.parts.find((p) => p.kind === 'ui' && p.key === key)
    if (part?.kind !== 'ui') return
    const next = messagesRef.current.map((m, i) => i === messageIndex ? { ...message, parts: update(part, message.parts) } : m)
    messagesRef.current = next
    setMessages(next)
  }, [])

  const setUIState = useCallback((messageIndex: number, key: string, formState: UIFormState) => {
    updateUI(messageIndex, key, (part, parts) => parts.map((p) => p === part ? { ...part, formState, error: undefined } : p))
  }, [updateUI])

  const reviewUI = useCallback((messageIndex: number, key: string, action: UIReviewAction) => {
    if (activeRef.current) return
    updateUI(messageIndex, key, (part, parts) => {
      const previous = parts.find((p) => p.kind === 'proposal' && p.key === part.reviewKey)
      if (previous?.kind === 'proposal' && previous.status !== 'rejected') return parts
      const result = uiReviewProposal(part.ui, action, draftRef.current, tempId, resolveEntryId)
      if ('error' in result) return parts.map((p) => p === part ? { ...part, error: result.error } : p)
      const reviewKey = nextProposalKey()
      const proposal = { ...result.proposal, sourceGuard: part.ui.target }
      return [...parts.map((p) => p === part ? { ...part, reviewKey, error: undefined } : p), { kind: 'proposal', key: reviewKey, proposal, status: 'pending' }]
    })
  }, [updateUI, resolveEntryId])

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
    setUIState,
    reviewUI,
  }
}

export type AgentChatController = ReturnType<typeof useAgentChat>
