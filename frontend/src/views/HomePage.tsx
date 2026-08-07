import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import './HomePage.css'

const regions = [
  { value: 'bay-area', label: 'Bay Area' },
  { value: 'north-bay', label: 'North Bay' },
  { value: 'south-bay', label: 'South Bay' },
  { value: 'east-bay', label: 'East Bay' },
  // expand this as needed
]

function HomePage() {
  const navigate = useNavigate()
  const [selectedRegion, setSelectedRegion] = useState('')

  const startAdventure = () => {
    navigate(`/adventure?region=${selectedRegion}`)
  }

  return (
    <div className="intro-page">
      <h1>Welcome to Solo Adventure Picker</h1>
      <p>Select your region to get started:</p>

      <select
        value={selectedRegion}
        onChange={(e) => setSelectedRegion(e.target.value)}
      >
        <option disabled value="">-- Select Region --</option>
        {regions.map(region => (
          <option key={region.value} value={region.value}>
            {region.label}
          </option>
        ))}
      </select>

      <button
        disabled={!selectedRegion}
        onClick={startAdventure}
      >
        Start Adventure
      </button>
    </div>
  )
}

export default HomePage
