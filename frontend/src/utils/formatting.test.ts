import { describe, it, expect } from 'vitest'
import { capitalizeWords } from './formatting.ts'

describe('capitalizeWords', () => {
  it('capitalizes the first letter of each word', () => {
    expect(capitalizeWords('mount tamalpais')).toBe('Mount Tamalpais')
  })

  it('returns an empty string for undefined input', () => {
    expect(capitalizeWords(undefined)).toBe('')
  })
})
