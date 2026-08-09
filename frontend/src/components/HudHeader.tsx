import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { API_BASE } from '../api.ts'
import { useAuth } from '../context/AuthContext.tsx'
import type { Achievement, XpResponse } from '../types.ts'
import './HudHeader.css'

interface HudHeaderProps {
  xp: XpResponse | null
}

function HudHeader({ xp }: HudHeaderProps) {
  const { logout } = useAuth()
  const [achievements, setAchievements] = useState<Achievement[]>([])

  useEffect(() => {
    fetch(`${API_BASE}/achievements`, { credentials: 'include' })
      .then(res => (res.ok ? res.json() : []))
      .then(data => setAchievements(Array.isArray(data) ? data : []))
      .catch(() => setAchievements([]))
  }, [xp?.totalXp])

  const progressPercent = xp && xp.nextLevelXp > 0
    ? Math.min(100, Math.round((xp.totalXp / xp.nextLevelXp) * 100))
    : 0

  return (
    <div className="hud-header">
      <nav className="hud-nav">
        <Link to="/adventure">Adventure</Link>
        <Link to="/map">Map</Link>
      </nav>

      <div className="hud-xp">
        <span className="hud-level">Level {xp?.level ?? 1}</span>
        <div className="hud-progress-bar">
          <div className="hud-progress-fill" style={{ width: `${progressPercent}%` }} />
        </div>
        <span className="hud-xp-label">{xp?.totalXp ?? 0} / {xp?.nextLevelXp ?? 100} XP</span>
      </div>

      {achievements.length > 0 && (
        <div className="hud-achievements">
          {achievements.map(a => (
            <span key={a.id} className="hud-achievement-badge" title={a.name}>{a.name}</span>
          ))}
        </div>
      )}

      <button className="logout-btn" onClick={() => logout()}>Log Out</button>
    </div>
  )
}

export default HudHeader
