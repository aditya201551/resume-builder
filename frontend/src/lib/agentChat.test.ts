import { afterEach, describe, expect, it, vi } from 'vitest'
import { streamAgentChat, type AgentChatHandlers } from './agentChat'

function streamFrom(chunks: string[]) {
  const encoder = new TextEncoder()
  return new ReadableStream<Uint8Array>({
    start(controller) {
      for (const chunk of chunks) controller.enqueue(encoder.encode(chunk))
      controller.close()
    },
  })
}

function mockFetchWithStream(chunks: string[]) {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(
      new Response(streamFrom(chunks), {
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
      }),
    ),
  )
}

function handlers() {
  return {
    onToken: vi.fn(),
    onProposal: vi.fn(),
    onTool: vi.fn(),
    onDone: vi.fn(),
    onError: vi.fn(),
  } satisfies AgentChatHandlers
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('streamAgentChat', () => {
  it('dispatches token chunks and the done event from the backend SSE shape', async () => {
    mockFetchWithStream([
      'event: token\ndata: {"text":"Here"}\n\n',
      'event: token\ndata: {"text":" is the work section"}\n\n',
      'event: done\ndata: {}\n\n',
    ])
    const h = handlers()

    await streamAgentChat('resume-1', { messages: [{ role: 'user', content: 'work?' }], draft: {} }, h)

    expect(h.onToken).toHaveBeenNthCalledWith(1, 'Here')
    expect(h.onToken).toHaveBeenNthCalledWith(2, ' is the work section')
    expect(h.onDone).toHaveBeenCalledTimes(1)
    expect(h.onError).not.toHaveBeenCalled()
  })

  it('dispatches tool activity events', async () => {
    mockFetchWithStream([
      'event: tool\ndata: {"id":"call-1","name":"read_resume","status":"running"}\n\n',
      'event: tool\ndata: {"id":"call-1","name":"read_resume","status":"done"}\n\n',
      'event: done\ndata: {}\n\n',
    ])
    const h = handlers()

    await streamAgentChat('resume-1', { messages: [{ role: 'user', content: 'read it' }], draft: {} }, h)

    expect(h.onTool).toHaveBeenNthCalledWith(1, { id: 'call-1', name: 'read_resume', status: 'running' })
    expect(h.onTool).toHaveBeenNthCalledWith(2, { id: 'call-1', name: 'read_resume', status: 'done' })
    expect(h.onDone).toHaveBeenCalledTimes(1)
    expect(h.onError).not.toHaveBeenCalled()
  })

  it('handles CRLF separators and a trailing done event without a final blank line', async () => {
    mockFetchWithStream(['event: token\r\ndata: {"text":"ok"}\r\n\r\n', 'event: done\r\ndata: {}'])
    const h = handlers()

    await streamAgentChat('resume-1', { messages: [{ role: 'user', content: 'ping' }], draft: {} }, h)

    expect(h.onToken).toHaveBeenCalledWith('ok')
    expect(h.onDone).toHaveBeenCalledTimes(1)
    expect(h.onError).not.toHaveBeenCalled()
  })
})
