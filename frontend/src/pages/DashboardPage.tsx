import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowRight,
  ArrowUp,
  Check,
  ChevronDown,
  Copy,
  FileText,
  FileUp,
  Loader2,
  MoreHorizontal,
  Plus,
  Search,
  Sparkles,
  Target,
  Trash2,
  WandSparkles,
  X,
} from 'lucide-react'
import { apiDelete, apiGet, apiPost } from '@/lib/http'
import { ASSISTANT_STARTERS, type AssistantTask } from '@/lib/assistantTask'
import { useAuth } from '@/hooks/useAuth'
import { useMediaQuery } from '@/hooks/useMediaQuery'
import type { Resume } from '@/types/resume'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
} from '@/components/ui/alert-dialog'
import ResumeThumbnail from '@/components/dashboard/ResumeThumbnail'
import ImportResumeDialog from '@/components/dashboard/ImportResumeDialog'
import RecentResumes from '@/components/dashboard/RecentResumes'
import PromptExamples from '@/components/dashboard/PromptExamples'

const resumesKey = ['resumes'] as const
const starterIcons = [WandSparkles, Check, Target]

export default function DashboardPage({
  libraryOnly = false,
}: {
  libraryOnly?: boolean
}) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { user } = useAuth()
  const mobile = useMediaQuery('(max-width: 767px)')
  const [prompt, setPrompt] = useState('')
  const [selection, setSelection] = useState('')
  const [jobOpen, setJobOpen] = useState(false)
  const [jobDescription, setJobDescription] = useState('')
  const [search, setSearch] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<Resume | null>(null)
  const [importOpen, setImportOpen] = useState(false)
  const [examplesOpen, setExamplesOpen] = useState(false)
  const resumesQuery = useQuery({
    queryKey: resumesKey,
    queryFn: () => apiGet<Resume[]>('/api/resumes'),
  })
  const resumes = [...(resumesQuery.data ?? [])].sort((a, b) =>
    b.updated_at.localeCompare(a.updated_at),
  )
  const selectedId =
    resumes.some((resume) => resume.id === selection) || selection === 'new'
      ? selection
      : (resumes[0]?.id ?? 'new')
  const selectedLabel = selectedId === 'new'
    ? 'New resume'
    : resumes.find((resume) => resume.id === selectedId)?.label || 'Untitled resume'

  const createMutation = useMutation({
    mutationFn: (_task: AssistantTask | undefined) =>
      apiPost<Resume>('/api/resumes', {
        label: 'Untitled resume',
        full_name: '',
      }),
    onSuccess: (resume, task) => {
      void queryClient.invalidateQueries({ queryKey: resumesKey })
      navigate(`/resumes/${resume.id}/edit`, {
        state: task ? { assistantTask: task } : { mode: 'content' },
      })
    },
  })
  const duplicateMutation = useMutation({
    mutationFn: (id: string) => apiPost<Resume>(`/api/resumes/${id}/duplicate`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: resumesKey })
    },
  })
  const deleteMutation = useMutation({
    mutationFn: (id: string) => apiDelete(`/api/resumes/${id}`),
    onSuccess: () => {
      setDeleteTarget(null)
      void queryClient.invalidateQueries({ queryKey: resumesKey })
    },
  })

  function startAssistant(event: FormEvent) {
    event.preventDefault()
    if (
      !prompt.trim() ||
      createMutation.isPending ||
      resumesQuery.isLoading ||
      resumesQuery.isError
    )
      return
    const task: AssistantTask = {
      prompt: prompt.trim(),
      jobDescription: jobDescription.trim(),
    }
    if (selectedId === 'new') createMutation.mutate(task)
    else
      navigate(`/resumes/${selectedId}/edit`, {
        state: { assistantTask: task },
      })
  }
  const shownResumes = resumes.filter((resume) =>
    `${resume.label} ${resume.full_name}`
      .toLowerCase()
      .includes(search.toLowerCase()),
  )
  const error =
    createMutation.isError ||
    duplicateMutation.isError ||
    deleteMutation.isError

  return (
    <div className={`dashboard-page${libraryOnly ? '' : ' dashboard-home'}`}>
      {!libraryOnly && (
        <section className="home-intro" aria-labelledby="home-heading">
          <h1 id="home-heading">
            What are we working toward?
          </h1>
          <p>
            {user?.name ? `${user.name.split(' ')[0]}, let's` : "Let's"} make your next move.
          </p>
          <form className="home-composer" onSubmit={startAssistant}>
            <div className="home-prompt-field">
              <Textarea
                id="home-prompt"
                aria-label="Message the resume assistant"
                value={prompt}
                onChange={(event) => setPrompt(event.target.value)}
                maxLength={8000}
                rows={1}
                className="home-prompt"
              />
              {!prompt && <PromptExamples />}
            </div>
            {jobOpen && (
              <div className="job-context-field">
                <div className="flex items-center justify-between gap-2">
                  <label htmlFor="home-job" className="text-xs font-medium">
                    Job description
                  </label>
                  <button
                    type="button"
                    aria-label="Remove job description"
                    onClick={() => {
                      setJobOpen(false)
                      setJobDescription('')
                    }}
                    className="rounded-full p-1 hover:bg-secondary"
                  >
                    <X className="size-3.5" />
                  </button>
                </div>
                <Textarea
                  id="home-job"
                  value={jobDescription}
                  onChange={(event) => setJobDescription(event.target.value)}
                  maxLength={16000}
                  rows={4}
                  placeholder="Paste the role you have in mind. We'll use it as context."
                />
              </div>
            )}
            <div className="composer-toolbar">
              <div className="composer-context-actions">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button
                      type="button"
                      className="resume-context-picker"
                      aria-label={`Resume to work on: ${selectedLabel}`}
                      title={selectedLabel}
                      disabled={resumesQuery.isLoading}
                    >
                      <FileText className="size-3.5 shrink-0" aria-hidden />
                      <span>{selectedLabel}</span>
                      <ChevronDown className="size-3 shrink-0" aria-hidden />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent className="resume-context-menu" align="start">
                    <DropdownMenuRadioGroup value={selectedId} onValueChange={setSelection} aria-label="Resume to work on">
                      <DropdownMenuRadioItem value="new">New resume</DropdownMenuRadioItem>
                      {resumes.map((resume) => (
                        <DropdownMenuRadioItem key={resume.id} value={resume.id}>
                          {resume.label || 'Untitled resume'}
                        </DropdownMenuRadioItem>
                      ))}
                    </DropdownMenuRadioGroup>
                  </DropdownMenuContent>
                </DropdownMenu>
                <button
                  type="button"
                  className="context-chip"
                  onClick={() => setJobOpen(!jobOpen)}
                  aria-expanded={jobOpen}
                  aria-controls="home-job"
                  aria-label={jobDescription.trim() ? 'Edit job description' : 'Add job description'}
                >
                  {jobDescription.trim() ? (
                    <Check className="size-3.5" />
                  ) : (
                    <Plus className="size-3.5" />
                  )}{' '}
                  <span className="home-context-full">{jobDescription.trim()
                    ? 'Job context added'
                    : 'Job description'}</span>
                  <span className="home-context-short">Job context</span>
                </button>
              </div>
              <Button
                type="submit"
                size="icon"
                className="start-ai-button"
                aria-label="Send to AI assistant"
                disabled={
                  !prompt.trim() ||
                  createMutation.isPending ||
                  resumesQuery.isLoading ||
                  resumesQuery.isError
                }
              >
                {createMutation.isPending ? (
                  <Loader2 className="size-5 animate-spin" aria-hidden />
                ) : (
                  <ArrowUp className="size-5" aria-hidden />
                )}
              </Button>
            </div>
          </form>
          <button
            type="button"
            className="home-examples-toggle"
            aria-expanded={examplesOpen}
            aria-controls="home-starter-actions"
            onClick={() => setExamplesOpen(!examplesOpen)}
          >
            Try an example <ChevronDown className="size-4" aria-hidden />
          </button>
          <div id="home-starter-actions" className={`starter-grid${examplesOpen ? ' is-expanded' : ''}`}>
            {ASSISTANT_STARTERS.map((starter, index) => {
              const Icon = starterIcons[index]
              return (
                <button
                  type="button"
                  key={starter.title}
                  className="starter-button"
                  onClick={() => {
                    setPrompt(starter.prompt)
                    setExamplesOpen(false)
                    if (index === 2) setJobOpen(true)
                    document.getElementById('home-prompt')?.focus()
                  }}
                >
                  <span className="starter-icon">
                    <Icon className="size-[18px]" aria-hidden />
                  </span>
                  <span>
                    <strong>{starter.title}</strong>
                    <small>{starter.description}</small>
                  </span>
                  <ArrowRight
                    className="ml-auto size-3.5 shrink-0 text-muted-foreground"
                    aria-hidden
                  />
                </button>
              )
            })}
          </div>
          <div className="manual-start">
            <button
              type="button"
              onClick={() => createMutation.mutate(undefined)}
              disabled={createMutation.isPending}
            >
              <Plus className="size-4" aria-hidden /> Create manually
            </button>
            <button type="button" className="home-mobile-import" onClick={() => setImportOpen(true)}>
              <FileUp className="size-4" aria-hidden /> Import resume
            </button>
          </div>
          {error && (
            <div role="alert" className="home-notice">
              <p>That change couldn’t be completed. Please try again.</p>
            </div>
          )}
          {mobile && resumesQuery.isError && (
            <div role="alert" className="home-notice">
              <p>We couldn’t load your resumes.</p>
              <Button type="button" variant="ghost" size="sm" onClick={() => void resumesQuery.refetch()}>Try again</Button>
            </div>
          )}
        </section>
      )}

      {!libraryOnly && !mobile && (
        <RecentResumes
          resumes={resumes}
          isLoading={resumesQuery.isLoading}
          isError={resumesQuery.isError}
          onRetry={() => void resumesQuery.refetch()}
          onImport={() => setImportOpen(true)}
        />
      )}

      {libraryOnly && <section className="resume-library" aria-labelledby="library-heading">
        <div className="library-heading">
          <div>
            <h1 id="library-heading" className="library-title">My resumes</h1>
              <p className="mt-2 text-sm text-muted-foreground">
                A version for every opportunity. Your original is always yours.
              </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" onClick={() => setImportOpen(true)}>
              <FileUp className="size-4" /> <span>Import<span className="library-import-extra"> existing</span> resume</span>
            </Button>
            <Button
              variant="outline"
              onClick={() => createMutation.mutate(undefined)}
              disabled={createMutation.isPending}
            >
              <Plus className="size-4" /> New resume
            </Button>
          </div>
        </div>
          <div className="library-search">
            <Search className="size-4 text-muted-foreground" aria-hidden />
            <Input
              aria-label="Search resumes"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Find a resume…"
            />
          </div>
        {error && (
          <p role="alert" className="mb-4 text-sm text-destructive">
            That change couldn’t be completed. Please try again.
          </p>
        )}
        {resumesQuery.isLoading ? (
          <div className="library-loading" role="status">
            <Loader2 className="size-5 animate-spin" /> Finding your resumes…
          </div>
        ) : resumesQuery.isError ? (
          <div className="library-empty" role="alert">
            <p>We couldn’t load your resumes.</p>
            <Button
              variant="outline"
              onClick={() => void resumesQuery.refetch()}
            >
              Try again
            </Button>
          </div>
        ) : shownResumes.length === 0 ? (
          <div className="library-empty">
            <span className="empty-document">
              <FileText className="size-6" />
            </span>
            <h2>
              {search
                ? 'No matching resumes'
                : 'Your first resume starts here.'}
            </h2>
            <p>
              {search
                ? 'Try another name.'
                : 'Import an existing resume or create a blank one.'}
            </p>
          </div>
        ) : (
          <div className="resume-grid">
            {shownResumes.map((resume) => (
              <article key={resume.id} className="resume-card">
                <Link
                  to={`/resumes/${resume.id}/edit`}
                  className="resume-card-preview"
                  aria-label={`Open ${resume.label || 'Untitled resume'}`}
                >
                  <div className="resume-miniature">
                    <ResumeThumbnail resumeId={resume.id} />
                  </div>
                  <span className="open-resume-hint">
                    Open workspace <ArrowRight className="size-3.5" />
                  </span>
                </Link>
                <div className="resume-card-details">
                  <div className="min-w-0">
                    <Link
                      to={`/resumes/${resume.id}/edit`}
                      className="block truncate text-sm font-semibold"
                    >
                      {resume.label || 'Untitled resume'}
                    </Link>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {resume.full_name || 'Your next opportunity'}
                    </p>
                    <p className="mt-3 text-[11px] text-muted-foreground">
                      Updated{' '}
                      {new Date(resume.updated_at).toLocaleDateString(
                        undefined,
                        { month: 'short', day: 'numeric' },
                      )}
                    </p>
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Options for ${resume.label || 'Untitled resume'}`}
                      >
                        <MoreHorizontal className="size-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                      align="end"
                      className="rounded-2xl p-1.5"
                    >
                      <DropdownMenuItem
                        disabled={duplicateMutation.isPending}
                        onSelect={() => duplicateMutation.mutate(resume.id)}
                      >
                        <Copy className="size-4" /> Duplicate
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        variant="destructive"
                        onSelect={() => {
                          deleteMutation.reset()
                          setDeleteTarget(resume)
                        }}
                      >
                        <Trash2 className="size-4" /> Delete
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>}
      {libraryOnly && <footer className="dashboard-footer">
        <span className="inline-flex items-center gap-1.5">
          <Sparkles className="size-3" aria-hidden /> Your experience. Your
          words. You’re in control.
        </span>
      </footer>}
      <ImportResumeDialog open={importOpen} onOpenChange={setImportOpen} />
      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open && !deleteMutation.isPending) setDeleteTarget(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete {deleteTarget?.label || 'this resume'}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              This permanently removes the resume and all its sections.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {deleteMutation.isError && (
            <p role="alert" className="text-sm text-destructive">
              Couldn’t delete this resume. Please try again.
            </p>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>
              Keep resume
            </AlertDialogCancel>
            <Button
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => {
                if (deleteTarget) deleteMutation.mutate(deleteTarget.id)
              }}
            >
              {deleteMutation.isPending && (
                <Loader2 className="size-4 animate-spin" />
              )}{' '}
              Delete resume
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
