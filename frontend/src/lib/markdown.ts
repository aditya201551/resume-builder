import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Markdown } from 'tiptap-markdown'

/**
 * The preview and the editor must render the same markdown identically, so
 * this reuses the exact extension set RichTextEditor uses, just headless.
 */
export function markdownToHtml(markdown: string): string {
  if (!markdown.trim()) return ''
  const editor = new Editor({
    extensions: [
      StarterKit.configure({ heading: false, blockquote: false, codeBlock: false, horizontalRule: false }),
      Markdown.configure({ html: false }),
    ],
    content: markdown,
  })
  const html = editor.getHTML()
  editor.destroy()
  return html
}
