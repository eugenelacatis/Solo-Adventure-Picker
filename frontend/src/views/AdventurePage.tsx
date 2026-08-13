import { useState, useEffect, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { API_BASE } from '../api.ts'
import { regions } from '../data/regions.ts'
import { adventureTypes, adventureEfforts } from '../data/adventureFilters.ts'
import { capitalizeWords } from '../utils/formatting.ts'
import HudHeader from '../components/HudHeader.tsx'
import type { Adventure, XpResponse, RerollStatus } from '../types.ts'
import './AdventurePage.css'

function AdventurePage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [adventure, setAdventure] = useState<Adventure | null>(null)
  const [errorMsg, setErrorMsg] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isExpanded, setIsExpanded] = useState(false)
  const [journalText, setJournalText] = useState('')
  const [journalSaved, setJournalSaved] = useState(false)
  const [isSavingJournal, setIsSavingJournal] = useState(false)
  const [visitSaved, setVisitSaved] = useState(false)
  const [isSavingVisit, setIsSavingVisit] = useState(false)
  const [xp, setXp] = useState<XpResponse | null>(null)
  const [rerollStatus, setRerollStatus] = useState<RerollStatus | null>(null)

  const region = searchParams.get('region') || ''
  const type = searchParams.get('type') || ''
  const effort = searchParams.get('effort') || ''

  const rerollsLeft = rerollStatus?.rerollTokens ?? null
  const outOfRerolls = rerollsLeft === 0

  const buttonText = useMemo(() => {
    if (isLoading) return 'Finding Adventure...'
    if (outOfRerolls) return 'No Rerolls Left Today'
    return 'Pick Another Adventure'
  }, [isLoading, outOfRerolls])

  const displayName = useMemo(() =>
    capitalizeWords(adventure?.name), [adventure?.name]
  )

  const displayType = useMemo(() =>
    capitalizeWords(adventure?.type), [adventure?.type]
  )

  const fetchXp = async () => {
    try {
      const res = await fetch(`${API_BASE}/xp`, { credentials: 'include' })
      if (res.ok) setXp(await res.json())
    } catch {
      // HUD simply shows stale/default values if this fails.
    }
  }

  const fetchRerollStatus = async () => {
    try {
      const res = await fetch(`${API_BASE}/reroll-status`, { credentials: 'include' })
      if (res.ok) setRerollStatus(await res.json())
    } catch {
      // Reroll button simply shows stale/default values if this fails.
    }
  }

  const getRandomAdventure = async () => {
    if (isLoading) return

    setIsLoading(true)
    setIsExpanded(false)
    setJournalText('')
    setJournalSaved(false)
    setVisitSaved(false)

    try {
      const params = new URLSearchParams()
      if (region) params.set('region', region)
      if (type) params.set('type', type)
      if (effort) params.set('effort', effort)

      const res = await fetch(`${API_BASE}/random?${params.toString()}`, { credentials: 'include' })

      if (!res.ok) {
        const errorJson = await res.json()
        throw new Error(errorJson.details || errorJson.error || 'Something went wrong')
      }

      const data: Adventure = await res.json()
      setAdventure(data)
      setErrorMsg('')
      fetchRerollStatus()
    } catch (err) {
      setAdventure(null)
      setErrorMsg(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setIsLoading(false)
    }
  }

  const submitJournalEntry = async () => {
    if (!journalText.trim() || !adventure) return

    setIsSavingJournal(true)
    try {
      const res = await fetch(`${API_BASE}/journal`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ adventureId: String(adventure.id), text: journalText }),
      })

      if (!res.ok) {
        const errorJson = await res.json()
        throw new Error(errorJson.details || errorJson.error || 'Something went wrong')
      }

      setXp(await res.json())
      setJournalSaved(true)
      setErrorMsg('')
      fetchRerollStatus()
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setIsSavingJournal(false)
    }
  }

  const markAsVisited = async () => {
    if (!adventure) return

    setIsSavingVisit(true)
    try {
      const res = await fetch(`${API_BASE}/visited`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ adventureId: String(adventure.id) }),
      })

      if (!res.ok) {
        const errorJson = await res.json()
        throw new Error(errorJson.details || errorJson.error || 'Something went wrong')
      }

      setXp(await res.json())
      setVisitSaved(true)
      setErrorMsg('')
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setIsSavingVisit(false)
    }
  }

  useEffect(() => {
    fetchXp()
    fetchRerollStatus()
    getRandomAdventure()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div id="app">
      <h1>Solo Adventure Picker</h1>
      <HudHeader xp={xp} />

      <select
        value={region}
        onChange={(e) => {
          const next = new URLSearchParams(searchParams)
          if (e.target.value) next.set('region', e.target.value)
          else next.delete('region')
          setSearchParams(next)
        }}
      >
        <option value="">All Regions</option>
        {regions.map(r => (
          <option key={r.value} value={r.value}>{r.label}</option>
        ))}
      </select>

      <select
        value={type}
        onChange={(e) => {
          const next = new URLSearchParams(searchParams)
          if (e.target.value) next.set('type', e.target.value)
          else next.delete('type')
          setSearchParams(next)
        }}
      >
        <option value="">All Types</option>
        {adventureTypes.map(t => (
          <option key={t.value} value={t.value}>{t.label}</option>
        ))}
      </select>

      <select
        value={effort}
        onChange={(e) => {
          const next = new URLSearchParams(searchParams)
          if (e.target.value) next.set('effort', e.target.value)
          else next.delete('effort')
          setSearchParams(next)
        }}
      >
        <option value="">All Efforts</option>
        {adventureEfforts.map(ef => (
          <option key={ef.value} value={ef.value}>{ef.label}</option>
        ))}
      </select>

      {adventure && (
        <div className="card" key={adventure.name}>
          <div className="xp-badge">+{adventure.xpValue || 100}XP</div>
          <div className="card-content">
            <h2>{displayName}</h2>
            <h3>{displayType}</h3>

            <div className={`card-details ${isExpanded ? 'expanded' : ''}`}>
              <p className="description">
                {adventure.description || 'Embark on this mysterious adventure!'}
              </p>

              <div className="journal-entry">
                <textarea
                  placeholder="Write about your adventure..."
                  value={journalText}
                  onChange={(e) => setJournalText(e.target.value)}
                />
                <button
                  onClick={submitJournalEntry}
                  disabled={isSavingJournal || !journalText.trim()}
                >
                  Save Journal Entry
                </button>
                {journalSaved && <p className="journal-confirmation">Journal entry saved!</p>}
              </div>
            </div>

            <button
              className="toggle-details"
              onClick={() => setIsExpanded(!isExpanded)}
            >
              {isExpanded ? 'Show Less' : 'Show More'}
            </button>

            <div className="visit-tracker">
              <button
                className="mark-visited-btn"
                onClick={markAsVisited}
                disabled={isSavingVisit || visitSaved}
              >
                {visitSaved ? 'Already Visited' : 'Mark as Visited'}
              </button>
              {visitSaved && (
                <p className="visit-confirmation">
                  {xp?.alreadyVisited ? 'Already visited before — no new XP.' : 'Marked as visited!'}
                </p>
              )}
            </div>
          </div>
        </div>
      )}

      {errorMsg && !adventure && (
        <div className="error card">
          {errorMsg}
        </div>
      )}

      <div className="reroll-btn-wrapper">
        <button
          className="reroll-btn"
          onClick={getRandomAdventure}
          disabled={isLoading || outOfRerolls}
        >
          {buttonText}
        </button>
        {rerollsLeft !== null && (
          <p className="reroll-count">{rerollsLeft} reroll{rerollsLeft === 1 ? '' : 's'} left today</p>
        )}
      </div>
    </div>
  )
}

export default AdventurePage
