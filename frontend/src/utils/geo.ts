const EARTH_RADIUS_KM = 6371

export interface LatLng {
  lat: number
  lng: number
}

function toRadians(deg: number): number {
  return (deg * Math.PI) / 180
}

export function distanceKm(a: LatLng, b: LatLng): number {
  const dLat = toRadians(b.lat - a.lat)
  const dLng = toRadians(b.lng - a.lng)
  const lat1 = toRadians(a.lat)
  const lat2 = toRadians(b.lat)

  const h =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLng / 2) ** 2

  return 2 * EARTH_RADIUS_KM * Math.asin(Math.sqrt(h))
}

export function isWithinRevealRadius(
  lat: number,
  lng: number,
  visited: LatLng[],
  radiusKm: number
): boolean {
  return visited.some(v => distanceKm({ lat, lng }, v) <= radiusKm)
}
