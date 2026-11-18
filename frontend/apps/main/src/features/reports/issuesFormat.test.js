import { describe, it, expect } from 'vitest'
import {
  formatReportDate,
  formatCompactDuration,
  formatResolutionDaysHours,
  formatReportHour24,
  formatReportDateTimeReadable,
  formatReportDurationSeconds
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

  it('formats readable report datetimes', () => {
    expect(formatReportDateTimeReadable('2026-11-17T14:50:00.000Z')).toMatch(/Nov 17, 2026/)
    expect(formatReportDateTimeReadable('Nov 17, 2026, 8:50 AM')).toBe('Nov 17, 2026, 8:50 AM')
  })

  it('formats optional report durations', () => {
    expect(formatReportDurationSeconds(null)).toBe('—')
    expect(formatReportDurationSeconds(0)).toBe('0s')
    expect(formatReportDurationSeconds(90)).toBe('1m')
  })
})
