import type { ReactNode } from 'react'
import './HeroBanner.css'

interface HeroBannerProps { children: ReactNode }

// Swap point: drop a real photo at public/background.jpg (or change
// HERO_IMAGE) — no other code changes needed.
const HERO_IMAGE = '/background.jpg'

function HeroBanner({ children }: HeroBannerProps) {
  return (
    <div className="hero-banner">
      <div
        className="hero-banner-bg"
        style={{ backgroundImage: `linear-gradient(180deg, rgba(22,33,28,0.15) 0%, rgba(22,33,28,0.55) 100%), url(${HERO_IMAGE})` }}
      />
      <div className="hero-banner-content">{children}</div>
    </div>
  )
}

export default HeroBanner
