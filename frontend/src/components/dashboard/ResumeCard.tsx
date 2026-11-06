import { useRef, useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowUpRight, Copy, Download, Loader2, MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import { apiDelete, apiGet, apiPost } from '@/lib/http'
import { renameResume } from '@/lib/api'
import { clearDraft, loadDraft, updateDraftLabel } from '@/lib/resumeDraftStorage'
import { downloadPdfBlob } from '@/lib/pdf/download'
import { generateResumePdf } from '@/lib/pdf/session'
import { fullResumeKey } from '@/hooks/useResumeEditor'
import type { FullResume, Resume } from '@/types/resume'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardAction, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import ResumeThumbnail from './ResumeThumbnail'

export default function ResumeCard({ resume }: { resume: Resume }) {
  const queryClient = useQueryClient()
  const [renaming, setRenaming] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [label, setLabel] = useState(resume.label)
  const optionsRef = useRef<HTMLButtonElement>(null)
  const title = resume.label || 'Untitled resume'

  const rename = useMutation({
    mutationFn: (nextLabel: string) => renameResume(resume.id, nextLabel),
    onSuccess: (updated) => {
      updateDraftLabel(resume.id, updated.label)
      queryClient.setQueryData<Resume[]>(['resumes'], (resumes) =>
        resumes?.map((item) => item.id === resume.id ? updated : item),
      )
      queryClient.setQueryData<FullResume>(fullResumeKey(resume.id), (data) =>
        data ? { ...data, resume: { ...data.resume, label: updated.label } } : data,
      )
      setRenaming(false)
      void queryClient.invalidateQueries({ queryKey: ['resumes'] })
    },
  })
  const duplicate = useMutation({
    mutationFn: () => apiPost<Resume>(`/api/resumes/${resume.id}/duplicate`),
    onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ['resumes'] }) },
  })
  const remove = useMutation({
    mutationFn: () => apiDelete(`/api/resumes/${resume.id}`),
    onSuccess: () => {
      clearDraft(resume.id)
      queryClient.removeQueries({ queryKey: fullResumeKey(resume.id) })
      setDeleting(false)
      void queryClient.invalidateQueries({ queryKey: ['resumes'] })
    },
  })
  const download = useMutation({
    mutationFn: async () => {
      const data = loadDraft(resume.id)?.data ?? await queryClient.fetchQuery<FullResume>({
        queryKey: fullResumeKey(resume.id),
        queryFn: () => apiGet<FullResume>(`/api/resumes/${resume.id}/full`),
        staleTime: 0,
      })
      if (!data) throw new Error('Resume is unavailable')
      downloadPdfBlob(await generateResumePdf(data), data.resume.label)
    },
  })
  const busy = rename.isPending || duplicate.isPending || remove.isPending || download.isPending

  function submitRename(event: FormEvent) {
    event.preventDefault()
    if (label.trim() && !rename.isPending) rename.mutate(label.trim())
  }

  function restoreFocus(event: Event) {
    event.preventDefault()
    if (optionsRef.current?.isConnected) optionsRef.current.focus()
    else document.getElementById('app-main')?.focus()
  }

  return (
    <article className="resume-card" aria-label={title}>
      <div className="resume-card-preview" aria-hidden inert>
        <div className="resume-miniature"><ResumeThumbnail resumeId={resume.id} /></div>
      </div>
      <Card className="resume-card-surface">
        <CardHeader className="resume-card-heading">
          <div className="min-w-0">
            <CardTitle className="resume-card-title" role="heading" aria-level={3} title={title}>{title}</CardTitle>
            {resume.full_name && <CardDescription className="resume-card-name">{resume.full_name}</CardDescription>}
            <CardDescription className="resume-card-date">
              Edited <time dateTime={resume.updated_at}>{new Date(resume.updated_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}</time>
            </CardDescription>
          </div>
          <CardAction>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button ref={optionsRef} variant="ghost" size="icon" className="resume-card-options" aria-label={`Options for ${title}`} disabled={busy}>
                  {busy ? <Loader2 className="size-4 animate-spin" aria-hidden /> : <MoreHorizontal className="size-4" aria-hidden />}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onSelect={() => {
                  rename.reset()
                  setLabel(resume.label)
                  setRenaming(true)
                }}>
                  <Pencil className="size-4" /> Rename
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => duplicate.mutate()}>
                  <Copy className="size-4" /> Duplicate
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => download.mutate()}>
                  <Download className="size-4" /> Download PDF
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem variant="destructive" onSelect={() => {
                  remove.reset()
                  setDeleting(true)
                }}>
                  <Trash2 className="size-4" /> Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </CardAction>
        </CardHeader>
        {download.isPending && <p className="resume-card-status" role="status">Generating PDF…</p>}
        {duplicate.isError && <p className="resume-card-status text-destructive" role="alert">Couldn’t duplicate this resume. Please try again.</p>}
        {download.isError && <p className="resume-card-status text-destructive" role="alert">Couldn’t generate the PDF. Please try again.</p>}
        <CardFooter className="resume-card-footer">
          <Button asChild className="resume-card-open">
            <Link to={`/resumes/${resume.id}/edit`} aria-label={`Open ${title}`}>
              Open resume <ArrowUpRight className="size-4" aria-hidden />
            </Link>
          </Button>
        </CardFooter>
      </Card>
      <Dialog open={renaming} onOpenChange={(open) => { if (!rename.isPending) setRenaming(open) }}>
        <DialogContent onCloseAutoFocus={restoreFocus}>
          <form onSubmit={submitRename} className="grid gap-4">
            <DialogHeader>
              <DialogTitle>Rename resume</DialogTitle>
              <DialogDescription>Change the name shown in your resume library.</DialogDescription>
            </DialogHeader>
            <label className="grid gap-2 text-sm">
              Resume name
              <Input value={label} onChange={(event) => setLabel(event.target.value)} disabled={rename.isPending} required autoFocus />
            </label>
            {rename.isError && <p className="text-sm text-destructive" role="alert">Couldn’t rename this resume. Please try again.</p>}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setRenaming(false)} disabled={rename.isPending}>Cancel</Button>
              <Button type="submit" disabled={!label.trim() || rename.isPending}>
                {rename.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />} Save name
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <AlertDialog open={deleting} onOpenChange={(open) => { if (!remove.isPending) setDeleting(open) }}>
        <AlertDialogContent onCloseAutoFocus={restoreFocus}>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {title}?</AlertDialogTitle>
            <AlertDialogDescription>This permanently removes the resume and all its sections.</AlertDialogDescription>
          </AlertDialogHeader>
          {remove.isError && <p className="text-sm text-destructive" role="alert">Couldn’t delete this resume. Please try again.</p>}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={remove.isPending}>Keep resume</AlertDialogCancel>
            <Button variant="destructive" disabled={remove.isPending} onClick={() => remove.mutate()}>
              {remove.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />} Delete resume
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </article>
  )
}
