import { FileText } from 'lucide-react'
import { cn } from '@/lib/utils'

export default function Brand({ compact = false }: { compact?: boolean }) {
  return (
    <span className="inline-flex items-center gap-2.5 font-semibold tracking-tight">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-[13px] bg-primary text-primary-foreground">
        <FileText className="size-[19px]" aria-hidden />
      </span>
      <span className={cn('text-[15px]', compact && 'hidden sm:inline')}>
        Resume Builder
      </span>
    </span>
  )
}
