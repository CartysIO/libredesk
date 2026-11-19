import { describe, test, expect } from 'vitest'
import { getRelativeTime } from './datetime.js'

describe('getRelativeTime', () => {
  test('uses calendar days once past 24 hours', () => {
    const now = new Date('2026-11-19T19:29:00.000Z')
    const then = new Date('2026-11-13T20:34:00.000Z')
    expect(getRelativeTime(then, now)).toBe('6d')
  })

  test('still shows hours under 24h elapsed', () => {
    const now = new Date('2026-11-19T19:00:00.000Z')
    const then = new Date('2026-11-19T08:00:00.000Z')
    expect(getRelativeTime(then, now)).toBe('11h')
  })
})