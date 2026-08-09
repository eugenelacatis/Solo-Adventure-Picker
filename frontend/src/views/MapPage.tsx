import { useState, useEffect } from 'react'
import { MapContainer, TileLayer, Marker, Popup, Circle } from 'react-leaflet'
import L from 'leaflet'
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png'
import markerIcon from 'leaflet/dist/images/marker-icon.png'
import markerShadow from 'leaflet/dist/images/marker-shadow.png'
import type { VisitedEntry } from '../types.ts'
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

  useEffect(() => {
    fetch('http://localhost:8080/visited', { credentials: 'include' })
      .then(res => res.json())
      .then(setVisited)
      .catch(() => setVisited([]))
  }, [])

  return (
    <div className="map-page">
      <h1>Your Explored World</h1>
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
        {visited.map((v, i) => (
          <Circle
            key={`fog-${i}`}
            center={[v.lat, v.lng]}
            radius={REVEAL_RADIUS_KM * 1000}
            pathOptions={{ color: '#646cff', fillOpacity: 0.05 }}
          />
        ))}
        {visited.map((v, i) => (
          <Marker
            key={`marker-${i}`}
            position={[v.lat, v.lng]}
            icon={defaultIcon}
            data-testid="visited-marker"
          >
            <Popup>Visited: {v.adventureId}</Popup>
          </Marker>
        ))}
      </MapContainer>
    </div>
  )
}

export default MapPage
