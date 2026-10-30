import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import Map, { Layer, NavigationControl, Popup, Source, type MapLayerMouseEvent, type MapRef } from 'react-map-gl/maplibre'
import { setWorkerUrl, type GeoJSONSource, type LayerSpecification } from 'maplibre-gl'
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url'
import { MapPin, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { jobFeatures } from '@/lib/jobs'
import { defaultJobMapCamera, jobMapCameraTarget, readJobMapViewport, sameJobMapCamera, type JobMapCamera, type JobMapViewport } from '@/lib/jobMapViewport'
import type { Bounds, Job } from '@/types/job'
import 'maplibre-gl/dist/maplibre-gl.css'

// MapLibre 6 requires an explicit bundled worker URL in Vite (dev and build).
setWorkerUrl(workerUrl)

type Props = {
  jobs: Job[]
  selected: string | null
  center?: JobMapCamera
  bounds?: Bounds
  onSelect: (id: string) => void
  onViewportChange: (viewport: JobMapViewport) => void
}

function tokenColor(token: string) {
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = 1
  const ctx = canvas.getContext('2d')!
  ctx.fillStyle = getComputedStyle(document.documentElement).getPropertyValue(token).trim()
  ctx.fillRect(0, 0, 1, 1)
  const [r, g, b, a] = ctx.getImageData(0, 0, 1, 1).data
  return `rgba(${r},${g},${b},${a / 255})`
}

export default function JobMap({ jobs, selected, center, bounds, onSelect, onViewportChange }: Props) {
  const map = useRef<MapRef>(null)
  const viewportTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const viewportCallback = useRef(onViewportChange)
  const [failed, setFailed] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [revision, setRevision] = useState(0)
  const [clusterJobs, setClusterJobs] = useState<{ longitude: number; latitude: number; jobs: Job[] } | null>(null)
  const [palette, setPalette] = useState(() => ({ ink: tokenColor('--color-ink'), paper: tokenColor('--color-cotton'), accent: tokenColor('--color-signal'), dark: document.documentElement.classList.contains('dark') }))
  const data = useMemo(() => {
    const collection = jobFeatures(jobs, bounds)
    return { ...collection, features: collection.features.map(feature => ({ ...feature, properties: { ...feature.properties, selected: feature.properties.jobId === selected ? 1 : 0 } })) }
  }, [jobs, bounds, selected])
  const pointCount = data.features.length

  useEffect(() => { viewportCallback.current = onViewportChange }, [onViewportChange])
  const cancelViewportSearch = useCallback(() => {
    clearTimeout(viewportTimer.current)
    viewportTimer.current = undefined
  }, [])
  const scheduleViewportSearch = useCallback(() => {
    cancelViewportSearch()
    viewportTimer.current = setTimeout(() => {
      viewportTimer.current = undefined
      const instance = map.current
      if (instance && !instance.isMoving()) viewportCallback.current(readJobMapViewport(instance))
    }, 400)
  }, [cancelViewportSearch])
  useEffect(() => cancelViewportSearch, [cancelViewportSearch])

  useEffect(() => {
    const observer = new MutationObserver(() => setPalette({ ink: tokenColor('--color-ink'), paper: tokenColor('--color-cotton'), accent: tokenColor('--color-signal'), dark: document.documentElement.classList.contains('dark') }))
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
    return () => observer.disconnect()
  }, [])
  useEffect(() => {
    if (loaded || failed) return
    const timeout = setTimeout(() => setFailed(true), 30_000)
    return () => clearTimeout(timeout)
  }, [loaded, failed, revision])
  useEffect(() => {
    const instance = map.current
    if (!loaded || !instance) return
    const target = center ?? defaultJobMapCamera
    const actual = readJobMapViewport(instance).camera
    if (sameJobMapCamera(actual, target)) return
    cancelViewportSearch()
    instance.jumpTo(jobMapCameraTarget(target))
    scheduleViewportSearch()
  }, [center, loaded, cancelViewportSearch, scheduleViewportSearch])

  const cluster: LayerSpecification = { id: 'job-clusters', type: 'circle', source: 'jobs', filter: ['has', 'point_count'], paint: { 'circle-color': ['case', ['>', ['get', 'selected'], 0], palette.accent, palette.ink], 'circle-radius': ['step', ['get', 'point_count'], 17, 10, 22, 50, 28], 'circle-stroke-color': palette.paper, 'circle-stroke-width': 2 } }
  const count: LayerSpecification = { id: 'job-cluster-count', type: 'symbol', source: 'jobs', filter: ['has', 'point_count'], layout: { 'text-field': ['get', 'point_count_abbreviated'], 'text-font': ['Noto Sans Regular'], 'text-size': 12 }, paint: { 'text-color': palette.paper } }
  const points: LayerSpecification = { id: 'job-points', type: 'circle', source: 'jobs', filter: ['!', ['has', 'point_count']], paint: { 'circle-color': ['case', ['==', ['get', 'jobId'], selected ?? ''], palette.accent, palette.ink], 'circle-radius': ['case', ['==', ['get', 'jobId'], selected ?? ''], 9, 6], 'circle-stroke-color': palette.paper, 'circle-stroke-width': 2 } }

  async function click(event: MapLayerMouseEvent) {
    const feature = event.features?.[0]
    if (!feature) { setClusterJobs(null); return }
    if (feature.properties?.jobId) { onSelect(String(feature.properties.jobId)); return }
    const source = map.current?.getSource('jobs') as GeoJSONSource | undefined
    const id = feature.properties?.cluster_id
    if (!source || id === undefined || feature.geometry.type !== 'Point') return
    try {
      const leaves = await source.getClusterLeaves(Number(id), 1000, 0)
      const ids = new Set(leaves.map(f => String(f.properties?.jobId)))
      const matches = jobs.filter(j => ids.has(j.id))
      const [longitude, latitude] = feature.geometry.coordinates
      setClusterJobs({ longitude, latitude, jobs: matches })
    } catch { /* A source can be replaced while a click is being processed. */ }
  }

  return <section className="jobs-map" data-ready={loaded} aria-busy={!loaded && !failed} aria-label="Map of workplaces from the loaded job results">
    {!failed && <Map key={revision} ref={map} initialViewState={center ?? defaultJobMapCamera} mapStyle={`https://tiles.openfreemap.org/styles/${palette.dark ? 'dark' : 'positron'}`} maxZoom={17} minZoom={1} renderWorldCopies={false}
      interactiveLayerIds={['job-clusters', 'job-points']} cursor="pointer"
      onLoad={() => { setLoaded(true); scheduleViewportSearch() }}
      onMoveStart={() => { cancelViewportSearch(); setClusterJobs(null) }} onMoveEnd={scheduleViewportSearch} onResize={scheduleViewportSearch}
      onClick={click} onError={event => { if (/webgl|context|style|failed to fetch|worker failed/i.test(event.error.message) && !loaded) setFailed(true) }}>
      <Source id="jobs" type="geojson" data={data} cluster clusterMaxZoom={17} clusterRadius={45} clusterProperties={{ selected: ['max', ['get', 'selected']] }}>
        <Layer {...cluster} /><Layer {...count} /><Layer {...points} />
      </Source>
      <NavigationControl position="bottom-right" showCompass={false} />
      {clusterJobs && <Popup longitude={clusterJobs.longitude} latitude={clusterJobs.latitude} onClose={() => setClusterJobs(null)} closeOnClick={false} maxWidth="300px">
        <div className="jobs-cluster-list"><p>Openings around this location</p>{clusterJobs.jobs.map(job => <button key={job.id} onClick={() => { onSelect(job.id); setClusterJobs(null) }}><strong>{job.title}</strong><span>{job.company}</span></button>)}</div>
      </Popup>}
    </Map>}
    {failed && <div className="jobs-map-fallback"><MapPin size={28} /><h3>The map could not load</h3><p>You can still browse and save every job in the list.</p><Button variant="outline" onClick={() => { setLoaded(false); setFailed(false); setRevision(n => n + 1) }}><RotateCcw size={16} />Retry map</Button></div>}
    {!failed && !loaded && <p className="jobs-map-status" role="status">Loading the basemap…</p>}
    {!failed && <div className="jobs-map-caption"><MapPin size={14} /><span>{pointCount} workplace {pointCount === 1 ? 'marker' : 'markers'} from this page. City positions are approximate.</span></div>}
  </section>
}
