import { Component, Suspense, lazy, useEffect, useMemo, useState, type FormEvent, type ReactNode } from 'react'
import { Link, NavLink, useSearchParams } from 'react-router'
import { Bookmark, BriefcaseBusiness, ChevronLeft, ChevronRight, Copy, ExternalLink, List, Map as MapIcon, MapPin, Monitor, Search, SlidersHorizontal } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { useJobCountries, useJobs, useSaveJob } from '@/hooks/useJobs'
import LocationSearch from '@/components/jobs/LocationSearch'
import JobDetails from '@/components/jobs/JobDetails'
import { jobDate, jobSearchQuery, locationLabel, parseBounds, safeJobUrl, workplaceLabels } from '@/lib/jobs'
import { jobMapViewportParams, type JobMapViewport } from '@/lib/jobMapViewport'
import type { Job, JobPlace, JobSource } from '@/types/job'
import '@/styles/jobs.css'

const WorkplaceMap = lazy(() => import('@/components/jobs/JobMap'))

export default function JobMapPage({ direct = false }: { direct?: boolean }) {
  return <main className="jobs-page">
    <header className="jobs-heading"><div><h1>Job map</h1><p>Technology roles, near you or remote.</p></div></header>
    <nav className="jobs-source-tabs" aria-label="Job sources">
      <NavLink to="/jobs/external">Third-party jobs</NavLink>
      <NavLink to="/jobs/direct">Direct jobs</NavLink>
    </nav>
    {direct ? <section className="jobs-coming-soon"><BriefcaseBusiness size={32} strokeWidth={1.3} aria-hidden /><h2>Direct jobs are coming soon</h2><p>Roles published through our own hiring platform will appear here. You can explore third-party openings today.</p><Button asChild><Link to="/jobs/external">Explore third-party jobs</Link></Button></section> : <ExternalJobs />}
  </main>
}

class MapBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false }
  static getDerivedStateFromError() { return { failed: true } }
  render() {
    if (this.state.failed) return <div className="jobs-map-fallback"><MapPin size={28} /><h3>The map is unavailable</h3><p>Continue browsing jobs in the list.</p><Button variant="outline" onClick={() => this.setState({ failed: false })}>Retry map</Button></div>
    return this.props.children
  }
}

function ExternalJobs() {
  const [params, setParams] = useSearchParams()
  const [queryInput, setQueryInput] = useState(params.get('q') ?? '')
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [mobile, setMobile] = useState(() => window.matchMedia('(max-width: 899px)').matches)
  const countries = useJobCountries()
  const searchString = jobSearchQuery(params)
  const results = useJobs(searchString)
  const save = useSaveJob()
  const workplace = params.get('workplace') ?? 'all'
  const savedOnly = params.get('saved') === 'true'
  const remoteOnly = workplace === 'remote'
  const mobileView = params.get('view') === 'map' ? 'map' : 'list'
  const showMap = !remoteOnly && (!mobile || mobileView === 'map')
  const boundsValue = params.get('bounds')
  const latitudeValue = params.get('lat')
  const longitudeValue = params.get('lng')
  const zoomValue = params.get('zoom')
  const queryValue = params.get('q') ?? ''
  const bounds = useMemo(() => parseBounds(boundsValue), [boundsValue])
  const center = useMemo(() => {
    const lat = Number(latitudeValue), lng = Number(longitudeValue), zoom = Number(zoomValue ?? 9)
    if (latitudeValue === null || longitudeValue === null || !Number.isFinite(lat) || !Number.isFinite(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180) return undefined
    return { latitude: lat, longitude: lng, zoom: Number.isFinite(zoom) ? Math.max(1, Math.min(17, zoom)) : 9 }
  }, [latitudeValue, longitudeValue, zoomValue])
  useEffect(() => { setQueryInput(queryValue) }, [queryValue])
  useEffect(() => {
    const media = window.matchMedia('(max-width: 899px)')
    const update = () => setMobile(media.matches)
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [])

  function update(values: Record<string, string | undefined>, resetPage = true) {
    setParams(previous => {
      const next = new URLSearchParams(previous)
      if (resetPage) { next.delete('page'); next.delete('job') }
      for (const [key, value] of Object.entries(values)) { if (value) next.set(key, value); else next.delete(key) }
      return next
    })
  }
  function selectPlace(place: JobPlace) {
    const lat = place.latitude, lng = place.longitude
    update({ place: [place.name, place.country].filter(Boolean).join(', '), lat: String(lat), lng: String(lng), zoom: '10', country: place.countryCode, bounds: [Math.max(-180, lng - 0.7), Math.max(-90, lat - 0.5), Math.min(180, lng + 0.7), Math.min(90, lat + 0.5)].join(',') })
  }
  function viewportChanged(viewport: JobMapViewport) {
    setParams(previous => jobMapViewportParams(previous, viewport), { replace: true })
  }
  function search(event: FormEvent) {
    event.preventDefault()
    const value = queryInput.trim()
    if (value === queryValue && Number(params.get('page') ?? 1) === 1) void results.refetch()
    else update({ q: value })
  }
  async function copyLink() {
    try { const url = new URL(window.location.href); url.searchParams.delete('job'); await navigator.clipboard.writeText(url.href); toast.success('Search link copied') }
    catch { toast.error('Could not copy the link. You can copy the address from your browser.') }
  }

  const jobs = results.data?.jobs ?? []
  const physical = jobs.filter(j => j.workplace !== 'remote')
  const remote = jobs.filter(j => j.workplace === 'remote')
  const countryOptions = countries.data ?? []
  const select = (id: string) => update({ job: id }, false)
  const renderJob = (job: Job) => <JobRow key={job.id} job={job} selected={params.get('job') === job.id} onSelect={() => select(job.id)} saving={save.isPending && save.variables?.id === job.id}
    onSave={() => save.mutate({ id: job.id, saved: !job.saved }, { onError: () => toast.error('Could not update your saved jobs. Try again.') })} />

  return <>
    <form className="jobs-search-form" onSubmit={search}>
      <div className="jobs-keyword"><label htmlFor="job-keywords">Role, skill or company</label><div><Search size={16} aria-hidden /><input id="job-keywords" value={queryInput} maxLength={160} placeholder="Try backend, Python or a company" onChange={event => setQueryInput(event.target.value)} /></div></div>
      <LocationSearch value={params.get('place') ?? ''} onSelect={selectPlace} onClear={() => update({ place: undefined, lat: undefined, lng: undefined, zoom: undefined, bounds: undefined, country: undefined })} />
      <Button type="submit" className="jobs-submit">Search jobs</Button>
    </form>
    <div className="jobs-toolbar">
      <div className="jobs-workplace-switch" role="group" aria-label="Workplace arrangement">{(['all', 'onsite', 'hybrid', 'remote'] as const).map(value => <button type="button" key={value} aria-pressed={workplace === value} onClick={() => update({ workplace: value === 'all' ? undefined : value })}>{value === 'all' ? 'All jobs' : workplaceLabels[value]}</button>)}</div>
      <div className="jobs-toolbar-actions"><Button variant={savedOnly ? 'secondary' : 'ghost'} size="sm" aria-pressed={savedOnly} onClick={() => update({ saved: savedOnly ? undefined : 'true' })}><Bookmark size={15} />Saved</Button><Button variant="ghost" size="sm" aria-expanded={filtersOpen} aria-controls="jobs-filters" onClick={() => setFiltersOpen(v => !v)}><SlidersHorizontal size={15} />Filters</Button><Button variant="ghost" size="icon-sm" aria-label="Copy search link" onClick={copyLink}><Copy size={15} /></Button></div>
    </div>
    {(workplace === 'all' || remoteOnly) && <div className="jobs-remote-preference" data-collapsed={mobile && !filtersOpen && !remoteOnly && !params.has('remoteCountry')}><Monitor size={16} aria-hidden /><label htmlFor="remote-country">I plan to work remotely from</label><select id="remote-country" value={params.get('remoteCountry') ?? ''} onChange={event => update({ remoteCountry: event.target.value })}><option value="">Any location</option>{countryOptions.map(c => <option key={c.code} value={c.code}>{c.name}</option>)}</select>{params.get('remoteCountry') && <label className="jobs-checkbox"><input type="checkbox" checked={params.get('includeUnknown') === 'true'} onChange={event => update({ includeUnknown: event.target.checked ? 'true' : undefined })} />Include unspecified hiring regions</label>}</div>}
    <div className="jobs-filters" id="jobs-filters" hidden={!filtersOpen}>
      <label>Workplace country<select value={params.get('country') ?? ''} onChange={event => update({ country: event.target.value, bounds: undefined, place: undefined, lat: undefined, lng: undefined, zoom: undefined })}><option value="">All countries</option>{countryOptions.map(c => <option key={c.code} value={c.code}>{c.name}</option>)}</select></label>
      <label>Employment<select value={params.get('employment') ?? ''} onChange={event => update({ employment: event.target.value })}><option value="">Any employment</option><option value="full_time">Full time</option><option value="part_time">Part time</option><option value="contract">Contract</option><option value="internship">Internship</option></select></label>
      <label>Seniority<select value={params.get('seniority') ?? ''} onChange={event => update({ seniority: event.target.value })}><option value="">Any seniority</option><option value="junior">Junior / intern</option><option value="mid">Mid level</option><option value="senior">Senior</option><option value="lead">Staff / lead</option></select></label>
      <label>Listing date<select value={params.get('days') ?? ''} onChange={event => update({ days: event.target.value })}><option value="">Any date</option><option value="1">Past day</option><option value="7">Past week</option><option value="30">Past month</option></select></label>
      <label>Listing source<select value={params.get('source') ?? ''} onChange={event => update({ source: event.target.value })}><option value="">All sources</option><option value="company">Company hiring feeds</option><option value="aggregator">Job boards</option><option value="career_page">Public career pages</option></select></label>
      <Button variant="ghost" size="sm" onClick={() => setParams({})}>Clear all filters</Button>
    </div>
    {bounds && !params.get('place') && <div className="jobs-area-note"><MapPin size={14} /><span>Workplaces update as you move the map. Remote preferences stay independent.</span><button type="button" onClick={() => update({ bounds: undefined, lat: undefined, lng: undefined, zoom: undefined, country: undefined })}>Reset map</button></div>}
    <div className="jobs-results-heading"><p aria-live="polite">{results.isPending ? 'Finding openings…' : results.isError ? 'Search unavailable' : <><strong>{results.data?.total ?? 0}</strong> matching indexed {savedOnly ? 'saved ' : ''}jobs{results.isFetching && <span> Updating…</span>}</>}</p><div className="jobs-mobile-switch" role="group" aria-label="Results view"><button type="button" aria-pressed={mobileView === 'list'} onClick={() => update({ view: undefined }, false)}><List size={15} />List</button><button type="button" disabled={remoteOnly} aria-pressed={mobileView === 'map' && !remoteOnly} onClick={() => update({ view: 'map' }, false)}><MapIcon size={15} />Map</button></div></div>
    <div className={`jobs-workspace${remoteOnly ? ' jobs-workspace-remote' : ''}`} data-mobile-view={remoteOnly ? 'list' : mobileView}>
      <section className="jobs-results" aria-label="Job results" aria-busy={results.isFetching}>
        {results.isPending && <div className="jobs-loading" role="status"><span className="sr-only">Loading job results</span>{Array.from({ length: 5 }, (_, i) => <div className="jobs-skeleton" key={i}><i /><i /><i /></div>)}</div>}
        {results.isError && <div className="jobs-empty"><Search size={28} /><h2>Could not load jobs</h2><p>Check your connection and try the search again.</p><Button variant="outline" onClick={() => void results.refetch()}>Retry search</Button></div>}
        {results.data && jobs.length === 0 && <div className="jobs-empty"><Search size={28} /><h2>{results.data.refreshing ? 'The first listings are arriving' : savedOnly ? 'No saved jobs match this search' : 'No indexed jobs match yet'}</h2><p>{results.data.refreshing ? 'Company feeds are refreshing. Results will appear here automatically.' : savedOnly ? 'Save an opening using its bookmark button, or broaden your filters.' : 'Try another role or a wider area. Coverage varies by source; an empty search does not mean there are no jobs here.'}</p><Button variant="outline" onClick={() => setParams({})}>Browse all indexed jobs</Button></div>}
        {physical.length > 0 && <div className="jobs-result-group"><h2>Workplaces {bounds || params.get('country') ? 'in your search area' : 'around the world'}</h2>{physical.map(renderJob)}</div>}
        {remote.length > 0 && <div className="jobs-result-group"><h2>Remote roles{params.get('remoteCountry') ? ' for your work location' : ''}</h2>{remote.map(renderJob)}</div>}
      </section>
      {showMap && <MapBoundary><Suspense fallback={<div className="jobs-map-fallback" role="status">Loading the workplace map…</div>}><WorkplaceMap jobs={jobs} selected={params.get('job')} center={center} bounds={bounds} onSelect={select} onViewportChange={viewportChanged} /></Suspense></MapBoundary>}
    </div>
    {results.data && <>
      <div className="jobs-pagination"><p>{jobs.length > 0 ? `Showing ${(results.data.page - 1) * results.data.pageSize + 1}–${(results.data.page - 1) * results.data.pageSize + jobs.length} of ${results.data.total}` : 'No matching listings'}{showMap && `; ${results.data.mapped} jobs mapped on this page`}</p><div><Button variant="outline" size="sm" disabled={results.data.page <= 1 || results.isFetching} onClick={() => update({ page: String(results.data!.page - 1) }, false)}><ChevronLeft size={15} />Previous</Button><Button variant="outline" size="sm" disabled={!results.data.hasMore || results.isFetching} onClick={() => update({ page: String(results.data!.page + 1) }, false)}>Next<ChevronRight size={15} /></Button></div></div>
      <Sources sources={results.data.sources} refreshing={results.data.refreshing} />
    </>}
    <footer className="jobs-attribution">Map data from <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noopener noreferrer">OpenStreetMap</a>, basemap by <a href="https://openfreemap.org/" target="_blank" rel="noopener noreferrer">OpenFreeMap</a>. City search and approximate positions from <a href="https://www.geonames.org/" target="_blank" rel="noopener noreferrer">GeoNames</a> under CC BY 4.0. Extended place search by <a href="https://www.geoapify.com/" target="_blank" rel="noopener noreferrer">Geoapify</a> when connected.</footer>
    <JobDetails id={params.get('job')} onClose={() => update({ job: undefined }, false)} />
  </>
}

function JobRow({ job, selected, onSelect, onSave, saving }: { job: Job; selected: boolean; onSelect: () => void; onSave: () => void; saving: boolean }) {
  const source = safeJobUrl(job.sourceUrl)
  return <article className="jobs-row" data-selected={selected}>
    <div className="jobs-row-top"><p>{job.company}</p><button type="button" className="jobs-icon-button" aria-label={`${job.saved ? 'Unsave' : 'Save'} ${job.title} at ${job.company}`} aria-pressed={job.saved} disabled={saving} onClick={onSave}><Bookmark size={17} fill={job.saved ? 'currentColor' : 'none'} /></button></div>
    <h3><button type="button" onClick={onSelect}>{job.title}</button></h3>
    <p className="jobs-row-location">{job.workplace === 'remote' ? <Monitor size={14} /> : <MapPin size={14} />}<span>{locationLabel(job)}</span></p>
    <div className="jobs-row-meta"><span>{workplaceLabels[job.workplace]}</span>{job.salary && <span>{job.salary}</span>}{job.closed ? <span>Closed / unconfirmed</span> : job.postedAt && <time dateTime={job.postedAt}>{jobDate(job.postedAt)}</time>}</div>
    <div className="jobs-row-bottom">{source ? <a href={source} target="_blank" rel="noopener noreferrer">{job.sourceName}<ExternalLink size={11} /></a> : <span>{job.sourceName}</span>}{job.locations.some(l => l.precision === 'city') && <span>Approximate city location</span>}</div>
  </article>
}

function Sources({ sources, refreshing }: { sources: JobSource[]; refreshing: boolean }) {
  const errors = sources.filter(s => s.status === 'error' || s.status === 'quota_exhausted')
  return <details className="jobs-sources"><summary>{refreshing ? 'Sources are refreshing' : `${sources.filter(s => s.status === 'ready').length} sources connected`}{errors.length > 0 && `; ${errors.length} need attention`}<span>Coverage and freshness</span></summary><p>Results cover the feeds indexed here. Sources may omit roles or locations. Broad search uses a limited prototype allowance.</p><ul>{sources.map(source => <li key={source.id}><strong>{source.name}</strong><span>{source.status === 'ready' ? `Updated ${jobDate(source.refreshedAt)}` : source.status === 'refreshing' ? 'Refreshing' : source.message}</span></li>)}</ul></details>
}
