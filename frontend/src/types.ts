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
  region?: string
  lat: number
  lng: number
  createdAt: string
}

export interface JournalEntryView {
  adventureId: string
  adventureName?: string
  text: string
  createdAt: string
}

export interface Achievement {
  id: string
  name: string
}

export interface RerollStatus {
  rerollTokens: number
  rerollResetAt: string
}

export interface QuestStatus {
  description: string
  completedToday: boolean
}

export interface ApiError {
  error: string
  code?: number
  details?: string
}

export interface TrailPoint {
  lat: number
  lng: number
  recordedAt: string
}

export interface TrailUploadResponse {
  newlyVisited: Adventure[]
}
