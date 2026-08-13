import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { demoAdventures } from '../data/demoAdventures.ts'
import { regions } from '../data/regions.ts'
import { capitalizeWords } from '../utils/formatting.ts'
import { useAuth } from '../context/AuthContext.tsx'
import HeroBanner from '../components/HeroBanner.tsx'
import type { Adventure } from '../types.ts'
import './HomePage.css'

function pickDemoAdventure(region: string, exclude?: Adventure): Adventure {
  const candidates = demoAdventures.filter(a => a.region === region)
  const pool = candidates.length > 1 && exclude
    ? candidates.filter(a => a.id !== exclude.id)
    : candidates
  return pool[Math.floor(Math.random() * pool.length)]
}

function HomePage() {
  const { user } = useAuth()
  const [selectedRegion, setSelectedRegion] = useState('bay-area')
  const [demoCard, setDemoCard] = useState<Adventure>(() => pickDemoAdventure('bay-area'))

  const displayName = useMemo(() => capitalizeWords(demoCard.name), [demoCard.name])

  const handleRegionChange = (region: string) => {
    setSelectedRegion(region)
    setDemoCard(pickDemoAdventure(region))
  }

  const reroll = () => {
    setDemoCard(pickDemoAdventure(selectedRegion, demoCard))
  }

  return (
    <div className="intro-page">
      <HeroBanner>
        <h1>Welcome to Solo Adventure Picker</h1>
        <p>Try a preview below, then sign up to explore the real map and save your progress.</p>
      </HeroBanner>

      <div className="demo-section">
        <label htmlFor="region-select">Region</label>
        <select
          id="region-select"
          value={selectedRegion}
          onChange={(e) => handleRegionChange(e.target.value)}
        >
          {regions.map(region => (
            <option key={region.value} value={region.value}>
              {region.label}
            </option>
          ))}
        </select>

        <div className="demo-card">
          <h2>{displayName}</h2>
          <p className="demo-card-xp">+{demoCard.xpValue}XP</p>
        </div>

        <button className="btn btn-secondary" onClick={reroll}>Reroll</button>

        <div className="auth-links">
          {user ? (
            <Link className="btn btn-primary" to="/adventure">Continue your adventure</Link>
          ) : (
            <>
              <Link className="btn btn-primary" to="/signup">Sign Up</Link>
              <Link className="btn btn-ghost" to="/login">Log In</Link>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

export default HomePage
