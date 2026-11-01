import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { ArrowDown, ArrowUpRight, Bookmark, BriefcaseBusiness, Check, Globe, RotateCcw, Send, SlidersHorizontal, Sparkles, Square } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Message, MessageContent } from '@/components/ui/message'
import ChatMarkdown from '@/components/agent/ChatMarkdown'
import CopyButton from '@/components/agent/CopyButton'
import type { JobAssistantController } from '@/hooks/useJobAssistant'

const starters = [
  { label: 'Find remote roles', icon: Globe, prompt: 'Find Go backend roles remote from India.' },
  { label: 'Explore hybrid work', icon: BriefcaseBusiness, prompt: 'Find hybrid product design roles in London, UK.' },
  { label: 'Search a new location', icon: SlidersHorizontal, prompt: 'Find senior frontend roles in Germany.' },
]

export default function JobChatPanel({ chat, active, saved, onManual }: { chat: JobAssistantController; active: boolean; saved: boolean; onManual: () => void }) {
  const [input, setInput] = useState('')
  const [showLatest, setShowLatest] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const stickToBottom = useRef(true)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    el.scrollTop = el.scrollHeight
    stickToBottom.current = true
    setShowLatest(false)
  }, [])

  useEffect(() => {
    if (active && stickToBottom.current) {
      if (chat.messages.length > 0 || chat.thinking) scrollToBottom()
      else if (scrollRef.current) scrollRef.current.scrollTop = 0
    }
  }, [active, chat.messages, chat.thinking, scrollToBottom])

  useEffect(() => {
    if (active && !chat.thinking && (document.activeElement === document.body || document.activeElement === textareaRef.current)) textareaRef.current?.focus()
  }, [active, chat.thinking])

  function send(text = input) {
    if (!text.trim() || chat.thinking) return
    setInput('')
    stickToBottom.current = true
    void chat.send(text)
  }
  function onKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) { event.preventDefault(); send() }
  }

  return <section className="assistant-panel jobs-chat-panel" aria-label="Job search assistant">
    <header className="assistant-header">
      <div className="jobs-chat-heading"><div><span className="jobs-chat-mark"><Sparkles size={16} aria-hidden /></span><h2>Job search assistant</h2></div><Button variant="ghost" size="icon-sm" aria-label="Clear job search conversation" disabled={chat.messages.length === 0 && !chat.thinking} onClick={() => { chat.clear(); setInput(''); textareaRef.current?.focus() }}><RotateCcw size={15} /></Button></div>
      <div className="assistant-scope"><span>{saved ? <Bookmark size={12} /> : <BriefcaseBusiness size={12} />}{saved ? 'Saved jobs' : 'Worldwide tech jobs'}</span><button type="button" onClick={onManual}><SlidersHorizontal size={12} />Edit filters</button></div>
    </header>
    <div className="jobs-chat-history-wrap">
      <div className="jobs-chat-history" ref={scrollRef} role="log" aria-label="Job search conversation" aria-live="polite" onScroll={() => { const el = scrollRef.current; if (!el) return; stickToBottom.current = el.scrollHeight - el.scrollTop - el.clientHeight < 96; setShowLatest(!stickToBottom.current) }}>
        <div className="jobs-chat-messages">
          {chat.messages.length === 0 && <div className="assistant-welcome"><h2>{saved ? 'Revisit the roles you saved.' : 'Let’s find your next role.'}</h2><p>{saved ? 'Ask me to narrow your saved jobs by role, work location, or experience level.' : 'Tell me what you want to do and where you can work. We’ll refine the search together.'}</p>{!saved && <div className="assistant-starters">{starters.map(({ label, icon: Icon, prompt }) => <button key={label} type="button" disabled={chat.thinking} onClick={() => send(prompt)}><Icon size={16} aria-hidden /><span>{label}</span><ArrowUpRight size={14} aria-hidden /></button>)}</div>}</div>}
          {chat.messages.map(message => <Message key={message.id} align={message.role === 'user' ? 'end' : 'start'}><MessageContent className={message.role === 'user' ? 'items-end' : 'max-w-full'}>
            {message.role === 'user' ? <div className="jobs-chat-user">{message.text}</div> : <div className="jobs-chat-answer"><ChatMarkdown text={message.text} animateWords /></div>}
            {message.applied && <span className="jobs-chat-applied"><Check size={12} />Search criteria updated</span>}
            <CopyButton getText={() => message.text} className={message.role === 'user' ? 'self-end' : 'self-start'} />
          </MessageContent></Message>)}
          {chat.thinking && <div className="jobs-chat-thinking" role="status"><Sparkles size={14} className="animate-pulse" aria-hidden />Refining your search…</div>}
        </div>
      </div>
      {showLatest && <Button className="jobs-chat-latest" size="sm" variant="secondary" onClick={scrollToBottom}><ArrowDown size={13} />Latest</Button>}
    </div>
    {chat.error && <div className="jobs-chat-error" role="alert"><p>{chat.error}</p><div><Button variant="ghost" size="sm" disabled={chat.thinking} onClick={() => void chat.retry()}>Retry request</Button><Button variant="ghost" size="sm" onClick={onManual}>Manual search</Button></div></div>}
    <form className="jobs-chat-composer" onSubmit={event => { event.preventDefault(); send() }}>
      <div className="assistant-composer"><Textarea ref={textareaRef} value={input} onChange={event => setInput(event.target.value)} onKeyDown={onKeyDown} maxLength={2000} rows={3} disabled={chat.thinking} aria-label="Message the job search assistant" placeholder={saved ? 'How would you like to filter your saved jobs?' : 'Describe your next role, or refine this search…'} /><div className="jobs-chat-composer-actions"><span>Enter to send · Shift+Enter for a new line</span>{chat.thinking ? <Button type="button" size="icon" variant="secondary" className="rounded-full" onClick={chat.cancel} aria-label="Stop job assistant response"><Square size={14} /></Button> : <Button type="submit" size="icon" className="rounded-full" disabled={!input.trim()} aria-label="Send job search message"><Send size={14} /></Button>}</div></div>
      <p className="jobs-chat-footnote">Search criteria stay visible in results. You can edit them in Manual search.</p>
    </form>
  </section>
}
