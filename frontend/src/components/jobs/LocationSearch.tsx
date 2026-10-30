import { useEffect, useId, useRef, useState } from 'react'
import { LocateFixed, MapPin, X } from 'lucide-react'
import { apiGet } from '@/lib/http'
import { useJobPlaces } from '@/hooks/useJobs'
import { Button } from '@/components/ui/button'
import { getJobMapPosition } from '@/lib/jobMapLocation'
import type { JobPlace } from '@/types/job'

export default function LocationSearch({ value, onSelect, onClear }: { value: string; onSelect: (place: JobPlace) => void; onClear: () => void }) {
  const [input, setInput] = useState(value)
  const [debounced, setDebounced] = useState('')
  const [open, setOpen] = useState(false)
  const [index, setIndex] = useState(0)
  const [locating, setLocating] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const root = useRef<HTMLDivElement>(null)
  const inputElement = useRef<HTMLInputElement>(null)
  const mounted = useRef(true)
  const locationRequest = useRef(0)
  const listId = useId()
  const places = useJobPlaces(debounced)
  useEffect(() => { setInput(value); inputElement.current?.setCustomValidity('') }, [value])
  useEffect(() => { mounted.current = true; return () => { mounted.current = false } }, [])
  useEffect(() => { const timer = setTimeout(() => setDebounced(open ? input : ''), 350); return () => clearTimeout(timer) }, [input, open])
  useEffect(() => {
    const close = (event: PointerEvent) => { if (!root.current?.contains(event.target as Node)) setOpen(false) }
    document.addEventListener('pointerdown', close)
    return () => document.removeEventListener('pointerdown', close)
  }, [])
  const cancelLocation = () => { locationRequest.current++; setLocating(false) }
  const select = (place: JobPlace) => { cancelLocation(); inputElement.current?.setCustomValidity(''); onSelect(place); setInput([place.name, place.country].filter(Boolean).join(', ')); setOpen(false); setError(''); setNotice('') }
  const locate = async () => {
    if (!navigator.geolocation) { setError('Location is unavailable in this browser. Search for a city instead.'); return }
    const request = ++locationRequest.current
    const active = () => mounted.current && request === locationRequest.current
    setLocating(true); setError(''); setNotice(''); setOpen(false)
    try {
      const { latitude, longitude, accuracy } = await getJobMapPosition(navigator.geolocation)
      if (!active()) return
      // The nearest-city endpoint labels the reading; it must never replace
      // the device's actual coordinates or prevent centering if it is unavailable.
      const place = await apiGet<JobPlace | null>(`/api/jobs/nearest?lat=${latitude}&lng=${longitude}`).catch(() => null)
      if (!active()) return
      select({ id: 'device-location', country: '', countryCode: '', attribution: '', ...place, name: place ? `Near ${place.name}` : 'Your location', latitude, longitude })
      if (Number.isFinite(accuracy) && accuracy > 2000) setNotice(`Your browser estimates your location within about ${Math.ceil(accuracy / 1000)} km. Choose a city if this area looks wrong.`)
    } catch (failure) {
      if (!active()) return
      const denied = typeof failure === 'object' && failure !== null && 'code' in failure && failure.code === 1
      setError(denied ? 'Location permission was declined. You can search for any city.' : 'Could not locate you. Search for a city instead.')
    } finally { if (active()) setLocating(false) }
  }
  const options = debounced === input ? places.data ?? [] : []
  return <div className="jobs-place" ref={root}>
    <label htmlFor={`${listId}-input`}>Explore workplaces near</label>
    <div className="jobs-place-input">
      <MapPin size={16} aria-hidden />
      <input ref={inputElement} id={`${listId}-input`} value={input} role="combobox" aria-expanded={open && input.length >= 2} aria-controls={listId} aria-autocomplete="list" aria-activedescendant={open && options[index] ? `${listId}-${index}` : undefined} placeholder="City, anywhere in the world" autoComplete="off"
        onFocus={() => { if (input !== value) setOpen(true) }}
        onChange={event => { cancelLocation(); setNotice(''); const next = event.target.value; event.target.setCustomValidity(next.trim() && next !== value ? 'Choose a city from the suggestions, or clear the field.' : ''); setInput(next); setOpen(true); setIndex(0); setError('') }}
        onKeyDown={event => {
          if (event.key === 'Escape') { setOpen(false); setInput(value); inputElement.current?.setCustomValidity('') }
          if (event.key === 'ArrowDown') { event.preventDefault(); setOpen(true); setIndex(i => Math.max(0, Math.min(i + 1, options.length - 1))) }
          if (event.key === 'ArrowUp') { event.preventDefault(); setIndex(i => Math.max(0, i - 1)) }
          if (event.key === 'Enter') { event.preventDefault(); if (open && options[index]) select(options[index]); else if (!input) onClear(); else { setOpen(true); setDebounced(input) } }
        }} />
      {(input || value) && <button type="button" className="jobs-icon-button" aria-label="Clear workplace area" onClick={() => { cancelLocation(); setNotice(''); inputElement.current?.setCustomValidity(''); setInput(''); setOpen(false); onClear() }}><X size={14} /></button>}
      <Button type="button" variant="ghost" size="icon-sm" className="jobs-locate" title={locating ? 'Finding your location' : 'Use my location'} aria-label={locating ? 'Finding your location' : 'Use my location'} disabled={locating} onClick={locate}><LocateFixed size={16} /></Button>
    </div>
    {open && input.length >= 2 && <div className="jobs-place-options" id={listId} role="listbox" aria-label="Matching cities">
      {(places.isFetching || debounced !== input) && <p role="status">Searching places…</p>}
      {places.isError && <p role="alert">Place search failed. Try again.</p>}
      {!places.isFetching && debounced === input && !places.isError && options.length === 0 && <p>No matching city. Try a larger nearby city.</p>}
      {options.map((place, i) => <button type="button" role="option" aria-selected={index === i} id={`${listId}-${i}`} key={place.id} onPointerMove={() => setIndex(i)} onClick={() => select(place)}><span>{place.name}</span><small>{place.country}</small></button>)}
    </div>}
    {locating && <p className="jobs-field-status" role="status">Finding your location…</p>}
    {notice && <p className="jobs-field-status" role="status">{notice}</p>}
    {error && <p className="jobs-field-error" role="alert">{error}</p>}
  </div>
}
