import { useMemo, useState } from 'react'
import { NavLink, useSearchParams } from 'react-router'
import { ArrowDown, Bookmark, BriefcaseBusiness, Copy, ExternalLink, Loader2, Search, SlidersHorizontal, Sparkles } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import JobDetails from '@/components/jobs/JobDetails'
import JobChatPanel from '@/components/jobs/JobChatPanel'
import JobSearchFilters from '@/components/jobs/JobSearchFilters'
import { useJobCountries, useSaveJob } from '@/hooks/useJobs'
import { useJobDiscovery } from '@/hooks/useJobDiscovery'
import { useJobAssistant } from '@/hooks/useJobAssistant'
import { defaultJobIntent, readJobIntent, uniqueDiscoveryJobs, type SearchIntent } from '@/lib/jobSearch'
import { employmentLabels, jobDate, jobDateTime, jobListingDate, locationLabel, safeJobUrl, workplaceLabels } from '@/lib/jobs'
import type { Job } from '@/types/job'
import '@/styles/jobs.css'

export default function JobsPage({ saved = false }: { saved?: boolean }) {
 const [params, setParams] = useSearchParams()
 const rawIntent = params.get('intent')
 const intent = useMemo(() => readJobIntent(rawIntent), [rawIntent])
 const mode = params.get('mode') === 'manual' || (!params.has('mode') && saved) ? 'manual' : 'assistant'
 const [started, setStarted] = useState(Boolean(rawIntent))
 const [mobileResults, setMobileResults] = useState(false)
 const countries = useJobCountries()
 const results = useJobDiscovery(intent, saved, started || saved || Boolean(rawIntent))
 const save = useSaveJob()
 const jobs = uniqueDiscoveryJobs(results.data?.pages ?? [])
 const lastPage = results.data?.pages.at(-1)
 const sources = [...new Map((results.data?.pages ?? []).flatMap(p => p.sources).map(s => [s.id, s])).values()]
 function applyIntent(next: SearchIntent) {
  if (JSON.stringify(next) === JSON.stringify(intent) && results.data) void results.refetch()
  setParams(previous => { const p = new URLSearchParams(previous); p.set('intent', JSON.stringify(next)); p.delete('job'); return p })
  setStarted(true)
 }
 const chat = useJobAssistant(intent, saved, applyIntent)
 function changeMode(next: 'assistant' | 'manual') {
  if (next === 'manual') chat.cancel()
  setParams(previous => { const p = new URLSearchParams(previous); p.set('mode', next); return p })
  setMobileResults(false)
 }
 function selectJob(id?: string) { setParams(previous => { const p = new URLSearchParams(previous); if (id) p.set('job', id); else p.delete('job'); return p }) }
 function manualSearch(query: string) {
  if (query === intent.query && (started || saved)) void results.refetch()
  else applyIntent({ ...intent, query })
  setStarted(true)
  setMobileResults(true)
 }
 async function copySearch() { try { const url = new URL(window.location.href); url.searchParams.delete('job'); await navigator.clipboard.writeText(url.href); toast.success('Search link copied') } catch { toast.error('Could not copy your search link') } }
 const countryLabel = (code: string) => countries.data?.find(c => c.code === code)?.name ?? code
 const summary = intent.branches.map(b => b.workplace === 'remote'
  ? `Remote · ${b.country ? `work from ${countryLabel(b.country)}` : 'any hiring region'}`
  : `${b.workplace === 'all' ? 'Any arrangement' : workplaceLabels[b.workplace]} · ${[b.city, countryLabel(b.country)].filter(Boolean).join(', ') || 'choose an office country'}`)
 return <section className="jobs-page">
  <header className="jobs-heading"><div><h1>Job search</h1><p>Find your next role with the assistant or search manually.</p></div><Button variant="ghost" size="sm" onClick={copySearch}><Copy size={14} />Copy search</Button></header>
  <div className="jobs-workspace-toolbar">
   <nav className="jobs-source-tabs" aria-label="Jobs navigation"><NavLink to="/jobs" end>Discover</NavLink><NavLink to="/jobs/saved"><Bookmark size={14} />Saved jobs</NavLink></nav>
   <div className="jobs-mode-picker" role="group" aria-label="Job search mode">
    <button type="button" className="workspace-mode-button" aria-pressed={mode === 'assistant'} aria-controls="jobs-assistant-pane" onClick={() => changeMode('assistant')}><Sparkles size={15} />AI assistant</button>
    <button type="button" className="workspace-mode-button" aria-pressed={mode === 'manual'} aria-controls="jobs-manual-pane" onClick={() => changeMode('manual')}><SlidersHorizontal size={15} />Manual search</button>
   </div>
  </div>
  <div className="jobs-mobile-surface" role="group" aria-label="Job search surface"><button type="button" aria-pressed={!mobileResults} onClick={() => setMobileResults(false)}>{mode === 'assistant' ? 'Assistant' : 'Filters'}</button><button type="button" aria-pressed={mobileResults} onClick={() => setMobileResults(true)}>Results{jobs.length > 0 && ` (${jobs.length})`}</button></div>
  <div className="jobs-workspace" data-mobile-results={mobileResults}>
   <div className="jobs-controls">
    <div className="jobs-control-pane" id="jobs-assistant-pane" hidden={mode !== 'assistant'}><JobChatPanel chat={chat} active={mode === 'assistant' && !mobileResults} saved={saved} onManual={() => changeMode('manual')} /></div>
    <div className="jobs-control-pane" id="jobs-manual-pane" hidden={mode !== 'manual'}><JobSearchFilters intent={intent} countries={countries.data ?? []} countriesError={countries.isError} onRetryCountries={() => void countries.refetch()} saved={saved} onChange={applyIntent} onSearch={manualSearch} onReset={() => applyIntent(defaultJobIntent())} /></div>
   </div>
   <section className="jobs-discovery-main" aria-label="Job search results">
    <div className="jobs-search-summary"><div><h2>Current search</h2><Button variant="ghost" size="sm" onClick={() => changeMode('manual')}>Edit filters</Button></div>{intent.query && <p className="jobs-summary-keywords">{intent.query}</p>}<ul>{summary.map((text, i) => <li key={i}>{i > 0 && <span>OR </span>}{text}</li>)}</ul><div className="jobs-active-filters">{intent.seniority && <span>{intent.seniority} level</span>}{intent.employmentType && <span>{employmentLabels[intent.employmentType]}</span>}{intent.postedDays > 0 && <span>Past {intent.postedDays} days</span>}{intent.excludeCompanies.map(c => <span key={c}>Exclude {c}</span>)}{intent.includeUnknown && <span>Include unspecified remote regions</span>}</div></div>
        <div className="jobs-results-heading" aria-live="polite"><h2>{saved ? 'Your saved jobs' : 'Search results'}</h2><span>{results.isFetching ? <><Loader2 size={13} className="animate-spin" />Searching…</> : results.data ? `${jobs.length} jobs loaded` : 'Ready when you are'}</span></div>
        {!results.data && !results.isFetching && !results.isError && <div className="jobs-empty"><BriefcaseBusiness size={28} /><h2>A clearer path to your next role.</h2><p>Use the assistant or switch to Manual search. Choose an office country for onsite and hybrid roles, or a remote work country to check hiring eligibility.</p><Button variant="outline" onClick={() => { applyIntent({ ...defaultJobIntent(), branches: [{ workplace: 'remote', country: '', city: '' }] }) }}>Explore remote roles</Button></div>}
        {results.isFetching && !results.data && <div className="jobs-loading" role="status"><p>Checking job sources…</p>{[0, 1, 2].map(n => <div key={n} className="jobs-skeleton"><i /><i /><i /></div>)}</div>}
        {results.isError && <div className="jobs-notice" role="alert"><p>Search could not complete. Your criteria are preserved.</p><Button size="sm" variant="outline" onClick={() => void results.refetch()}>Retry search</Button></div>}
        {sources.some(s => s.status !== 'ready') && <div className="jobs-source-notices" role="status">{sources.filter(s => s.status !== 'ready').map(s => <p key={s.id}><strong>{s.name}:</strong> {s.message}</p>)}</div>}
        {results.data && jobs.length === 0 && !results.isFetching && <div className="jobs-empty"><Search size={24} /><h2>{saved ? 'No saved jobs match these criteria.' : 'No matches in the pages checked.'}</h2><p>{saved ? 'Save roles from Discover, or change your filters.' : 'Try fewer keywords or another location. These sources do not cover every opening.'}{results.hasNextPage && ' More source pages may contain matches.'}</p></div>}
        <div className="jobs-results-list" aria-busy={results.isFetching}>{jobs.map(job => <JobCard key={job.id} job={job} selected={params.get('job') === job.id} onSelect={() => selectJob(job.id)} saving={save.isPending && save.variables?.id === job.id} onSave={() => save.mutate({ id: job.id, saved: !job.saved }, { onError: () => toast.error('Could not update saved jobs') })} />)}</div>
        {results.data && <div className="jobs-pagination"><p>Results from requested source pages, with duplicates removed. Coverage varies by source.</p><div>{lastPage?.retry && <Button variant="outline" size="sm" disabled={results.isFetching} onClick={() => void results.refetch()}>Retry sources</Button>}{results.hasNextPage && <Button variant="outline" size="sm" disabled={results.isFetching} onClick={() => void results.fetchNextPage()}>{results.isFetchingNextPage ? <Loader2 size={14} className="animate-spin" /> : <ArrowDown size={14} />}Load more jobs</Button>}</div></div>}
        {sources.some(s => s.status === 'ready') && <details className="jobs-sources"><summary>Sources and freshness</summary><ul>{sources.filter(s => s.status === 'ready').map(s => <li key={s.id}><strong>{s.name}</strong><span>Fetched {jobDateTime(s.fetchedAt)}</span></li>)}</ul></details>}
   </section>
  </div>
  <JobDetails id={params.get('job')} onClose={() => selectJob()} />
 </section>
}

function JobCard({ job, selected, saving, onSelect, onSave }: { job: Job; selected: boolean; saving: boolean; onSelect: () => void; onSave: () => void }) {
  const date = jobListingDate(job)
  const source = safeJobUrl(job.sourceUrl)
  return <article className="jobs-row" data-selected={selected}>
    <div className="jobs-row-top"><p>{job.company}</p><button type="button" className="jobs-icon-button" aria-label={`${job.saved ? 'Unsave' : 'Save'} ${job.title} at ${job.company}`} aria-pressed={job.saved} disabled={saving} onClick={onSave}><Bookmark size={17} fill={job.saved ? 'currentColor' : 'none'} /></button></div>
    <h3><button type="button" onClick={onSelect}>{job.title}</button></h3>
    <p className="jobs-row-location">{locationLabel(job)}</p>
    <div className="jobs-row-meta"><span>{workplaceLabels[job.workplace]}</span><span>{job.salary || 'Salary not disclosed'}</span>{job.employmentType !== 'unknown' && <span>{employmentLabels[job.employmentType] ?? job.employmentType}</span>}{job.closed ? <span>Closed / unconfirmed</span> : date.date && <time dateTime={date.date}>{date.label} {jobDate(date.date)}</time>}</div>
    <div className="jobs-row-bottom">{source ? <a href={source} target="_blank" rel="noopener noreferrer">{job.sourceName}<ExternalLink size={11} /></a> : <span>{job.sourceName}</span>}<button type="button" onClick={onSelect}>View role and requirements</button></div>
  </article>
}
