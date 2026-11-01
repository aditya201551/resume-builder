import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router'
import { FileUp, FileText, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { importResume, resumeFileError } from '@/lib/importResume'

export default function ImportResumeDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const [file, setFile] = useState<File | null>(null)
  const [fileError, setFileError] = useState<string | null>(null)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: importResume,
    retry: false,
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['resumes'] })
      onOpenChange(false)
      navigate(`/resumes/${result.resume.id}/edit`, { state: { mode: 'chat', importWarnings: result.warnings } })
    },
  })
  function changeOpen(next: boolean) {
    if (mutation.isPending) return
    if (!next) { setFile(null); setFileError(null); mutation.reset() }
    onOpenChange(next)
  }
  const error = fileError ?? mutation.error?.message
  return <Dialog open={open} onOpenChange={changeOpen}>
    <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto rounded-3xl p-6 sm:max-w-lg" showCloseButton={!mutation.isPending}
      onEscapeKeyDown={(event) => { if (mutation.isPending) event.preventDefault() }}
      onPointerDownOutside={(event) => { if (mutation.isPending) event.preventDefault() }}>
      <DialogHeader>
        <div className="mb-2 flex size-11 items-center justify-center rounded-2xl bg-secondary text-primary"><FileUp className="size-5" aria-hidden /></div>
        <DialogTitle>Import existing resume</DialogTitle>
        <DialogDescription>Bring your experience into a new, editable resume. Your content will use the Vertical template; the original design isn't copied.</DialogDescription>
      </DialogHeader>
      <label className="relative flex cursor-pointer flex-col items-center gap-3 rounded-2xl bg-secondary/30 px-5 py-7 text-center transition-colors hover:bg-secondary/60 focus-within:bg-secondary">
        <FileText className="size-7 text-primary" aria-hidden />
        <span className="max-w-full break-all text-sm font-medium">{file ? file.name : 'Choose your resume PDF'}</span>
        <span className="text-xs text-muted-foreground">PDF only · Up to 10 MB · 1–10 pages</span>
        <input type="file" accept="application/pdf,.pdf" aria-label="Resume PDF" className="sr-only" disabled={mutation.isPending} onChange={(event) => {
          const selected = event.target.files?.[0] ?? null
          mutation.reset()
          setFileError(selected ? resumeFileError(selected) : null)
          setFile(selected && !resumeFileError(selected) ? selected : null)
          event.target.value = ''
        }} />
      </label>
      <p className="text-xs leading-relaxed text-muted-foreground">We'll check that the PDF contains a resume or CV. It is sent to our AI provider for import. We save the extracted resume content, not the original file.</p>
      {error && <p role="alert" className="rounded-xl bg-destructive/10 p-3 text-sm text-destructive">{error}</p>}
      {mutation.isPending && <div role="status" className="flex items-start gap-2 rounded-xl bg-secondary p-3 text-sm"><Loader2 className="mt-0.5 size-4 shrink-0 animate-spin" aria-hidden /><span>Checking and importing your resume…<span className="mt-1 block text-xs text-muted-foreground">This can take about a minute. Keep this window open.</span></span></div>}
      <DialogFooter>
        <Button variant="ghost" disabled={mutation.isPending} onClick={() => changeOpen(false)}>Cancel</Button>
        <Button disabled={!file || mutation.isPending} onClick={() => { if (file) mutation.mutate(file) }}>
          {mutation.isPending ? <Loader2 className="size-4 animate-spin" /> : <FileUp className="size-4" />} {mutation.isPending ? 'Importing…' : 'Import resume'}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
}
