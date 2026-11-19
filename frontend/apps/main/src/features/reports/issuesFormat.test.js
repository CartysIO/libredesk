import { describe, it, expect } from 'vitest'
import { es, enUS } from 'date-fns/locale'
import {
  formatReportDate,
  formatReportWeekTooltipTitle,
  formatCompactDuration,
  formatResolutionDaysHours,
  formatReportHour24,
  formatReportDateTimeReadable,
  formatReportDurationSeconds,
  formatReportDayOfWeek,
  parseReportDay
} from './issuesFormat.js'

describe('issuesFormat', () => {
  it('formats localized chart dates without UTC day shift', () => {
    expect(formatReportDate('2026-08-05', enUS)).toBe('08/05/2026')
    expect(formatReportDate('2026-08-05', es)).toMatch(/2026/)
    const d = parseReportDay('2026-08-05')
    expect(d.getFullYear()).toBe(2026)
    expect(d.getMonth()).toBe(7)
    expect(d.getDate()).toBe(5)
  })

  it('formats week tooltip with range', () => {
    const title = formatReportWeekTooltipTitle('2026-08-05', enUS, (key, params) => {
      if (key === 'report.chart.weekOf') return `Week of ${params.date}`
      return `${params.start} – ${params.end}`
    })
    expect(title).toContain('Week of 08/05/2026')
    expect(title).toContain('08/11/2026')
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

  it('formats localized day of week', () => {
    expect(formatReportDayOfWeek(1, enUS)).toMatch(/Mon/i)
    expect(formatReportDayOfWeek(1, es)).toMatch(/lun/i)
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