import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { demoAdventures } from '../data/demoAdventures.ts'
import { capitalizeWords } from '../utils/formatting.ts'
import type { Adventure } from '../types.ts'
import './HomePage.css'

const regions = [
  { value: 'bay-area', label: 'Bay Area' },
  { value: 'north-bay', label: 'North Bay' },
  { value: 'south-bay', label: 'South Bay' },
  { value: 'east-bay', label: 'East Bay' },
  // expand this as needed
]

function pickDemoAdventure(region: string, exclude?: Adventure): Adventure {
  const candidates = demoAdventures.filter(a => a.region === region)
  const pool = candidates.length > 1 && exclude
    ? candidates.filter(a => a.id !== exclude.id)
    : candidates
  return pool[Math.floor(Math.random() * pool.length)]
}

function HomePage() {
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
      <h1>Welcome to Solo Adventure Picker</h1>
      <p>Try a preview below, then sign up to explore the real map and save your progress.</p>

      <select
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
        <p>+{demoCard.xpValue}XP</p>
      </div>

      <button onClick={reroll}>Reroll</button>

      <div className="auth-links">
        <Link to="/signup">Sign Up</Link>
        <Link to="/login">Log In</Link>
      </div>
    </div>
  )
}

export default HomePage
