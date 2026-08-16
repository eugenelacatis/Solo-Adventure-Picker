import { useState, useEffect, useMemo } from 'react'
import { MapContainer, TileLayer, Marker, Popup, GeoJSON } from 'react-leaflet'
import L from 'leaflet'
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png'
import markerIcon from 'leaflet/dist/images/marker-icon.png'
import markerShadow from 'leaflet/dist/images/marker-shadow.png'
import { API_BASE } from '../api.ts'
import { regions } from '../data/regions.ts'
import HudHeader from '../components/HudHeader.tsx'
import { bufferTrail } from '../utils/trailGeometry.ts'
import { startTracking, stopTracking, isTracking } from '../native/trailTracker.ts'
import type { VisitedEntry, XpResponse, TrailPoint, Adventure } from '../types.ts'
import 'leaflet/dist/leaflet.css'
import './MapPage.css'

const defaultIcon = L.icon({
  iconUrl: markerIcon,
  iconRetinaUrl: markerIcon2x,
  shadowUrl: markerShadow,
  iconSize: [25, 41],
  iconAnchor: [12, 41],
})

const REVEAL_RADIUS_KM = 5
const DEFAULT_CENTER: [number, number] = [37.5, -122.2] // Bay Area

function MapPage() {
  const [visited, setVisited] = useState<VisitedEntry[]>([])
  const [xp, setXp] = useState<XpResponse | null>(null)
  const [trail, setTrail] = useState<TrailPoint[]>([])
  const [tracking, setTracking] = useState(isTracking())
  const [newlyVisited, setNewlyVisited] = useState<Adventure[]>([])

  const toggleTracking = () => {
    if (tracking) {
      stopTracking()
    } else {
      startTracking(adventures => setNewlyVisited(adventures))
    }
    setTracking(!tracking)
  }

  useEffect(() => {
    fetch(`${API_BASE}/visited`, { credentials: 'include' })
      .then(res => res.json())
      .then(setVisited)
      .catch(() => setVisited([]))

    fetch(`${API_BASE}/xp`, { credentials: 'include' })
      .then(res => (res.ok ? res.json() : null))
      .then(setXp)
      .catch(() => setXp(null))

    fetch(`${API_BASE}/trail`, { credentials: 'include' })
      .then(res => (res.ok ? res.json() : []))
      .then(setTrail)
      .catch(() => setTrail([]))
  }, [])

  const fogReveal = useMemo(() => bufferTrail(trail, REVEAL_RADIUS_KM * 1000), [trail])

  return (
    <div className="map-page">
      <h1>Your Explored World</h1>
      <HudHeader xp={xp} />
      <button
        type="button"
        className="trail-tracking-toggle"
        onClick={toggleTracking}
        data-testid="trail-tracking-toggle"
      >
        {tracking ? 'Stop Trail Tracking' : 'Start Trail Tracking'}
      </button>
      {newlyVisited.length > 0 && (
        <div className="trail-discovery-notice" data-testid="trail-discovery-notice">
          New discovery! {newlyVisited.map(a => a.name).join(', ')}
        </div>
      )}

      <ul className="region-progress">
        {regions.map(r => {
          const count = visited.filter(v => v.region === r.value).length
          return (
            <li key={r.value} className="region-progress-item">
              <span className="region-progress-label">{r.label}</span>
              <span className="region-progress-count">{count} visited</span>
            </li>
          )
        })}
      </ul>

      <MapContainer
        center={DEFAULT_CENTER}
        zoom={9}
        className="map-container"
        data-testid="map-container"
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        {fogReveal && (
          <GeoJSON
            key={trail.length}
            data={fogReveal}
            pathOptions={{ color: '#1c7ed6', fillOpacity: 0.05 }}
          />
        )}
        {visited.map((v, i) => (
          <Marker
            key={`marker-${i}`}
            position={[v.lat, v.lng]}
            icon={defaultIcon}
            data-testid="visited-marker"
          >
            <Popup>Visited: {v.name || v.adventureId}</Popup>
          </Marker>
        ))}
      </MapContainer>
    </div>
  )
}

export default MapPage
