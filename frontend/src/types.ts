export interface Adventure {
  id: number
  name: string
  type?: string
  region: string
  scenery?: string
  effort?: string
  duration?: string
  description?: string
  xpValue?: number
  lat?: number
  lng?: number
}

export interface XpResponse {
  totalXp: number
  level: number
  nextLevelXp: number
  alreadyVisited: boolean
}

export interface VisitedEntry {
  adventureId: string
  name?: string
  lat: number
  lng: number
}

export interface Achievement {
  id: string
  name: string
}

export interface ApiError {
  error: string
  code?: number
  details?: string
}
