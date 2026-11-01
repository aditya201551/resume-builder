import { Link } from 'react-router'
import { ArrowRight, FileUp, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import ResumeThumbnail from '@/components/dashboard/ResumeThumbnail'
import type { Resume } from '@/types/resume'

interface RecentResumesProps {
  resumes: Resume[]
  isLoading: boolean
  isError: boolean
  onRetry: () => void
  onImport: () => void
}

export default function RecentResumes({ resumes, isLoading, isError, onRetry, onImport }: RecentResumesProps) {
  return (
    <section className="home-recent" aria-labelledby="recent-resumes-heading">
      <div className="home-recent-heading">
        <h2 id="recent-resumes-heading">Recent resumes</h2>
        <div className="home-recent-actions">
          <Button asChild variant="ghost" size="sm">
            <Link to="/resumes/library">View all <ArrowRight className="size-3.5" aria-hidden /></Link>
          </Button>
          <Button type="button" variant="outline" size="sm" onClick={onImport}>
            <FileUp className="size-3.5" aria-hidden /> Import existing resume
          </Button>
        </div>
      </div>
      {isLoading ? (
        <div className="home-recent-state" role="status">
          <Loader2 className="size-4 animate-spin" aria-hidden /> Loading your resumes…
        </div>
      ) : isError ? (
        <div className="home-recent-state" role="alert">
          <p>We couldn’t load your resumes.</p>
          <Button type="button" size="sm" variant="ghost" onClick={onRetry}>Try again</Button>
        </div>
      ) : resumes.length === 0 ? (
        <p className="home-recent-state">Create or import a resume to get started.</p>
      ) : (
        <div className="home-recent-list">
          {resumes.slice(0, 4).map((resume) => (
            <Link
              key={resume.id}
              to={`/resumes/${resume.id}/edit`}
              className="home-recent-card"
              aria-label={`Open ${resume.label || 'Untitled resume'}`}
            >
              <div className="home-recent-cover" aria-hidden>
                <div className="home-recent-preview"><ResumeThumbnail resumeId={resume.id} /></div>
              </div>
              <div className="home-recent-details">
                <h3 title={resume.label || 'Untitled resume'}>{resume.label || 'Untitled resume'}</h3>
                <p title={resume.full_name || 'Your next opportunity'}>{resume.full_name || 'Your next opportunity'}</p>
                <div className="home-recent-footer">
                  <time dateTime={resume.updated_at}>
                    Edited {new Date(resume.updated_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
                  </time>
                  <span className="home-recent-open">Continue editing <ArrowRight aria-hidden /></span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </section>
  )
}
