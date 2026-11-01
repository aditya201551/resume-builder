import { useEffect, useState, type FormEvent } from 'react'
import { Plus, Search, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { employmentLabels } from '@/lib/jobs'
import type { JobSearchBranch, SearchIntent } from '@/lib/jobSearch'
import type { JobCountry } from '@/types/job'

interface Props {
 intent: SearchIntent
 countries: JobCountry[]
 countriesError: boolean
 onRetryCountries: () => void
 saved: boolean
 onChange: (intent: SearchIntent) => void
 onSearch: (query: string) => void
 onReset: () => void
}

export default function JobSearchFilters({ intent, countries, countriesError, onRetryCountries, saved, onChange, onSearch, onReset }: Props) {
 const [keyword, setKeyword] = useState(intent.query)
 useEffect(() => setKeyword(intent.query), [intent.query])
 function update(values: Partial<SearchIntent>) { onChange({ ...intent, ...values }) }
 function manualSearch(e: FormEvent) { e.preventDefault(); onSearch(keyword.trim()) }
 const changed = keyword.trim() !== intent.query
 return <section className="jobs-manual-panel" aria-label="Manual job search">
  <header className="workspace-pane-heading"><div><h2>Manual search</h2><p>Set your criteria directly. Every change updates the same results.</p></div></header>
  <div className="jobs-manual-scroll">
        <form className="jobs-manual-form" onSubmit={manualSearch}>
          <label htmlFor="jobs-keywords">Role, skill or company</label>
          <div><Search size={17} /><input id="jobs-keywords" value={keyword} onChange={e => setKeyword(e.target.value)} maxLength={160} placeholder="Backend engineer, React, a company…" /><Button type="submit" size="sm">{saved ? 'Filter saved jobs' : 'Search jobs'}</Button></div>
          {changed && <p className="jobs-field-status">Press Search to apply these keywords.</p>}
        </form>
        <div className="jobs-criteria">
          <div className="jobs-criteria-heading"><h2>Search criteria</h2><Button size="sm" variant="ghost" onClick={() => { onReset() }}>Reset</Button></div>
          <p>Match any of these work locations. Remote eligibility and office locations stay separate.</p>
          {intent.branches.map((branch, i) => <BranchEditor key={`${i}-${branch.workplace}`} branch={branch} index={i} countries={countries} removable={intent.branches.length > 1} onChange={next => update({ branches: intent.branches.map((b, n) => n === i ? next : b) })} onRemove={() => update({ branches: intent.branches.filter((_, n) => n !== i) })} />)}
          {countriesError && <p className="jobs-field-error">Countries could not load. <button type="button" onClick={onRetryCountries}>Retry</button></p>}
          <div className="jobs-criteria-actions"><Button size="sm" variant="ghost" disabled={intent.branches.length >= 4} onClick={() => update({ branches: [...intent.branches, { workplace: 'remote', country: '', city: '' }] })}><Plus size={14} />Add alternative</Button></div>
          <div id="jobs-advanced" className="jobs-filters">
            <label>Employment<select value={intent.employmentType} onChange={e => update({ employmentType: e.target.value })}><option value="">Any employment</option>{Object.entries(employmentLabels).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select></label>
            <label>Seniority<select value={intent.seniority} onChange={e => update({ seniority: e.target.value })}><option value="">Any seniority</option><option value="junior">Junior / entry</option><option value="mid">Mid level</option><option value="senior">Senior</option><option value="lead">Staff / lead</option><option value="unknown">Not specified</option></select></label>
            <label>Posted or updated<select value={intent.postedDays} onChange={e => update({ postedDays: Number(e.target.value) })}><option value={0}>Any date</option><option value={1}>Past day</option><option value={7}>Past week</option><option value={30}>Past month</option></select></label>
            <label className="jobs-exclude-field">Exclude companies (comma separated)<input key={intent.excludeCompanies.join(',')} defaultValue={intent.excludeCompanies.join(', ')} maxLength={1000} onBlur={e => { const excludeCompanies = e.target.value.split(',').map(s => s.trim()).filter(Boolean).slice(0, 10); if (excludeCompanies.join(',') !== intent.excludeCompanies.join(',')) update({ excludeCompanies }) }} /></label>
          </div>
          <label className="jobs-checkbox"><input type="checkbox" checked={intent.includeUnknown} disabled={!intent.branches.some(b => b.workplace === 'remote' && b.country)} onChange={e => update({ includeUnknown: e.target.checked })} />Include unspecified hiring regions for a selected remote country</label>
          {(intent.seniority || intent.employmentType || intent.postedDays > 0 || intent.excludeCompanies.length > 0) && <div className="jobs-active-filters" aria-label="Additional active filters">{intent.seniority && <span>{intent.seniority} level</span>}{intent.employmentType && <span>{employmentLabels[intent.employmentType]}</span>}{intent.postedDays > 0 && <span>Past {intent.postedDays} days</span>}{intent.excludeCompanies.map(c => <span key={c}>Exclude {c}</span>)}</div>}
        </div>
  </div>
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

