import type { ReactNode } from 'react'
import { AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'

interface SectionAccordionItemProps {
  value: string
  title: string
  count: number
  dragHandle?: ReactNode
  children: ReactNode
}

export default function SectionAccordionItem({ value, title, count, dragHandle, children }: SectionAccordionItemProps) {
  return (
    <AccordionItem value={value} className="overflow-hidden rounded-xl border border-border bg-card shadow-sm">
      <div className="flex items-center gap-1 pl-2 pr-4">
        {dragHandle}
        <AccordionTrigger className="cursor-pointer items-center gap-3 rounded-lg py-4 pl-1 pr-3 text-base font-medium hover:no-underline">
          <span className="flex flex-1 items-center gap-3">
            <span>{title}</span>
            {count > 0 && (
              <Badge variant="secondary" className="ml-0.5">
                {count}
              </Badge>
            )}
          </span>
        </AccordionTrigger>
      </div>
      <AccordionContent className="flex flex-col gap-4 px-4 pt-1 pb-5">{children}</AccordionContent>
    </AccordionItem>
  )
}
