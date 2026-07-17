import { useNavigate } from 'react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, MoreVertical } from 'lucide-react'
import { apiDelete, apiGet, apiPost } from '@/lib/http'
import type { Resume, Template } from '@/types/resume'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import ResumeThumbnail from '@/components/dashboard/ResumeThumbnail'

const resumesKey = ['resumes'] as const

export default function DashboardPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const resumesQuery = useQuery({
    queryKey: resumesKey,
    queryFn: () => apiGet<Resume[]>('/api/resumes'),
  })

  const templatesQuery = useQuery({
    queryKey: ['templates'],
    queryFn: () => apiGet<Template[]>('/api/templates'),
  })

  const createMutation = useMutation({
    mutationFn: () =>
      apiPost<Resume>('/api/resumes', {
        label: 'Untitled resume',
        full_name: '',
        template_id: templatesQuery.data?.[0]?.id ?? null,
      }),
    onSuccess: (resume) => {
      queryClient.invalidateQueries({ queryKey: resumesKey })
      navigate(`/resumes/${resume.id}/edit`)
    },
  })

  const duplicateMutation = useMutation({
    mutationFn: (id: string) => apiPost<Resume>(`/api/resumes/${id}/duplicate`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: resumesKey }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => apiDelete(`/api/resumes/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: resumesKey }),
  })

  const resumes = resumesQuery.data ?? []

  return (
    <div className="px-10 py-10">
      <h1 className="text-3xl font-bold tracking-tight text-foreground">My Resumes</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        Create, tailor, and export resumes for each application.
      </p>

      {resumesQuery.isLoading ? (
        <p className="mt-8 text-sm text-muted-foreground">Loading…</p>
      ) : (
        <div className="mt-8 grid grid-cols-2 gap-6 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
          <button
            type="button"
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending}
            className="flex flex-col items-center justify-center gap-2 rounded-md border border-dashed border-border text-muted-foreground hover:border-ring hover:text-foreground"
            style={{ aspectRatio: '0.773' }}
          >
            <Plus className="size-6" />
            <span className="text-sm font-medium">New resume</span>
          </button>

          {resumes.map((resume) => (
            <div key={resume.id} className="flex flex-col gap-2">
              <div
                className="group relative cursor-pointer"
                onClick={() => navigate(`/resumes/${resume.id}/edit`)}
              >
                <ResumeThumbnail resumeId={resume.id} />
                <div className="absolute inset-0 rounded-md ring-1 ring-inset ring-transparent transition-colors group-hover:ring-ring" />
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="secondary"
                      size="icon"
                      className="absolute right-2 top-2 size-7 opacity-0 shadow-sm group-hover:opacity-100"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <MoreVertical className="size-3.5" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
                    <DropdownMenuItem onSelect={() => duplicateMutation.mutate(resume.id)}>
                      Duplicate
                    </DropdownMenuItem>
                    <DropdownMenuItem variant="destructive" onSelect={() => deleteMutation.mutate(resume.id)}>
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
              <p className="truncate text-sm font-medium text-foreground">{resume.label}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
