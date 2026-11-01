import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { NavLink, useSearchParams } from 'react-router'
import { ArrowDown, ArrowUp, Bookmark, BriefcaseBusiness, Check, Copy, ExternalLink, Loader2, Plus, Search, SlidersHorizontal, Sparkles, X } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import JobDetails from '@/components/jobs/JobDetails'
import { useJobCountries, useSaveJob } from '@/hooks/useJobs'
import { useJobDiscovery } from '@/hooks/useJobDiscovery'
import { apiPost } from '@/lib/http'
import { defaultJobIntent, readJobIntent, uniqueDiscoveryJobs, type JobRefinement, type JobSearchBranch, type SearchIntent } from '@/lib/jobSearch'
import { employmentLabels, jobDate, jobDateTime, jobListingDate, locationLabel, safeJobUrl, workplaceLabels } from '@/lib/jobs'
import type { Job, JobCountry } from '@/types/job'
import '@/styles/jobs.css'

const starters = ['Find Go backend roles remote from India', 'Hybrid product design roles in London, UK', 'Senior frontend roles in Germany']

export default function JobsPage({ saved = false }: { saved?: boolean }) {
  const [params, setParams] = useSearchParams()
  const rawIntent = params.get('intent')
  const intent = useMemo(() => readJobIntent(rawIntent), [rawIntent])
  const [started, setStarted] = useState(Boolean(rawIntent))
  const [keyword, setKeyword] = useState(intent.query)
  const [prompt, setPrompt] = useState('')
  const [advanced, setAdvanced] = useState(false)
  const [messages, setMessages] = useState<{ role: 'user' | 'assistant'; text: string }[]>([])
  const [thinking, setThinking] = useState(false)
  const assistantRequest = useRef<AbortController | null>(null)
  const searchVersion = useRef('')
  searchVersion.current = JSON.stringify(intent) + saved
  const countries = useJobCountries()
  const results = useJobDiscovery(intent, saved, started || saved || Boolean(rawIntent))
  const save = useSaveJob()
  const jobs = uniqueDiscoveryJobs(results.data?.pages ?? [])
  const lastPage = results.data?.pages.at(-1)
  const sources = [...new Map((results.data?.pages ?? []).flatMap(p => p.sources).map(s => [s.id, s])).values()]
  useEffect(() => { setKeyword(intent.query) }, [intent.query])
  useEffect(() => () => assistantRequest.current?.abort(), [])

  function applyIntent(next: SearchIntent) {
    setParams(previous => { const p = new URLSearchParams(previous); p.set('intent', JSON.stringify(next)); p.delete('job'); return p })
    setStarted(true)
  }
  function update(values: Partial<SearchIntent>) { applyIntent({ ...intent, ...values }) }
  function selectJob(id?: string) { setParams(previous => { const p = new URLSearchParams(previous); if (id) p.set('job', id); else p.delete('job'); return p }) }
  function manualSearch(e: FormEvent) { e.preventDefault(); if (keyword.trim() === intent.query && (started || saved)) void results.refetch(); else update({ query: keyword.trim() }); setStarted(true) }
  async function refine(text: string) {
    if (!text.trim() || thinking) return
    const version = searchVersion.current
    const controller = new AbortController()
    assistantRequest.current = controller
    setThinking(true); setPrompt('')
    setMessages(m => [...m.slice(-5), { role: 'user', text }])
    try {
      const response = await apiPost<JobRefinement>('/api/jobs/assist', { intent, message: text, history: messages.slice(-4) }, { signal: controller.signal })
      if (searchVersion.current !== version) { setMessages(m => [...m, { role: 'assistant', text: 'Your filters changed while I was working. I kept your latest search; send the refinement again to apply it there.' }]); return }
      setMessages(m => [...m, { role: 'assistant', text: response.message }])
      if (response.apply) applyIntent(response.intent)
    } catch {
      if (!controller.signal.aborted) setMessages(m => [...m, { role: 'assistant', text: 'The assistant is unavailable right now. Your search is unchanged, and you can continue using the filters.' }])
    } finally { if (assistantRequest.current === controller) { assistantRequest.current = null; setThinking(false) } }
  }
  function cancelAssistant() { assistantRequest.current?.abort(); assistantRequest.current = null; setThinking(false); setMessages(m => [...m, { role: 'assistant', text: 'Cancelled. Your search criteria are unchanged.' }]) }
  async function copySearch() { try { const url = new URL(window.location.href); url.searchParams.delete('job'); await navigator.clipboard.writeText(url.href); toast.success('Search link copied') } catch { toast.error('Could not copy your search link') } }
  const changed = keyword.trim() !== intent.query

  return <section className="jobs-page">
    <header className="jobs-heading"><div><p className="jobs-eyebrow">YOUR NEXT CHAPTER</p><h1>Find work that fits.</h1><p>Technology roles worldwide. Search your way.</p></div><Button variant="ghost" size="sm" onClick={copySearch}><Copy size={14} />Copy search</Button></header>
    <nav className="jobs-source-tabs" aria-label="Jobs navigation"><NavLink to="/jobs" end>Discover</NavLink><NavLink to="/jobs/saved"><Bookmark size={14} />Saved jobs</NavLink></nav>
    <div className="jobs-discovery-layout">
      <aside className="jobs-assistant" aria-label="Job search assistant">
        <div className="jobs-assistant-heading"><Sparkles size={18} /><h2>Your search, in your words.</h2></div>
        <p>Describe the role you want, or ask me to refine your current filters.</p>
        <form onSubmit={e => { e.preventDefault(); void refine(prompt) }}>
          <label htmlFor="jobs-prompt" className="sr-only">Describe or refine your job search</label>
          <textarea id="jobs-prompt" value={prompt} onChange={e => setPrompt(e.target.value)} maxLength={2000} rows={4} placeholder="A backend role using Go, remote from India or hybrid in Bengaluru…" disabled={thinking} />
          <div className="jobs-assistant-submit"><span>AI-assisted search</span>{thinking ? <Button type="button" size="sm" variant="outline" onClick={cancelAssistant}><Loader2 size={14} className="animate-spin" />Cancel</Button> : <Button type="submit" size="sm" disabled={!prompt.trim()}><ArrowUp size={15} />Find my next role</Button>}</div>
        </form>
        {messages.length === 0 && <div className="jobs-starters"><p>Try a starting point</p>{starters.map(text => <button type="button" key={text} onClick={() => void refine(text)} disabled={thinking}>{text}<ArrowUp size={12} /></button>)}</div>}
        {messages.length > 0 && <div className="jobs-conversation" role="log" aria-label="Search conversation" aria-live="polite">{messages.slice(-6).map((m, i) => <div key={i} data-role={m.role}><span>{m.role === 'user' ? 'You' : 'Assistant'}</span><p>{m.text}</p></div>)}{thinking && <p role="status">Working out your search criteria…</p>}</div>}
        <p className="jobs-assistant-note"><Check size={14} />Every criterion stays visible and editable. Search filters work independently of AI.</p>
      </aside>
      <div className="jobs-discovery-main">
        <form className="jobs-manual-form" onSubmit={manualSearch}>
          <label htmlFor="jobs-keywords">Role, skill or company</label>
          <div><Search size={17} /><input id="jobs-keywords" value={keyword} onChange={e => setKeyword(e.target.value)} maxLength={160} placeholder="Backend engineer, React, a company…" /><Button type="submit" size="sm">{saved ? 'Filter saved jobs' : 'Search'}</Button></div>
          {changed && <p className="jobs-field-status">Press Search to apply these keywords.</p>}
        </form>
        <div className="jobs-criteria">
          <div className="jobs-criteria-heading"><h2>Search criteria</h2><Button size="sm" variant="ghost" onClick={() => { applyIntent(defaultJobIntent()); setMessages([]) }}>Reset</Button></div>
          <p>Match any of these work locations. Remote eligibility and office locations stay separate.</p>
          {intent.branches.map((branch, i) => <BranchEditor key={`${i}-${branch.workplace}`} branch={branch} index={i} countries={countries.data ?? []} removable={intent.branches.length > 1} onChange={next => update({ branches: intent.branches.map((b, n) => n === i ? next : b) })} onRemove={() => update({ branches: intent.branches.filter((_, n) => n !== i) })} />)}
          {countries.isError && <p className="jobs-field-error">Countries could not load. <button type="button" onClick={() => void countries.refetch()}>Retry</button></p>}
          <div className="jobs-criteria-actions"><Button size="sm" variant="ghost" disabled={intent.branches.length >= 4} onClick={() => update({ branches: [...intent.branches, { workplace: 'remote', country: '', city: '' }] })}><Plus size={14} />Add alternative</Button><Button size="sm" variant="ghost" aria-expanded={advanced} aria-controls="jobs-advanced" onClick={() => setAdvanced(!advanced)}><SlidersHorizontal size={14} />More filters</Button></div>
          <div id="jobs-advanced" className="jobs-filters" hidden={!advanced}>
            <label>Employment<select value={intent.employmentType} onChange={e => update({ employmentType: e.target.value })}><option value="">Any employment</option>{Object.entries(employmentLabels).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select></label>
            <label>Seniority<select value={intent.seniority} onChange={e => update({ seniority: e.target.value })}><option value="">Any seniority</option><option value="junior">Junior / entry</option><option value="mid">Mid level</option><option value="senior">Senior</option><option value="lead">Staff / lead</option><option value="unknown">Not specified</option></select></label>
            <label>Posted or updated<select value={intent.postedDays} onChange={e => update({ postedDays: Number(e.target.value) })}><option value={0}>Any date</option><option value={1}>Past day</option><option value={7}>Past week</option><option value={30}>Past month</option></select></label>
            <label className="jobs-exclude-field">Exclude companies (comma separated)<input key={intent.excludeCompanies.join(',')} defaultValue={intent.excludeCompanies.join(', ')} maxLength={1000} onBlur={e => { const excludeCompanies = e.target.value.split(',').map(s => s.trim()).filter(Boolean).slice(0, 10); if (excludeCompanies.join(',') !== intent.excludeCompanies.join(',')) update({ excludeCompanies }) }} /></label>
          </div>
          <label className="jobs-checkbox"><input type="checkbox" checked={intent.includeUnknown} disabled={!intent.branches.some(b => b.workplace === 'remote' && b.country)} onChange={e => update({ includeUnknown: e.target.checked })} />Include unspecified hiring regions for a selected remote country</label>
          {(intent.seniority || intent.employmentType || intent.postedDays > 0 || intent.excludeCompanies.length > 0) && <div className="jobs-active-filters" aria-label="Additional active filters">{intent.seniority && <span>{intent.seniority} level</span>}{intent.employmentType && <span>{employmentLabels[intent.employmentType]}</span>}{intent.postedDays > 0 && <span>Past {intent.postedDays} days</span>}{intent.excludeCompanies.map(c => <span key={c}>Exclude {c}</span>)}</div>}
        </div>
        <div className="jobs-results-heading" aria-live="polite"><h2>{saved ? 'Your saved jobs' : 'Search results'}</h2><span>{results.isFetching ? <><Loader2 size={13} className="animate-spin" />Searching…</> : results.data ? `${jobs.length} jobs loaded` : 'Ready when you are'}</span></div>
        {!results.data && !results.isFetching && !results.isError && <div className="jobs-empty"><BriefcaseBusiness size={28} /><h2>A clearer path to your next role.</h2><p>Ask the assistant or use the filters. Choose an office country for onsite and hybrid roles, or a remote work country to check hiring eligibility.</p><Button variant="outline" onClick={() => { applyIntent({ ...defaultJobIntent(), branches: [{ workplace: 'remote', country: '', city: '' }] }) }}>Explore remote roles</Button></div>}
        {results.isFetching && !results.data && <div className="jobs-loading" role="status"><p>Checking job sources…</p>{[0, 1, 2].map(n => <div key={n} className="jobs-skeleton"><i /><i /><i /></div>)}</div>}
        {results.isError && <div className="jobs-notice" role="alert"><p>Search could not complete. Your criteria are preserved.</p><Button size="sm" variant="outline" onClick={() => void results.refetch()}>Retry search</Button></div>}
        {sources.some(s => s.status !== 'ready') && <div className="jobs-source-notices" role="status">{sources.filter(s => s.status !== 'ready').map(s => <p key={s.id}><strong>{s.name}:</strong> {s.message}</p>)}</div>}
        {results.data && jobs.length === 0 && !results.isFetching && <div className="jobs-empty"><Search size={24} /><h2>{saved ? 'No saved jobs match these criteria.' : 'No matches in the pages checked.'}</h2><p>{saved ? 'Save roles from Discover, or change your filters.' : 'Try fewer keywords or another location. These sources do not cover every opening.'}{results.hasNextPage && ' More source pages may contain matches.'}</p></div>}
        <div className="jobs-results-list" aria-busy={results.isFetching}>{jobs.map(job => <JobCard key={job.id} job={job} selected={params.get('job') === job.id} onSelect={() => selectJob(job.id)} saving={save.isPending && save.variables?.id === job.id} onSave={() => save.mutate({ id: job.id, saved: !job.saved }, { onError: () => toast.error('Could not update saved jobs') })} />)}</div>
        {results.data && <div className="jobs-pagination"><p>Results from requested source pages, with duplicates removed. Coverage varies by source.</p><div>{lastPage?.retry && <Button variant="outline" size="sm" disabled={results.isFetching} onClick={() => void results.refetch()}>Retry sources</Button>}{results.hasNextPage && <Button variant="outline" size="sm" disabled={results.isFetching} onClick={() => void results.fetchNextPage()}>{results.isFetchingNextPage ? <Loader2 size={14} className="animate-spin" /> : <ArrowDown size={14} />}Load more jobs</Button>}</div></div>}
        {sources.some(s => s.status === 'ready') && <details className="jobs-sources"><summary>Sources and freshness</summary><ul>{sources.filter(s => s.status === 'ready').map(s => <li key={s.id}><strong>{s.name}</strong><span>Fetched {jobDateTime(s.fetchedAt)}</span></li>)}</ul></details>}
      </div>
    </div>
    <JobDetails id={params.get('job')} onClose={() => selectJob()} />
  </section>
}

function BranchEditor({ branch, index, countries, removable, onChange, onRemove }: { branch: JobSearchBranch; index: number; countries: JobCountry[]; removable: boolean; onChange: (branch: JobSearchBranch) => void; onRemove: () => void }) {
  const [city, setCity] = useState(branch.city)
  useEffect(() => setCity(branch.city), [branch.city])
  return <div className="jobs-branch">
    {index > 0 && <span className="jobs-branch-or">OR</span>}
    <div className="jobs-branch-fields">
      <label>Work arrangement<select value={branch.workplace} onChange={e => onChange({ workplace: e.target.value as JobSearchBranch['workplace'], country: '', city: '' })}><option value="all">All arrangements</option><option value="remote">Remote</option><option value="hybrid">Hybrid</option><option value="onsite">Onsite</option><option value="unknown">Not specified</option></select></label>
      <label>{branch.workplace === 'remote' ? 'Work remotely from' : 'Office country'}<select value={branch.country} onChange={e => onChange({ ...branch, country: e.target.value, city: '' })}><option value="">{branch.workplace === 'remote' ? 'Any hiring region' : 'Choose a country'}</option>{branch.country && !countries.some(c => c.code === branch.country) && <option value={branch.country}>{branch.country}</option>}{countries.map(c => <option key={c.code} value={c.code}>{c.name}</option>)}</select></label>
      {branch.workplace !== 'remote' && <label>Office city (optional)<input value={city} onChange={e => setCity(e.target.value)} onBlur={() => { if (city.trim() !== branch.city) onChange({ ...branch, city: city.trim() }) }} placeholder="Any city" maxLength={100} disabled={!branch.country} /></label>}
      {removable && <Button size="icon-sm" variant="ghost" onClick={onRemove} aria-label={`Remove location alternative ${index + 1}`}><X size={15} /></Button>}
    </div>
    {branch.workplace === 'all' && branch.country && <p className="jobs-field-status">This country filters office roles. Add a Remote alternative to search remote eligibility separately.</p>}
    {branch.workplace === 'remote' && !branch.country && <p className="jobs-field-status">Includes country-restricted roles. Choose where you plan to work to check eligibility.</p>}
  </div>
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
