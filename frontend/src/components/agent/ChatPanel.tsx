import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { ArrowDown, Check, FileText, Loader2, Send, SpellCheck, Sparkles, Square, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Message, MessageContent } from '@/components/ui/message'
import ActionItem from '@/components/agent/ActionItem'
import ChatMarkdown from '@/components/agent/ChatMarkdown'
import CopyButton from '@/components/agent/CopyButton'
import StreamingMarkdown from '@/components/agent/StreamingMarkdown'
import { groupAssistantParts } from '@/lib/agentProposal'
import { useAgentChat, type AssistantPart, type ChatMessage } from '@/hooks/useAgentChat'

function isNearBottom(el: HTMLDivElement) {
  return el.scrollHeight - el.scrollTop - el.clientHeight < 96
}

function TypingIndicator() {
  return (
    <div className="flex w-fit items-center gap-1.5 rounded-full bg-secondary px-3 py-2 text-muted-foreground">
      <span className="sr-only">Assistant is thinking</span>
      {[0, 1, 2].map((i) => (
        <span
          key={i}
          className="size-1.5 rounded-full bg-current opacity-40 [animation:chat-dot_1.05s_ease-in-out_infinite]"
          style={{ animationDelay: `${i * 140}ms` }}
          aria-hidden
        />
      ))}
    </div>
  )
}

/**
 * One-click entry points for the two PRD-listed Phase 2 asks that had no
 * dedicated UI before this — "AI-generated professional summary from
 * existing Work Experience/Skills data" and "Grammar/style check pass on
 * resume text." The general-purpose chat agent can already do both from a
 * plain-language ask (it has propose_meta_update and propose_update), so
 * this is only a guided shortcut into the same flow, not a new tool or
 * endpoint — clicking one just sends the canned prompt as a normal message.
 */
const QUICK_ACTIONS = [
  {
    label: 'Write my summary',
    icon: FileText,
    prompt:
      'Generate a professional summary for my resume based on my current work experience and skills, and propose it as a contact-info update.',
  },
  {
    label: 'Check grammar & style',
    icon: SpellCheck,
    prompt:
      'Review all my resume content (work experience, project, and summary text) for grammar and style issues, and propose fixes for anything you find.',
  },
] as const

const TOOL_LABELS: Record<string, string> = {
  read_resume: 'Reading resume',
  propose_create: 'Drafting an addition',
  propose_update: 'Drafting an update',
  propose_delete: 'Drafting a removal',
  propose_skill_change: 'Updating skills',
  propose_custom_section_change: 'Updating custom sections',
  propose_meta_update: 'Updating contact info',
}

function ToolActivity({ event }: { event: Extract<AssistantPart, { kind: 'tool' }>['event'] }) {
  const label = TOOL_LABELS[event.name] ?? event.name.replace(/_/g, ' ')
  const isRunning = event.status === 'running'
  const isError = event.status === 'error'

  return (
    <div className="animate-in fade-in slide-in-from-bottom-1 flex w-fit items-center gap-2 rounded-full border border-border bg-secondary/50 px-2.5 py-1.5 text-xs text-muted-foreground duration-200">
      {isRunning ? (
        <Loader2 className="size-3.5 animate-spin text-accent" />
      ) : isError ? (
        <X className="size-3.5 text-destructive" />
      ) : (
        <Check className="size-3.5 text-emerald-600 dark:text-emerald-400" />
      )}
      <span>{label}</span>
      {isRunning && <span className="size-1 rounded-full bg-accent/70 [animation:chat-dot_1.05s_ease-in-out_infinite]" aria-hidden />}
      {!isRunning && !isError && <span className="text-muted-foreground/70">done</span>}
      {isError && <span className="text-destructive">{event.error || 'failed'}</span>}
    </div>
  )
}

function AssistantParts({
  parts,
  streaming = false,
  onResolve,
}: {
  parts: AssistantPart[]
  streaming?: boolean
  onResolve?: (key: string, accept: boolean) => void
}) {
  const items = groupAssistantParts(parts)

  return (
    <div className="flex flex-col gap-2">
      {items.map((item, i) => {
        // Text gets its own part every time a tool call interrupts it (a
        // reply that reads "...creating this: [tool] ...then this: [tool]"
        // produces several separate text items). Only the very last item in
        // the message is still growing — every earlier text segment is
        // already finished, so only the last one should carry the blinking
        // cursor. Without this check every finished segment kept its own
        // cursor animating forever.
        const isLast = i === items.length - 1
        return item.kind === 'text' ? (
          <div key={item.key} className="text-sm leading-6 text-foreground">
            {streaming && isLast ? <StreamingMarkdown text={item.text} streaming /> : <ChatMarkdown text={item.text} />}
          </div>
        ) : item.kind === 'tool' ? (
          <ToolActivity key={item.key} event={item.event} />
        ) : (
          <ActionItem
            key={item.key}
            item={item}
            onAccept={() => onResolve?.(item.part.key, true)}
            onReject={() => onResolve?.(item.part.key, false)}
          />
        )
      })}
    </div>
  )
}

/** The assistant's natural-language reply, ignoring tool/proposal parts —
 * what "copy this message" should put on the clipboard, not the structured
 * actions alongside it. */
function assistantText(parts: AssistantPart[]) {
  return parts.filter((p) => p.kind === 'text').map((p) => p.text).join('')
}

function MessageRow({
  message,
  index,
  onResolve,
}: {
  message: ChatMessage
  index: number
  onResolve: (index: number, key: string, accept: boolean) => void
}) {
  if (message.role === 'user') {
    return (
      <Message align="end" className="animate-in fade-in slide-in-from-bottom-1 duration-200">
        <MessageContent>
          <div className="max-w-[85%] self-end rounded-2xl bg-primary px-3.5 py-2 text-sm whitespace-pre-wrap text-primary-foreground">
            {message.content}
          </div>
          <CopyButton className="self-end" getText={() => message.content} />
        </MessageContent>
      </Message>
    )
  }

  const hasText = message.parts.some((p) => p.kind === 'text' && p.text.trim())

  return (
    <Message align="start" className="animate-in fade-in slide-in-from-bottom-1 duration-200">
      <MessageContent className="max-w-[90%]">
        <AssistantParts parts={message.parts} onResolve={(key, accept) => onResolve(index, key, accept)} />
        {hasText && <CopyButton getText={() => assistantText(message.parts)} />}
      </MessageContent>
    </Message>
  )
}

function StreamingMessage({ parts }: { parts: AssistantPart[] }) {
  return (
    <Message align="start" className="animate-in fade-in slide-in-from-bottom-1 duration-200">
      <MessageContent className="max-w-[90%]">
        {parts.length === 0 ? <TypingIndicator /> : <AssistantParts parts={parts} streaming />}
      </MessageContent>
    </Message>
  )
}

export default function ChatPanel() {
  const { messages, streamingParts, isStreaming, autoAccept, setAutoAccept, error, sendMessage, cancel, resolveProposal } =
    useAgentChat()
  const [input, setInput] = useState('')
  const [showJumpToLatest, setShowJumpToLatest] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const shouldStickRef = useRef(true)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'smooth') => {
    const el = scrollRef.current
    if (!el) return
    el.scrollTo({ top: el.scrollHeight, behavior })
    shouldStickRef.current = true
    setShowJumpToLatest(false)
  }, [])

  useEffect(() => {
    if (shouldStickRef.current) scrollToBottom('auto')
  }, [messages, streamingParts, scrollToBottom])

  // Sending disables the textarea (see `disabled={isStreaming}` below), and a
  // disabled element loses focus to document.body — the browser does this
  // for us, we never call blur() ourselves. Once streaming ends and the
  // textarea is re-enabled, nothing gives focus back automatically, so the
  // user has to click back in before they can keep typing. Restore it
  // ourselves — but only when nothing else has since taken focus on
  // purpose (a resume field the user clicked into while the assistant was
  // still responding, an accept/reject button, etc.); document.body is what
  // activeElement is left holding after the disable-triggered blur, so that
  // check is what distinguishes "nothing else grabbed focus" from "the user
  // moved on deliberately."
  useEffect(() => {
    if (isStreaming) return
    const active = document.activeElement
    if (active === document.body || active === textareaRef.current) {
      textareaRef.current?.focus()
    }
  }, [isStreaming])

  function handleScroll() {
    const el = scrollRef.current
    if (!el) return
    const pinned = isNearBottom(el)
    shouldStickRef.current = pinned
    setShowJumpToLatest(!pinned && (messages.length > 0 || Boolean(streamingParts)))
  }

  function handleSend() {
    const content = input.trim()
    if (!content) return
    setInput('')
    shouldStickRef.current = true
    void sendMessage(content)
  }

  function runQuickAction(prompt: string) {
    if (isStreaming) return
    shouldStickRef.current = true
    void sendMessage(prompt)
  }

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className="flex h-full min-h-0 w-full flex-col bg-background">
      <div className="relative min-h-0 flex-1">
        <div ref={scrollRef} onScroll={handleScroll} className="h-full overflow-y-auto scroll-smooth">
          <div className="mx-auto flex w-full max-w-2xl flex-col gap-6 px-4 py-6">
            {messages.length === 0 && !streamingParts && (
              <div className="flex flex-col items-center gap-4 py-16 text-center">
                <Sparkles className="size-8 text-muted-foreground" />
                <p className="max-w-sm text-sm text-muted-foreground">
                  Ask me to add, rewrite, or reorganize anything on your resume. I&apos;ll propose changes for you to review.
                </p>
                <div className="flex flex-wrap justify-center gap-2">
                  {QUICK_ACTIONS.map(({ label, icon: Icon, prompt }) => (
                    <button
                      key={label}
                      type="button"
                      onClick={() => runQuickAction(prompt)}
                      className="flex items-center gap-1.5 rounded-full border border-border bg-secondary/50 px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-secondary"
                    >
                      <Icon className="size-3.5 text-muted-foreground" />
                      {label}
                    </button>
                  ))}
                </div>
              </div>
            )}
            {messages.map((m, i) => (
              <MessageRow key={i} message={m} index={i} onResolve={resolveProposal} />
            ))}
            {streamingParts && <StreamingMessage parts={streamingParts} />}
          </div>
        </div>

        {showJumpToLatest && (
          <Button
            type="button"
            size="sm"
            variant="secondary"
            className="absolute bottom-3 left-1/2 h-7 -translate-x-1/2 gap-1.5 rounded-full border border-border bg-background/95 px-3 shadow-sm backdrop-blur"
            onClick={() => scrollToBottom()}
          >
            <ArrowDown className="size-3.5" />
            Latest
          </Button>
        )}
      </div>

      {error && (
        <div className="mx-auto w-full max-w-2xl px-4">
          <div className="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">{error}</div>
        </div>
      )}

      <div className="mx-auto w-full max-w-2xl p-3 pt-2">
        <div className="rounded-xl border border-input bg-background px-2.5 py-1.5 transition-colors focus-within:border-ring">
          <Textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Message the assistant..."
            rows={1}
            className="min-h-7 resize-none border-0 bg-transparent px-0 py-0.5 text-sm shadow-none focus-visible:ring-0"
            disabled={isStreaming}
          />
          <div className="flex items-center justify-between gap-3 pt-0.5">
            <button
              type="button"
              className="flex items-center gap-1.5 rounded-full px-0.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:text-foreground"
              onClick={() => setAutoAccept(!autoAccept)}
              aria-pressed={autoAccept}
            >
              <span
                className="relative inline-flex h-3 w-5 shrink-0 rounded-full bg-input transition-colors data-[checked=true]:bg-primary"
                data-checked={autoAccept}
                aria-hidden
              >
                <span
                  className="absolute left-0.5 top-0.5 size-2 rounded-full bg-background transition-transform data-[checked=true]:translate-x-2"
                  data-checked={autoAccept}
                />
              </span>
              <span>Auto-accept edits</span>
            </button>

            {isStreaming ? (
              <Button type="button" size="icon-xs" variant="secondary" onClick={cancel} aria-label="Stop response">
                <Square className="size-3.5" />
              </Button>
            ) : (
              <Button type="button" size="icon-xs" onClick={handleSend} disabled={!input.trim()} aria-label="Send message">
                <Send className="size-3.5" />
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
