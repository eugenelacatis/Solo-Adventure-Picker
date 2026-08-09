import type { Adventure } from '../types.ts'

// Real adventures from the seeded catalog (backend/scripts/seed.go), captured
// once and replayed for every anonymous visitor instead of calling /random
// live per-demo-view.
export const demoAdventures: Adventure[] = [
  {
    id: 2,
    name: 'Mount Tamalpais',
    type: 'hike',
    region: 'bay-area',
    xpValue: 150,
    lat: 37.9235,
    lng: -122.5965,
  },
  {
    id: 10,
    name: 'Big Basin Redwoods',
    type: 'hike',
    region: 'bay-area',
    xpValue: 150,
    lat: 37.1719,
    lng: -122.2245,
  },
  {
    id: 12,
    name: 'Point Reyes National Seashore',
    type: 'hike',
    region: 'north-bay',
    xpValue: 150,
    lat: 38.07,
    lng: -122.96,
  },
  {
    id: 13,
    name: 'Annadel State Park',
    type: 'hike',
    region: 'north-bay',
    xpValue: 150,
    lat: 38.4419,
    lng: -122.6142,
  },
  {
    id: 15,
    name: 'Rancho San Antonio Preserve',
    type: 'hike',
    region: 'south-bay',
    xpValue: 150,
    lat: 37.3327,
    lng: -122.0866,
  },
  {
    id: 14,
    name: 'Almaden Quicksilver County Park',
    type: 'hike',
    region: 'south-bay',
    xpValue: 150,
    lat: 37.1897,
    lng: -121.8244,
  },
  {
    id: 17,
    name: 'Redwood Regional Park',
    type: 'hike',
    region: 'east-bay',
    xpValue: 150,
    lat: 37.8158,
    lng: -122.1622,
  },
  {
    id: 18,
    name: 'Mission Peak Regional Preserve',
    type: 'hike',
    region: 'east-bay',
    xpValue: 150,
    lat: 37.5091,
    lng: -121.8807,
  },
]
