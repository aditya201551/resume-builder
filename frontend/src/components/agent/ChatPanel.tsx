import {
  useCallback,
  useEffect,
  useRef,
  useState,
  lazy,
  Suspense,
  type KeyboardEvent,
} from 'react'
import {
  ArrowDown,
  ArrowUp,
  ArrowUpRight,
  Check,
  FileText,
  Loader2,
  Palette,
  Plus,
  SpellCheck,
  Square,
  X,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Message, MessageContent } from '@/components/ui/message'
import ActionItem from '@/components/agent/ActionItem'
import ChatMarkdown from '@/components/agent/ChatMarkdown'
import CopyButton from '@/components/agent/CopyButton'
import { groupAssistantParts } from '@/lib/agentProposal'
import {
  type AgentChatController,
  type AssistantPart,
  type ChatMessage,
} from '@/hooks/useAgentChat'
import type { UIFormState, UIReviewAction } from '@/lib/assistantUI'

const AssistantUIMessage = lazy(() => import('./openui/AssistantUIMessage'))

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

const QUICK_ACTIONS = [
  {
    label: 'Write my summary',
    icon: FileText,
    prompt:
      'Write a professional summary grounded in my current experience and skills and propose the update directly. Ask only if a missing fact or ambiguity prevents the change.',
  },
  {
    label: 'Find a layout that fits',
    icon: Palette,
    prompt:
      'Review my resume and recommend a suitable available template and layout. Explain your choice and propose design changes while preserving all my content.',
  },
  {
    label: 'Check grammar & style',
    icon: SpellCheck,
    prompt:
      'Review all my resume content (work experience, project, and summary text) for grammar and style issues, and propose fixes for anything you find.',
  },
  {
    label: 'Add work experience',
    icon: Plus,
    prompt: 'Help me add a new work experience. Use details I have already supplied and ask only for missing required facts or ambiguities, then propose the entry directly.',
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
  show_resume_ui: 'Preparing an interactive card',
}

function ToolActivity({
  event,
}: {
  event: Extract<AssistantPart, { kind: 'tool' }>['event']
}) {
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
        <Check className="size-3.5 text-emerald-600 " />
      )}
      <span>{label}</span>
      {isRunning && (
        <span
          className="size-1 rounded-full bg-accent/70 [animation:chat-dot_1.05s_ease-in-out_infinite]"
          aria-hidden
        />
      )}
      {!isRunning && !isError && (
        <span className="text-muted-foreground/70">done</span>
      )}
      {isError && (
        <span className="text-destructive">{event.error || 'failed'}</span>
      )}
    </div>
  )
}

function AssistantParts({
  parts,
  streaming = false,
  onResolve,
  busy = false,
  onUIState,
  onReviewUI,
}: {
  parts: AssistantPart[]
  streaming?: boolean
  onResolve?: (
    key: string,
    accept: boolean,
    edits?: Record<string, string>,
  ) => void
  busy?: boolean
  onUIState?: (key: string, state: UIFormState) => void
  onReviewUI?: (key: string, action: UIReviewAction) => void
}) {
  const items = groupAssistantParts(parts)

  return (
    <div className="flex flex-col gap-2">
      {items.map((item, i) => {
        const isLast = i === items.length - 1
        const review = item.kind === 'ui' ? parts.find((p) => p.kind === 'proposal' && p.key === item.part.reviewKey) : undefined
        return item.kind === 'text' ? (
          <div key={item.key} className="text-sm leading-6 text-foreground">
            <ChatMarkdown text={item.text} animateWords={streaming} streaming={streaming && isLast} />
          </div>
        ) : item.kind === 'tool' ? (
          <ToolActivity key={item.key} event={item.event} />
        ) : item.kind === 'ui' ? (
          <Suspense key={item.key} fallback={<div role="status" className="rounded-2xl bg-secondary p-4 text-xs text-muted-foreground">Opening interactive card...</div>}>
            <AssistantUIMessage ui={item.part.ui} formState={item.part.formState} disabled={streaming || busy || !onReviewUI}
              reviewStatus={review?.kind === 'proposal' ? review.status : undefined}
              error={item.part.error} onStateChange={(state) => onUIState?.(item.key, state)} onReview={(action) => onReviewUI?.(item.key, action)} />
          </Suspense>
        ) : (
          <ActionItem
            key={item.key}
            item={item}
            canResolve={Boolean(onResolve)}
            onAccept={(edits) => onResolve?.(item.part.key, true, edits)}
            onReject={() => onResolve?.(item.part.key, false)}
          />
        )
      })}
    </div>
  )
}

function assistantText(parts: AssistantPart[]) {
  return parts
    .filter((p) => p.kind === 'text')
    .map((p) => p.text)
    .join('')
}

function MessageRow({
  message,
  index,
  onResolve,
  busy,
  onUIState,
  onReviewUI,
}: {
  message: ChatMessage
  index: number
  onResolve: (
    index: number,
    key: string,
    accept: boolean,
    edits?: Record<string, string>,
  ) => void
  busy: boolean
  onUIState: AgentChatController['setUIState']
  onReviewUI: AgentChatController['reviewUI']
}) {
  if (message.role === 'user') {
    return (
      <Message
        align="end"
        className="animate-in fade-in slide-in-from-bottom-1 duration-200"
      >
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
    <Message
      align="start"
      className="animate-in fade-in slide-in-from-bottom-1 duration-200"
    >
      <MessageContent className="max-w-full">
        <AssistantParts
          parts={message.parts}
          busy={busy}
          onUIState={(key, state) => onUIState(index, key, state)}
          onReviewUI={(key, action) => onReviewUI(index, key, action)}
          onResolve={(key, accept, edits) =>
            onResolve(index, key, accept, edits)
          }
        />
        {hasText && <CopyButton getText={() => assistantText(message.parts)} />}
      </MessageContent>
    </Message>
  )
}

function StreamingMessage({ parts }: { parts: AssistantPart[] }) {
  return (
    <Message
      align="start"
      className="animate-in fade-in slide-in-from-bottom-1 duration-200"
    >
      <MessageContent className="max-w-full">
        {parts.length === 0 ? (
          <TypingIndicator />
        ) : (
          <AssistantParts parts={parts} streaming />
        )}
      </MessageContent>
    </Message>
  )
}

export default function ChatPanel({
  chat,
  jobDescription,
  onJobDescriptionChange,
}: {
  chat: AgentChatController
  jobDescription: string
  onJobDescriptionChange: (value: string) => void
}) {
  const {
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
  } = chat
  const [input, setInput] = useState('')
  const [jobOpen, setJobOpen] = useState(false)
  const [showJumpToLatest, setShowJumpToLatest] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const shouldStickRef = useRef(true)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const contextButtonRef = useRef<HTMLButtonElement>(null)

  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'smooth') => {
    const el = scrollRef.current
    if (!el) return
    el.scrollTo({
      top: el.scrollHeight,
      behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches
        ? 'instant'
        : behavior,
    })
    shouldStickRef.current = true
    setShowJumpToLatest(false)
  }, [])

  useEffect(() => {
    if ((messages.length > 0 || streamingParts) && shouldStickRef.current)
      scrollToBottom('auto')
  }, [messages, streamingParts, scrollToBottom])

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
    setShowJumpToLatest(
      !pinned && (messages.length > 0 || Boolean(streamingParts)),
    )
  }

  function handleSend() {
    const content = input.trim()
    if (!content || isStreaming) return
    setInput('')
    shouldStickRef.current = true
    void sendMessage(content, jobDescription)
  }

  function runQuickAction(prompt: string) {
    if (isStreaming) return
    shouldStickRef.current = true
    void sendMessage(prompt, jobDescription)
  }

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <section className="assistant-panel resume-assistant-panel flex h-full min-h-0 w-full flex-col" aria-label="Resume assistant">
      <span role="status" className="sr-only">
        {isStreaming ? 'Working on it...' : 'Here to help'}
      </span>
      <div className="relative min-h-0 flex-1">
        <div
          ref={scrollRef}
          onScroll={handleScroll}
          className="h-full overflow-y-auto scroll-smooth"
        >
          <div className="assistant-conversation mx-auto flex w-full max-w-2xl flex-col gap-6 px-5 py-6">
            {messages.length === 0 && !streamingParts && (
              <div className="assistant-welcome">
                <h2>
                  Work on your resume.
                </h2>
                <p>
                  Rewrite an entry, add a section, or find a layout that fits.
                  You’ll review the changes.
                </p>
                <div className="assistant-starters">
                  {QUICK_ACTIONS.map(({ label, icon: Icon, prompt }) => (
                    <button
                      key={label}
                      type="button"
                      onClick={() => runQuickAction(prompt)}
                    >
                      <Icon className="size-4 text-primary" aria-hidden />
                      <span className="flex-1">{label}</span>
                      <ArrowUpRight
                        className="size-3.5 text-muted-foreground"
                        aria-hidden
                      />
                    </button>
                  ))}
                </div>
              </div>
            )}
            {messages.map((m, i) => (
              <MessageRow
                key={i}
                message={m}
                index={i}
                onResolve={resolveProposal}
                busy={isStreaming}
                onUIState={(index, key, state) => {
                  shouldStickRef.current = false
                  setUIState(index, key, state)
                }}
                onReviewUI={(index, key, action) => {
                  shouldStickRef.current = true
                  reviewUI(index, key, action)
                }}
              />
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
          <div
            role="alert"
            className="flex flex-wrap items-center gap-2 rounded-xl bg-destructive/10 px-3 py-2 text-xs text-destructive"
          >
            <span className="flex-1">{error}</span>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                const last = messages.findLast(
                  (message) => message.role === 'user',
                )
                if (last?.role === 'user') setInput(last.content)
                textareaRef.current?.focus()
              }}
            >
              Edit and retry
            </Button>
          </div>
        </div>
      )}

      <form className="resume-chat-compose-wrap" onSubmit={(event) => { event.preventDefault(); handleSend() }}>
        <div className="assistant-composer resume-chat-composer">
          <div className="resume-composer-fields">
            {jobOpen && (
              <div id="assistant-job-context" className="resume-job-context">
                <div className="resume-job-context-heading">
                  <label htmlFor="assistant-job">Job description</label>
                  <button
                    type="button"
                    aria-label="Remove job context"
                    onClick={() => {
                      onJobDescriptionChange('')
                      setJobOpen(false)
                      contextButtonRef.current?.focus()
                    }}
                  >
                    <X className="size-3.5" aria-hidden />
                  </button>
                </div>
                <Textarea
                  id="assistant-job"
                  autoFocus
                  value={jobDescription}
                  maxLength={16000}
                  onChange={(event) => onJobDescriptionChange(event.target.value)}
                  rows={3}
                  placeholder="Paste a job description to guide your next request."
                />
              </div>
            )}
            <Textarea
              ref={textareaRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="What would you like to work on?"
              aria-label="Message the resume assistant"
              rows={1}
              className="resume-chat-input"
              disabled={isStreaming}
            />
          </div>
          <div className="resume-composer-toolbar">
            <div className="resume-composer-options">
              <button
                type="button"
                ref={contextButtonRef}
                className="resume-context-button"
                data-filled={Boolean(jobDescription.trim())}
                aria-label={jobDescription.trim() ? 'Edit job context' : 'Add job context'}
                onClick={() => setJobOpen(!jobOpen)}
                aria-expanded={jobOpen}
                aria-controls="assistant-job-context"
              >
                {jobDescription.trim() ? <Check className="size-3.5" aria-hidden /> : <Plus className="size-3.5" aria-hidden />}
                Job context
              </button>
              <button
                type="button"
                className="resume-auto-accept"
                onClick={() => setAutoAccept(!autoAccept)}
                aria-pressed={autoAccept}
                aria-label="Auto-accept edits"
                title="Automatically apply other edits. Complete new entries are added directly."
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
                <span>Auto-accept</span>
              </button>
            </div>

            {isStreaming ? (
              <Button
                type="button"
                size="icon"
                className="resume-chat-send"
                variant="secondary"
                onClick={cancel}
                aria-label="Stop response"
              >
                <Square className="size-3.5" />
              </Button>
            ) : (
              <Button
                type="submit"
                size="icon"
                className="resume-chat-send"
                disabled={!input.trim()}
                aria-label="Send message"
              >
                <ArrowUp className="size-4" aria-hidden />
              </Button>
            )}
          </div>
        </div>
      </form>
    </section>
  )
}
