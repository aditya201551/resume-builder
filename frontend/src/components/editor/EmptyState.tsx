import type { LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface EmptyStateProps {
  icon: LucideIcon
  message: string
  ctaLabel: string
  onClick: () => void
}

export default function EmptyState({ icon: Icon, message, ctaLabel, onClick }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-md border border-dashed border-border py-10 text-center">
      <Icon className="size-6 text-muted-foreground/60" />
      <p className="text-sm text-muted-foreground">{message}</p>
      <Button variant="secondary" size="sm" onClick={onClick}>
        {ctaLabel}
      </Button>
    </div>
  )
}
