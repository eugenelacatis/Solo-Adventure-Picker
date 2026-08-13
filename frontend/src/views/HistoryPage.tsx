import { useState, useEffect } from 'react'
import { API_BASE } from '../api.ts'
import { capitalizeWords } from '../utils/formatting.ts'
import HudHeader from '../components/HudHeader.tsx'
import type { VisitedEntry, JournalEntryView, XpResponse } from '../types.ts'
import './HistoryPage.css'

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

function HistoryPage() {
  const [visited, setVisited] = useState<VisitedEntry[]>([])
  const [journalEntries, setJournalEntries] = useState<JournalEntryView[]>([])
  const [xp, setXp] = useState<XpResponse | null>(null)

  useEffect(() => {
    fetch(`${API_BASE}/visited`, { credentials: 'include' })
      .then(res => (res.ok ? res.json() : []))
      .then(data => setVisited(Array.isArray(data) ? data : []))
      .catch(() => setVisited([]))

    fetch(`${API_BASE}/journal`, { credentials: 'include' })
      .then(res => (res.ok ? res.json() : []))
      .then(data => setJournalEntries(Array.isArray(data) ? data : []))
      .catch(() => setJournalEntries([]))

    fetch(`${API_BASE}/xp`, { credentials: 'include' })
      .then(res => (res.ok ? res.json() : null))
      .then(setXp)
      .catch(() => setXp(null))
  }, [])

  return (
    <div className="history-page">
      <h1>Your Adventure History</h1>
      <HudHeader xp={xp} />

      <section className="history-section">
        <h2>Past Adventures</h2>
        {visited.length === 0 ? (
          <p className="history-empty">No adventures visited yet.</p>
        ) : (
          <ul className="history-list">
            {visited.map((v, i) => (
              <li key={`visited-${i}`} className="history-item">
                <span className="history-item-name">{capitalizeWords(v.name) || v.adventureId}</span>
                <span className="history-item-date">{formatDate(v.createdAt)}</span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="history-section">
        <h2>Journal Entries</h2>
        {journalEntries.length === 0 ? (
          <p className="history-empty">No journal entries yet.</p>
        ) : (
          <ul className="history-list">
            {journalEntries.map((entry, i) => (
              <li key={`journal-${i}`} className="history-item history-journal-item">
                <div className="history-journal-header">
                  <span className="history-item-name">{capitalizeWords(entry.adventureName) || entry.adventureId}</span>
                  <span className="history-item-date">{formatDate(entry.createdAt)}</span>
                </div>
                <p className="history-journal-text">{entry.text}</p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}

export default HistoryPage
