export const jobMapPositionOptions: PositionOptions = { enableHighAccuracy: true, maximumAge: 0, timeout: 20_000 }

export function getJobMapPosition(geolocation: Pick<Geolocation, 'getCurrentPosition'>): Promise<{ latitude: number; longitude: number; accuracy: number }> {
  return new Promise((resolve, reject) => geolocation.getCurrentPosition(position => {
    const { latitude, longitude, accuracy } = position.coords
    if (!Number.isFinite(latitude) || !Number.isFinite(longitude) || latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180) {
      reject(new Error('Invalid device coordinates'))
      return
    }
    resolve({ latitude: Number(latitude.toFixed(3)), longitude: Number(longitude.toFixed(3)), accuracy })
  }, reject, jobMapPositionOptions))
}
