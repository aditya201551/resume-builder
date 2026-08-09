import type { ReactNode } from 'react'
import {
  DndContext,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
  arrayMove,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical } from 'lucide-react'
import { cn } from '@/lib/utils'

interface SortableListProps<T extends { id: string }> {
  items: T[]
  onReorder: (orderedIds: string[]) => void
  renderItem: (item: T, index: number, dragHandle: ReactNode) => ReactNode
  className?: string
  dragHandlePlacement?: 'gutter' | 'inline'
}

export default function SortableList<T extends { id: string }>({
  items,
  onReorder,
  renderItem,
  className,
  dragHandlePlacement = 'gutter',
}: SortableListProps<T>) {
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 4 } }))

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const oldIndex = items.findIndex((i) => i.id === active.id)
    const newIndex = items.findIndex((i) => i.id === over.id)
    if (oldIndex === -1 || newIndex === -1) return
    onReorder(arrayMove(items, oldIndex, newIndex).map((i) => i.id))
  }

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
      <SortableContext items={items.map((i) => i.id)} strategy={verticalListSortingStrategy}>
        <div className={cn('flex flex-col gap-3', className)}>
          {items.map((item, index) => (
            <SortableRow key={item.id} id={item.id} placement={dragHandlePlacement}>
              {(dragHandle) => renderItem(item, index, dragHandle)}
            </SortableRow>
          ))}
        </div>
      </SortableContext>
    </DndContext>
  )
}

function SortableRow({
  id,
  placement,
  children,
}: {
  id: string
  placement: 'gutter' | 'inline'
  children: (dragHandle: ReactNode) => ReactNode
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging, isOver } = useSortable({ id })

  const dragHandle = (
    <button
      type="button"
      aria-label="Drag to reorder"
      onClick={(e) => e.stopPropagation()}
      className={cn(
        'flex shrink-0 cursor-grab items-center justify-center rounded text-muted-foreground/50 hover:bg-muted hover:text-muted-foreground active:cursor-grabbing',
        placement === 'gutter' ? 'mt-2 size-5' : '-my-1 size-6',
      )}
      {...attributes}
      {...listeners}
    >
      <GripVertical className="size-4" />
    </button>
  )

  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn(
        'relative',
        placement === 'gutter' && 'flex items-start gap-2',
        isDragging && (placement === 'gutter' ? 'opacity-50' : 'z-10 opacity-60 shadow-lg'),
      )}
    >
      {isOver && (
        <span className="absolute -top-2 left-0 right-0 flex items-center">
          <span className="size-1.5 shrink-0 rounded-full bg-accent" />
          <span className="h-0.5 flex-1 rounded-full bg-accent" />
        </span>
      )}
      {placement === 'gutter' ? (
        <>
          {dragHandle}
          <div className="min-w-0 flex-1">{children(dragHandle)}</div>
        </>
      ) : (
        children(dragHandle)
      )}
    </div>
  )
}
