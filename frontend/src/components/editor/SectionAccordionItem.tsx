import type { ReactNode } from 'react'
import { AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'

interface SectionAccordionItemProps {
  value: string
  title: string
  count: number
  children: ReactNode
}

export default function SectionAccordionItem({ value, title, count, children }: SectionAccordionItemProps) {
  return (
    <AccordionItem value={value}>
      <AccordionTrigger className="text-sm font-medium">
        <span className="flex items-center gap-2">
          {title}
          {count > 0 && <Badge variant="secondary">{count}</Badge>}
        </span>
      </AccordionTrigger>
      <AccordionContent className="flex flex-col gap-3 pt-1">{children}</AccordionContent>
    </AccordionItem>
  )
}
