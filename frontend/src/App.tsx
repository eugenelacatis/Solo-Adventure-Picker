import { Routes, Route } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import RequireAuth from './context/RequireAuth'
import HomePage from './views/HomePage'
import AdventurePage from './views/AdventurePage'
import MapPage from './views/MapPage'
import HistoryPage from './views/HistoryPage'
import LoginPage from './views/LoginPage'
import SignupPage from './views/SignupPage'
import './App.css'

function App() {
  return (
    <AuthProvider>
      <div id="app">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/signup" element={<SignupPage />} />
          <Route path="/adventure" element={<RequireAuth><AdventurePage /></RequireAuth>} />
          <Route path="/map" element={<RequireAuth><MapPage /></RequireAuth>} />
          <Route path="/history" element={<RequireAuth><HistoryPage /></RequireAuth>} />
        </Routes>
      </div>
    </AuthProvider>
  )
}

export default App
