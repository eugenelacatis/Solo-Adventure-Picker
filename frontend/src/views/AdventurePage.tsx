import { useState, useEffect, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { capitalizeWords } from '../utils/formatting.ts'
import { useAuth } from '../context/AuthContext.tsx'
import type { Adventure } from '../types.ts'
import './AdventurePage.css'

function AdventurePage() {
  const { logout } = useAuth()
  const [searchParams] = useSearchParams()
  const [adventure, setAdventure] = useState<Adventure | null>(null)
  const [errorMsg, setErrorMsg] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isExpanded, setIsExpanded] = useState(false)
  const [journalText, setJournalText] = useState('')
  const [journalSaved, setJournalSaved] = useState(false)
  const [isSavingJournal, setIsSavingJournal] = useState(false)
  const [visitSaved, setVisitSaved] = useState(false)
  const [isSavingVisit, setIsSavingVisit] = useState(false)

  const region = searchParams.get('region') || ''

  const buttonText = useMemo(() => {
    if (isLoading) return 'Finding Adventure...'
    return 'Pick Another Adventure'
  }, [isLoading])

  const displayName = useMemo(() =>
    capitalizeWords(adventure?.name), [adventure?.name]
  )

  const displayType = useMemo(() =>
    capitalizeWords(adventure?.type), [adventure?.type]
  )

  const getRandomAdventure = async () => {
    if (isLoading) return

    setIsLoading(true)
    setIsExpanded(false)
    setJournalText('')
    setJournalSaved(false)
    setVisitSaved(false)

    try {
      const res = await fetch(`http://localhost:8080/random?region=${region}`)

      if (!res.ok) {
        const errorJson = await res.json()
        throw new Error(errorJson.details || errorJson.error || 'Something went wrong')
      }

      const data: Adventure = await res.json()
      setAdventure(data)
      setErrorMsg('')
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
      const res = await fetch('http://localhost:8080/journal', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ adventureId: String(adventure.id), text: journalText }),
      })

      if (!res.ok) {
        const errorJson = await res.json()
        throw new Error(errorJson.details || errorJson.error || 'Something went wrong')
      }

      await res.json()
      setJournalSaved(true)
      setErrorMsg('')
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
      const res = await fetch('http://localhost:8080/xp/add', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          adventureId: String(adventure.id),
          xp: adventure.xpValue || 100,
          lat: adventure.lat,
          lng: adventure.lng,
        }),
      })

      if (!res.ok) {
        const errorJson = await res.json()
        throw new Error(errorJson.details || errorJson.error || 'Something went wrong')
      }

      await res.json()
      setVisitSaved(true)
      setErrorMsg('')
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setIsSavingVisit(false)
    }
  }

  useEffect(() => {
    getRandomAdventure()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div id="app">
      <h1>Solo Adventure Picker</h1>
      <button className="logout-btn" onClick={() => logout()}>Log Out</button>

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
                {visitSaved ? 'Marked as Visited' : 'Mark as Visited'}
              </button>
              {visitSaved && <p className="visit-confirmation">Marked as visited!</p>}
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
          disabled={isLoading}
        >
          {buttonText}
        </button>
      </div>
    </div>
  )
}

export default AdventurePage
