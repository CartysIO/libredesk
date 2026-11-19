import { describe, test, expect } from 'vitest'
import { getRelativeTime } from './datetime'

describe('getRelativeTime', () => {
    // Pylon parity: Nov 13 2:34 PM America/Chicago ≈ 2026-11-13T20:34:00Z
    test('counts calendar days for stale tickets (board vs Pylon)', () => {
        const now = new Date('2026-11-19T19:29:00.000Z')
        const then = new Date('2026-11-13T20:34:00.000Z')
        expect(getRelativeTime(then, now)).toBe('6d')
    })

    test('shows hours for same-calendar-day elapsed time', () => {
        const now = new Date('2026-11-19T19:00:00.000Z')
        const then = new Date('2026-11-19T08:00:00.000Z')
        expect(getRelativeTime(then, now)).toBe('11h')
    })

    test('prefers 1d over hours when a calendar day boundary was crossed', () => {
        const now = new Date('2026-11-19T07:00:00.000Z')
        const then = new Date('2026-11-18T06:00:00.000Z')
        expect(getRelativeTime(then, now)).toBe('1d')
    })
})