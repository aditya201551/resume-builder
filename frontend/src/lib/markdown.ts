import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Markdown } from 'tiptap-markdown'

export function markdownToHtml(markdown: string | null | undefined): string {
  if (!markdown || typeof markdown !== 'string' || !markdown.trim()) return ''
  const editor = new Editor({
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
    content: markdown,
  })
  const html = editor.getHTML()
  editor.destroy()
  return html
}
