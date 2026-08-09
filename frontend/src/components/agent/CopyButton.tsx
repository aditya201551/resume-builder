import { useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { cn } from '@/lib/utils'

export default function CopyButton({ getText, className }: { getText: () => string; className?: string }) {
  const [copied, setCopied] = useState(false)

  async function handleCopy() {
    const text = getText()
    if (!text.trim()) return
    try {
      await navigator.clipboard.writeText(text)
    } catch {
      return
    }
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1500)
  }

  return (
    <button
      type="button"
      onClick={handleCopy}
      className={cn(
        'inline-flex w-fit items-center gap-1 rounded px-1 py-0.5 text-[11px] text-muted-foreground transition-colors hover:text-foreground',
        className,
      )}
      aria-label={copied ? 'Copied' : 'Copy message'}
    >
      {copied ? <Check className="size-3" /> : <Copy className="size-3" />}
      {copied ? 'Copied' : 'Copy'}
    </button>
  )
}
