import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { API_BASE } from '../api.ts'
import { useAuth } from '../context/AuthContext.tsx'
import type { Achievement, QuestStatus, XpResponse } from '../types.ts'
import './HudHeader.css'

interface HudHeaderProps {
  xp: XpResponse | null
}

function HudHeader({ xp }: HudHeaderProps) {
  const { logout } = useAuth()
  const [achievements, setAchievements] = useState<Achievement[]>([])
  const [quest, setQuest] = useState<QuestStatus | null>(null)

  useEffect(() => {
    fetch(`${API_BASE}/achievements`, { credentials: 'include' })
      .then(res => (res.ok ? res.json() : []))
      .then(data => setAchievements(Array.isArray(data) ? data : []))
      .catch(() => setAchievements([]))

    fetch(`${API_BASE}/quest`, { credentials: 'include' })
      .then(res => (res.ok ? res.json() : null))
      .then(setQuest)
      .catch(() => setQuest(null))
  }, [xp?.totalXp])

  const progressPercent = xp && xp.nextLevelXp > 0
    ? Math.min(100, Math.round((xp.totalXp / xp.nextLevelXp) * 100))
    : 0

  return (
    <div className="hud-header">
      <nav className="hud-nav">
        <Link to="/adventure">Adventure</Link>
        <Link to="/map">Map</Link>
        <Link to="/history">History</Link>
      </nav>

      <div className="hud-xp">
        <span className="hud-level">Level {xp?.level ?? 1}</span>
        <div className="hud-progress-bar">
          <div className="hud-progress-fill" style={{ width: `${progressPercent}%` }} />
        </div>
        <span className="hud-xp-label">{xp?.totalXp ?? 0} / {xp?.nextLevelXp ?? 100} XP</span>
      </div>

      {quest && (
        <div className={`hud-quest ${quest.completedToday ? 'hud-quest-done' : ''}`}>
          {quest.completedToday ? 'Daily quest complete!' : quest.description}
        </div>
      )}

      {achievements.length > 0 && (
        <div className="hud-achievements">
          {achievements.map(a => (
            <span key={a.id} className="hud-achievement-badge" title={a.name}>{a.name}</span>
          ))}
        </div>
      )}

      {/* Styling only — gear/perks aren't earnable yet. */}
      <div className="hud-gear">
        <span className="hud-gear-badge">Trail Boots</span>
      </div>

      <button className="logout-btn" onClick={() => logout()}>Log Out</button>
    </div>
  )
}

export default HudHeader
