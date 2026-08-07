import { describe, it, expect, beforeEach } from 'vitest'
import { getUserId } from './userId.ts'

describe('getUserId', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('generates and persists a new id when none exists', () => {
    const id = getUserId()
    expect(id).toBeTruthy()
    expect(localStorage.getItem('userId')).toBe(id)
  })

  it('returns the same id on subsequent calls', () => {
    const first = getUserId()
    const second = getUserId()
    expect(second).toBe(first)
  })
})
