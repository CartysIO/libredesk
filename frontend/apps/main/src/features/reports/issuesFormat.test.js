import { describe, it, expect } from 'vitest'
import {
  formatReportDate,
  formatCompactDuration,
  formatResolutionDaysHours,
  formatReportHour24
} from './issuesFormat.js'

describe('issuesFormat', () => {
  it('formats US dates', () => {
    expect(formatReportDate('2026-08-05')).toBe('08/05/2026')
  })

  it('formats compact duration', () => {
    expect(formatCompactDuration(86400 * 11 + 86400 * 4)).toContain('w')
  })

  it('formats resolution as days and hours', () => {
    expect(formatResolutionDaysHours(90000)).toBe('1d 1h')
  })

  it('formats hours in twelve hour clock', () => {
    expect(formatReportHour24(0)).toMatch(/12:00 AM/i)
    expect(formatReportHour24(15)).toMatch(/3:00 PM/i)
  })
})
