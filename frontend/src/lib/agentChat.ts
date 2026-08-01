/**
 * Thin SSE client for POST /api/resumes/{resumeId}/agent/chat — a custom
 * event protocol (token/proposal/done/error), not OpenAI's stream shape.
 * Uses fetch()+ReadableStream rather than EventSource, since EventSource
 * can't POST a body (see backend/internal/api/handlers/agent_handler.go for
 * the server side of this contract).
 */

export interface AgentChatMessage {
  role: 'user' | 'assistant'
  content: string
}

// Mirrors backend/internal/agent/proposal.go's Proposal struct. Fields are
// optional because which ones are populated depends on `type`.
export interface AgentProposal {
  type: string
  entity?: string
  id?: string
  tempId?: string
  groupId?: string
  sectionId?: string
  sectionType?: string
  customSectionId?: string | null
  fields?: Record<string, unknown>
  patch?: Record<string, unknown>
  // Id of the propose_* tool call that produced this proposal. Used to pair
  // a proposal with its "tool" event in the UI — the two arrive as separate
  // SSE events, and when several propose_* calls run concurrently their
  // "done"/proposal events can interleave in either order, so stream
  // position alone can't be trusted to pair them.
  toolCallId?: string
}

export interface AgentToolEvent {
  id?: string
  name: string
  status: 'running' | 'done' | 'error'
  arguments?: string
  error?: string
}

export interface AgentChatHandlers {
  onToken: (text: string) => void
  onProposal: (proposal: AgentProposal) => void
  onTool: (event: AgentToolEvent) => void
  onDone: () => void
  onError: (message: string) => void
}

export async function streamAgentChat(
  resumeId: string,
  body: { messages: AgentChatMessage[]; draft: unknown },
  handlers: AgentChatHandlers,
  signal?: AbortSignal,
): Promise<void> {
  let res: Response
  try {
    res = await fetch(`/api/resumes/${resumeId}/agent/chat`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal,
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') return
    handlers.onError(err instanceof Error ? err.message : 'request failed')
    return
  }

  if (!res.ok || !res.body) {
    const text = await res.text().catch(() => '')
    handlers.onError(text || `request failed: ${res.status}`)
    return
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })

      let sep: EventSeparator | null
      while ((sep = findEventSeparator(buffer))) {
        const { index: sepIndex, length: sepLength } = sep
        const rawEvent = buffer.slice(0, sepIndex)
        buffer = buffer.slice(sepIndex + sepLength)
        dispatchEvent(rawEvent, handlers)
      }
    }
    buffer += decoder.decode()
    if (buffer.trim()) dispatchEvent(buffer, handlers)
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') return
    handlers.onError(err instanceof Error ? err.message : 'stream read failed')
  }
}

type EventSeparator = { index: number; length: number }

function findEventSeparator(buffer: string): EventSeparator | null {
  const lf = buffer.indexOf('\n\n')
  const crlf = buffer.indexOf('\r\n\r\n')

  if (lf === -1 && crlf === -1) return null
  if (lf === -1) return { index: crlf, length: 4 }
  if (crlf === -1) return { index: lf, length: 2 }
  return lf < crlf ? { index: lf, length: 2 } : { index: crlf, length: 4 }
}

function dispatchEvent(raw: string, handlers: AgentChatHandlers) {
  let event = ''
  let data = ''
  for (const line of raw.replace(/\r\n/g, '\n').split('\n')) {
    if (line.startsWith('event:')) event = line.slice('event:'.length).trim()
    else if (line.startsWith('data:')) data += line.slice('data:'.length).trim()
  }
  if (!event) return
  if (!data && event === 'done') data = '{}'
  if (!data) return

  let parsed: unknown
  try {
    parsed = JSON.parse(data)
  } catch {
    return
  }

  switch (event) {
    case 'token':
      handlers.onToken((parsed as { text: string }).text)
      break
    case 'proposal':
      handlers.onProposal((parsed as { action: AgentProposal }).action)
      break
    case 'tool':
      handlers.onTool(parsed as AgentToolEvent)
      break
    case 'done':
      handlers.onDone()
      break
    case 'error':
      handlers.onError((parsed as { message: string }).message)
      break
  }
}
