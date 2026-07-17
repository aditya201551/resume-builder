import type { ReactNode } from 'react'
import { Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import AutosaveStatus from '@/components/editor/AutosaveStatus'
import type { SaveStatus } from '@/hooks/useAutosave'

interface RowCardProps {
  status: SaveStatus
  onDelete: () => void
  children: ReactNode
}

export default function RowCard({ status, onDelete, children }: RowCardProps) {
  return (
    <div className="rounded-md border border-border bg-card p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <AutosaveStatus status={status} />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-muted-foreground hover:text-destructive"
          aria-label="Delete"
          onClick={onDelete}
        >
          <Trash2 className="size-3.5" />
        </Button>
      </div>
      <div className="flex flex-col gap-3">{children}</div>
    </div>
  )
}
