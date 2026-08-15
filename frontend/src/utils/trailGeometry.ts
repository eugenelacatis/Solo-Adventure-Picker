import { lineString, point } from '@turf/helpers'
import buffer from '@turf/buffer'
import type { Feature, Polygon, MultiPolygon } from 'geojson'
import type { TrailPoint } from '../types'

// bufferTrail turns a chronological list of trail points into one buffered
// polygon covering the ground the trail passed near, for the dashboard's
// fog-reveal layer. A single point still produces a buffered circle (a
// lineString needs 2+ positions), so MapPage can render a reveal even
// before a second GPS sample has landed.
export function bufferTrail(
  points: TrailPoint[],
  bufferMeters: number
): Feature<Polygon | MultiPolygon> | null {
  if (points.length === 0) {
    return null
  }

  const geometry =
    points.length === 1
      ? point([points[0].lng, points[0].lat])
      : lineString(points.map(p => [p.lng, p.lat]))

  return buffer(geometry, bufferMeters, { units: 'meters' })
}
