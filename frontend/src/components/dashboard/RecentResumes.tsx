import { Link } from 'react-router'
import { ArrowRight, FileUp, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import ResumeCard from './ResumeCard'
import type { Resume } from '@/types/resume'

interface RecentResumesProps {
  resumes: Resume[]
  isLoading: boolean
  isError: boolean
  onRetry: () => void
  onImport?: () => void
}

export default function RecentResumes({ resumes, isLoading, isError, onRetry, onImport }: RecentResumesProps) {
  if (!isLoading && !isError && resumes.length === 0) return null

  return (
    <section className="home-recent" aria-labelledby="recent-resumes-heading">
      <div className="home-recent-heading">
        <h2 id="recent-resumes-heading">Recent resumes</h2>
        <div className="home-recent-actions">
          <Button asChild variant="ghost" size="sm">
            <Link to="/resumes/library">View all <ArrowRight className="size-3.5" aria-hidden /></Link>
          </Button>
          {onImport && <Button type="button" variant="outline" size="sm" onClick={onImport}>
            <FileUp className="size-3.5" aria-hidden /> Import existing resume
          </Button>}
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
      ) : (
        <div className="home-recent-list">
          {resumes.slice(0, 4).map((resume) => (
            <ResumeCard key={resume.id} resume={resume} />
          ))}
        </div>
      )}
    </section>
  )
}
