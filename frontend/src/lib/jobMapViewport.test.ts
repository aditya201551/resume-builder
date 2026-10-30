import { describe, expect, it } from 'vitest'
import { jobMapCameraTarget, jobMapViewportParams, readJobMapViewport, sameJobMapCamera } from './jobMapViewport'
import { getJobMapPosition } from './jobMapLocation'

describe('location and automatic viewport search', () => {
  it('moves the native map to Bengaluru, using longitude/latitude as a center pair', () => {
    expect(jobMapCameraTarget({ latitude: 12.972, longitude: 77.595, zoom: 10 })).toEqual({ center: [77.595, 12.972], zoom: 10 })
  })
  it('requests a fresh high-accuracy position and retains device coordinates', async () => {
    let options: PositionOptions | undefined
    const geolocation = { getCurrentPosition: (success: PositionCallback, _failure?: PositionErrorCallback | null, input?: PositionOptions) => {
      options = input
      success({ coords: { latitude: 12.9716, longitude: 77.5946, accuracy: 25 } } as GeolocationPosition)
    } }
    await expect(getJobMapPosition(geolocation)).resolves.toEqual({ latitude: 12.972, longitude: 77.595, accuracy: 25 })
    expect(options).toMatchObject({ enableHighAccuracy: true, maximumAge: 0 })
  })
  it('rejects invalid coordinates and propagates permission failures', async () => {
    await expect(getJobMapPosition({ getCurrentPosition: success => success({ coords: { latitude: NaN, longitude: 77, accuracy: 1 } } as GeolocationPosition) })).rejects.toThrow()
    const denied = { code: 1 }
    await expect(getJobMapPosition({ getCurrentPosition: (_success, failure) => failure!(denied as GeolocationPositionError) })).rejects.toBe(denied)
  })
  it('normalizes actual map bounds across the dateline', () => {
    const viewport = readJobMapViewport({ getBounds: () => ({ getWest: () => 170, getEast: () => 190, getNorth: () => 10, getSouth: () => -10 }), getCenter: () => ({ lng: 180, lat: 0 }), getZoom: () => 4 })
    expect(viewport.bounds).toEqual([170, -10, -170, 10])
    expect(viewport.camera.longitude).toBe(-180)
  })
  it('updates the focused window and resets pagination while retaining remote preferences', () => {
    const before = new URLSearchParams('q=backend&remoteCountry=IN&includeUnknown=true&country=DE&place=Berlin&lat=52&lng=13&zoom=9&page=3&job=old')
    const next = jobMapViewportParams(before, { bounds: [77.4, 12.8, 77.8, 13.1], camera: { longitude: 77.6, latitude: 12.97, zoom: 10 } })
    expect(next.get('remoteCountry')).toBe('IN')
    expect(next.get('includeUnknown')).toBe('true')
    expect(next.get('q')).toBe('backend')
    expect(next.get('bounds')).toBe('77.40000,12.80000,77.80000,13.10000')
    for (const key of ['country', 'place', 'page', 'job']) expect(next.has(key)).toBe(false)
    expect(before.get('place')).toBe('Berlin')
  })
  it('preserves a just-selected city and avoids rounding feedback loops', () => {
    const camera = { latitude: 12.972, longitude: 77.595, zoom: 10 }
    const before = new URLSearchParams('lat=12.972&lng=77.595&zoom=10&place=Bengaluru&country=IN')
    const viewport = { bounds: [77.4, 12.8, 77.8, 13.1] as [number, number, number, number], camera }
    const next = jobMapViewportParams(before, viewport)
    expect(next.get('place')).toBe('Bengaluru')
    expect(next.get('country')).toBe('IN')
    expect(jobMapViewportParams(next, viewport)).toBe(next)
    expect(sameJobMapCamera(camera, { ...camera, longitude: camera.longitude + 0.000001 })).toBe(true)
  })
  it('keeps an explicit country filter when first initializing an unpositioned map', () => {
    const next = jobMapViewportParams(new URLSearchParams('country=DE'), { bounds: [-180, -85, 180, 85], camera: { longitude: 20, latitude: 22, zoom: 1.5 } })
    expect(next.get('country')).toBe('DE')
  })
})
