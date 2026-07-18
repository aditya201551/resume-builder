import { useEffect } from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { Markdown } from 'tiptap-markdown'
import { Bold, Italic, List, ListOrdered } from 'lucide-react'
import { cn } from '@/lib/utils'

interface RichTextEditorProps {
  value: string
  onChange: (markdown: string) => void
  placeholder?: string
}

export default function RichTextEditor({ value, onChange, placeholder }: RichTextEditorProps) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: false,
        blockquote: false,
        codeBlock: false,
        horizontalRule: false,
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
          '[&_li]:my-0.5 [&_li]:pl-0.5',
      },
    },
    onUpdate: ({ editor }) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      onChange((editor.storage as any).markdown.getMarkdown())
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
    <div className="rounded-md border border-input bg-card">
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
      </div>
      <EditorContent editor={editor} className="px-3 py-2" placeholder={placeholder} />
    </div>
  )
}
