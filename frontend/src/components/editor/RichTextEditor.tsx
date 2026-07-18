import { useEffect, useRef, useState } from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { Markdown } from 'tiptap-markdown'
import { Bold, Check, Italic, Link as LinkIcon, List, ListOrdered, Unlink } from 'lucide-react'
import { cn } from '@/lib/utils'

interface RichTextEditorProps {
  value: string
  onChange: (markdown: string) => void
  placeholder?: string
}

export default function RichTextEditor({ value, onChange, placeholder }: RichTextEditorProps) {
  const [linkPopoverOpen, setLinkPopoverOpen] = useState(false)
  const [linkUrl, setLinkUrl] = useState('')
  // Position the popover was last dismissed at — lets a user close it (Done/
  // Remove/Escape) without the cursor still sitting inside the link mark
  // immediately reopening it, while still reopening if they click away and
  // back into the same link later.
  const dismissedAtRef = useRef<number | null>(null)

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: false,
        blockquote: false,
        codeBlock: false,
        horizontalRule: false,
        link: {
          openOnClick: false,
          autolink: true,
          HTMLAttributes: { rel: 'noopener noreferrer', target: '_blank' },
        },
      }),
      Markdown.configure({ html: false }),
    ],
    content: value,
    editorProps: {
      attributes: {
        class:
          'max-w-none focus:outline-none min-h-20 text-sm text-foreground ' +
          '[&_p]:my-1 ' +
          '[&_ul]:my-1 [&_ul]:list-disc [&_ul]:pl-5 ' +
          '[&_ol]:my-1 [&_ol]:list-decimal [&_ol]:pl-5 ' +
          '[&_li]:my-0.5 [&_li]:pl-0.5 ' +
          '[&_a]:text-accent [&_a]:underline',
      },
      handleDOMEvents: {
        // Links are real <a href> elements inside the contenteditable, so
        // the browser will still navigate on click unless we stop it —
        // openOnClick:false only disables tiptap's own open-on-cmd-click.
        // event.target can be a Text node (rendered text has no element of
        // its own), which has no .closest — walk up to an Element first or
        // the lookup throws and preventDefault() never runs.
        click: (_view, event) => {
          const target = event.target
          const el = target instanceof Element ? target : (target as Node | null)?.parentElement
          if (el?.closest('a')) {
            event.preventDefault()
            event.stopPropagation()
          }
          return false
        },
      },
    },
    onUpdate: ({ editor }) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      onChange((editor.storage as any).markdown.getMarkdown())
    },
    onSelectionUpdate: ({ editor }) => {
      const { href } = editor.getAttributes('link') as { href?: string }
      if (!href) {
        dismissedAtRef.current = null
        setLinkPopoverOpen(false)
        return
      }
      if (dismissedAtRef.current === editor.state.selection.from) {
        setLinkPopoverOpen(false)
        return
      }
      setLinkUrl(href)
      setLinkPopoverOpen(true)
    },
  })

  useEffect(() => {
    if (!editor) return
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const current = (editor.storage as any).markdown.getMarkdown()
    if (current !== value) {
      editor.commands.setContent(value, { emitUpdate: false })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value])

  if (!editor) return null

  function openLinkPopover() {
    if (!editor) return
    const { href } = editor.getAttributes('link') as { href?: string }
    if (!href && editor.state.selection.empty) return
    setLinkUrl(href ?? '')
    setLinkPopoverOpen(true)
  }

  function dismissPopover() {
    dismissedAtRef.current = editor?.state.selection.from ?? null
    setLinkPopoverOpen(false)
  }

  function confirmLink() {
    if (!editor) return
    const trimmed = linkUrl.trim()
    if (trimmed) {
      editor.chain().focus().extendMarkRange('link').setLink({ href: trimmed }).run()
    } else {
      editor.chain().focus().extendMarkRange('link').unsetLink().run()
    }
    dismissPopover()
  }

  function removeLink() {
    if (!editor) return
    editor.chain().focus().extendMarkRange('link').unsetLink().run()
    dismissPopover()
  }

  function cancelLink() {
    dismissPopover()
    editor?.commands.focus()
  }

  const button = (
    active: boolean,
    onClick: () => void,
    Icon: typeof Bold,
    label: string,
  ) => (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className={cn(
        'flex size-6 items-center justify-center rounded text-muted-foreground hover:bg-secondary hover:text-foreground',
        active && 'bg-secondary text-foreground',
      )}
    >
      <Icon className="size-3.5" />
    </button>
  )

  return (
    <div className="relative rounded-md border border-input bg-card">
      <div className="flex items-center gap-1 border-b border-input px-2 py-1">
        {button(editor.isActive('bold'), () => editor.chain().focus().toggleBold().run(), Bold, 'Bold')}
        {button(editor.isActive('italic'), () => editor.chain().focus().toggleItalic().run(), Italic, 'Italic')}
        {button(
          editor.isActive('bulletList'),
          () => editor.chain().focus().toggleBulletList().run(),
          List,
          'Bullet list',
        )}
        {button(
          editor.isActive('orderedList'),
          () => editor.chain().focus().toggleOrderedList().run(),
          ListOrdered,
          'Numbered list',
        )}
        {button(editor.isActive('link'), openLinkPopover, LinkIcon, 'Link')}
      </div>

      {linkPopoverOpen && (
        <div className="absolute left-2 top-9 z-10 flex items-center gap-1 rounded-md border border-input bg-popover p-1 shadow-md">
          <LinkIcon className="ml-1 size-3.5 shrink-0 text-muted-foreground" />
          <input
            autoFocus
            value={linkUrl}
            onChange={(e) => setLinkUrl(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                confirmLink()
              }
              if (e.key === 'Escape') {
                e.preventDefault()
                cancelLink()
              }
            }}
            placeholder="https://…"
            className="h-7 w-56 bg-transparent px-1 text-sm outline-none"
          />
          <button
            type="button"
            aria-label="Apply link"
            onClick={confirmLink}
            className="flex size-6 shrink-0 items-center justify-center rounded text-accent hover:bg-secondary"
          >
            <Check className="size-3.5" />
          </button>
          {editor.isActive('link') && (
            <button
              type="button"
              aria-label="Remove link"
              onClick={removeLink}
              className="flex size-6 shrink-0 items-center justify-center rounded text-muted-foreground hover:bg-secondary hover:text-destructive"
            >
              <Unlink className="size-3.5" />
            </button>
          )}
        </div>
      )}

      <EditorContent editor={editor} className="px-3 py-2" placeholder={placeholder} />
    </div>
  )
}
