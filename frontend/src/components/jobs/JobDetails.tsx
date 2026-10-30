import { Bookmark, ExternalLink, MapPin, Monitor, X } from 'lucide-react'
import { Dialog as DialogPrimitive } from 'radix-ui'
import { Button } from '@/components/ui/button'
import { useJob, useSaveJob } from '@/hooks/useJobs'
import { employmentLabels, jobDate, locationLabel, safeJobUrl, timezoneLabel, workplaceLabels } from '@/lib/jobs'
import { toast } from 'sonner'

export default function JobDetails({ id, onClose }: { id: string | null; onClose: () => void }) {
  const query = useJob(id)
  const save = useSaveJob()
  const job = query.data
  const apply = safeJobUrl(job?.applyUrl)
  const source = safeJobUrl(job?.sourceUrl)
  return <DialogPrimitive.Root open={Boolean(id)} onOpenChange={open => { if (!open) onClose() }}>
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="jobs-detail-overlay" />
      <DialogPrimitive.Content className="jobs-detail" aria-describedby="job-detail-summary">
        <div className="jobs-detail-top"><span>Job details</span><Button variant="ghost" size="icon" aria-label="Close job details" onClick={onClose}><X size={18} /></Button></div>
        {query.isPending && <><DialogPrimitive.Title className="sr-only">Loading job</DialogPrimitive.Title><DialogPrimitive.Description id="job-detail-summary" className="jobs-detail-state" role="status">Loading the original listing…</DialogPrimitive.Description></>}
        {query.isError && <><DialogPrimitive.Title>Could not load this job</DialogPrimitive.Title><DialogPrimitive.Description id="job-detail-summary">Try again, or return to the results.</DialogPrimitive.Description><Button variant="outline" onClick={() => void query.refetch()}>Retry</Button></>}
        {job && <>
          <div className="jobs-detail-heading"><p>{job.company}</p><DialogPrimitive.Title>{job.title}</DialogPrimitive.Title><DialogPrimitive.Description id="job-detail-summary"><MapPin size={15} />{locationLabel(job)}</DialogPrimitive.Description></div>
          {job.closed && <p className="jobs-notice" role="status">This listing is closed or no longer recently confirmed. It stays here because you saved it.</p>}
          <dl className="jobs-detail-facts">
            <div><dt>Workplace</dt><dd>{workplaceLabels[job.workplace]}</dd></div>
            <div><dt>Employment</dt><dd>{employmentLabels[job.employmentType] ?? job.employmentType}</dd></div>
            {job.salary && <div><dt>Salary as listed</dt><dd>{job.salary}</dd></div>}
            <div><dt>Listing date</dt><dd>{jobDate(job.postedAt)}</dd></div>
            <div><dt>Last fetched</dt><dd>{jobDate(job.fetchedAt)}</dd></div>
          </dl>
          {job.workplace === 'remote' && <div className="jobs-remote-policy"><Monitor size={18} /><div><h3>Remote hiring locations</h3><p>{job.remote.scope === 'worldwide' ? 'The source lists worldwide hiring.' : job.remote.scope === 'restricted' ? job.remote.regions.join(', ') : 'The source does not specify accepted hiring regions.'}</p>{job.remote.timezones.length > 0 && <p>{job.remote.timezones.length <= 12 ? `Listed timezone offsets: ${job.remote.timezones.map(timezoneLabel).join(', ')}.` : 'Timezone information is supplied with this listing.'} Check the original posting for working hours.</p>}</div></div>}
          {job.locations.some(l => l.precision === 'city') && <p className="jobs-detail-note">The map uses approximate city centres from GeoNames. These are not verified office addresses.</p>}
          <div className="jobs-detail-description"><h3>About the role</h3><p>{job.description || 'The source did not include a description. Read the original posting for details.'}</p></div>
          <p className="jobs-detail-source">Source: {source ? <a href={source} target="_blank" rel="noopener noreferrer">{job.sourceName}<ExternalLink size={12} /></a> : job.sourceName}</p>
          <div className="jobs-detail-actions">
            {apply && !job.closed && <Button asChild><a href={apply} target="_blank" rel="noopener noreferrer">Apply on source website<ExternalLink size={15} /></a></Button>}
            <Button variant="outline" disabled={save.isPending} aria-pressed={job.saved} onClick={() => save.mutate({ id: job.id, saved: !job.saved }, { onError: () => toast.error('Could not update your saved jobs. Try again.') })}><Bookmark size={15} fill={job.saved ? 'currentColor' : 'none'} />{job.saved ? 'Saved' : 'Save job'}</Button>
          </div>
        </>}
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  </DialogPrimitive.Root>
}
