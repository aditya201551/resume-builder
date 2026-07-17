import type { LucideIcon } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

export interface AddSectionOption {
  key: string
  title: string
  description: string
  icon: LucideIcon
  emphasized?: boolean
}

interface AddSectionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  options: AddSectionOption[]
  onSelect: (key: string) => void
}

export default function AddSectionDialog({ open, onOpenChange, options, onSelect }: AddSectionDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-full max-w-[95vw] gap-6 p-6 sm:max-w-3xl sm:p-8 lg:max-w-4xl">
        <DialogHeader>
          <DialogTitle className="text-2xl font-bold">Add content</DialogTitle>
          <DialogDescription className="sr-only">Choose a section to add to your resume</DialogDescription>
        </DialogHeader>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {options.map(({ key, title, description, icon: Icon, emphasized }) => (
            <button
              key={key}
              type="button"
              onClick={() => {
                onSelect(key)
                onOpenChange(false)
              }}
              className={cn(
                'flex flex-col items-start gap-2 rounded-lg border border-transparent bg-secondary/60 p-5 text-left transition-colors hover:bg-secondary',
                emphasized && 'border-dashed border-border bg-transparent hover:bg-secondary/40',
              )}
            >
              <span className="flex size-9 items-center justify-center rounded-md bg-background text-foreground">
                <Icon className="size-4" />
              </span>
              <span className="font-semibold text-foreground">{title}</span>
              <span className="text-sm leading-relaxed text-muted-foreground">{description}</span>
            </button>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  )
}
