import { Routes, Route } from 'react-router-dom'
import HomePage from './views/HomePage'
import AdventurePage from './views/AdventurePage'
import MapPage from './views/MapPage'
import './App.css'

function App() {
  return (
    <div id="app">
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/adventure" element={<AdventurePage />} />
        <Route path="/map" element={<MapPage />} />
      </Routes>
    </div>
  )
}

export default App
