import type { Bounds } from '@/types/job'
import { parseBounds } from '@/lib/jobs'

export interface JobMapCamera { latitude: number; longitude: number; zoom: number }
export interface JobMapViewport { bounds: Bounds; camera: JobMapCamera }

export const defaultJobMapCamera: JobMapCamera = { longitude: 20, latitude: 22, zoom: 1.5 }

export function wrapLongitude(value: number): number {
  return ((value + 180) % 360 + 360) % 360 - 180
}

// React initialViewState uses latitude/longitude; the native camera API uses
// center: [longitude, latitude]. Passing the former to jumpTo only changes zoom.
export function jobMapCameraTarget(camera: JobMapCamera): { center: [number, number]; zoom: number } {
  return { center: [camera.longitude, camera.latitude], zoom: camera.zoom }
}

export function sameJobMapCamera(a: JobMapCamera, b: JobMapCamera): boolean {
  return Math.abs(a.latitude - b.latitude) <= 0.00001
    && Math.abs(wrapLongitude(a.longitude - b.longitude)) <= 0.00001
    && Math.abs(a.zoom - b.zoom) <= 0.01
}

export function readJobMapViewport(map: {
  getBounds: () => { getWest(): number; getSouth(): number; getEast(): number; getNorth(): number }
  getCenter: () => { lat: number; lng: number }
  getZoom: () => number
}): JobMapViewport {
  const b = map.getBounds(), c = map.getCenter()
  const world = b.getEast() - b.getWest() >= 360
  return {
    bounds: [world ? -180 : wrapLongitude(b.getWest()), Math.max(-90, b.getSouth()), world ? 180 : wrapLongitude(b.getEast()), Math.min(90, b.getNorth())],
    camera: { longitude: wrapLongitude(c.lng), latitude: c.lat, zoom: map.getZoom() },
  }
}

// Map movement replaces the current history entry. Numeric tolerances prevent
// URL rounding from causing camera jumps and another round of search requests.
export function jobMapViewportParams(previous: URLSearchParams, viewport: JobMapViewport): URLSearchParams {
  const oldBounds = parseBounds(previous.get('bounds'))
  const oldCamera = {
    latitude: Number(previous.get('lat')),
    longitude: Number(previous.get('lng')),
    zoom: Number(previous.get('zoom') ?? defaultJobMapCamera.zoom),
  }
  const hadCamera = previous.has('lat') && previous.has('lng')
  const sameCamera = hadCamera && sameJobMapCamera(oldCamera, viewport.camera)
  const sameBounds = oldBounds?.every((value, i) => Math.abs(value - viewport.bounds[i]) <= 0.00001)
  if (sameCamera && sameBounds) return previous

  const next = new URLSearchParams(previous)
  next.set('bounds', viewport.bounds.map(n => n.toFixed(5)).join(','))
  next.set('lat', viewport.camera.latitude.toFixed(5))
  next.set('lng', viewport.camera.longitude.toFixed(5))
  next.set('zoom', viewport.camera.zoom.toFixed(2))
  next.delete('page')
  // A GPS/city jump is already reflected in the URL. A changed camera here
  // means the user explored another area; the remote preference is independent.
  if (hadCamera && !sameCamera) {
    next.delete('place')
    next.delete('country')
    next.delete('job')
  }
  return next
}
