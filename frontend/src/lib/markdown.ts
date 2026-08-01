import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Markdown } from 'tiptap-markdown'

/**
 * The preview and the editor must render the same markdown identically, so
 * this reuses the exact extension set RichTextEditor uses, just headless.
 *
 * Accepts null/undefined rather than requiring a string: draft entities can
 * legitimately reach the preview without a `content` key — an agent proposal
 * that omitted it, a localStorage draft written by an older schema — and a
 * missing content block should render as an empty section, never take the
 * whole editor down.
 */
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
