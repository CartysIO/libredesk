import { describe, expect, it, vi, afterEach } from 'vitest'
import {
  createdAtBetweenFilter,
  resolveInboxAllDateBounds,
  inboxAllCreatedAtListFilter
} from './inboxAllDateRange'

afterEach(() => {
  vi.useRealTimers()
})

describe('resolveInboxAllDateBounds', () => {
  it('uses preset days when not custom', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-15T12:00:00Z'))
    expect(resolveInboxAllDateBounds({ presetDays: 7, isCustom: false, customStart: '', customEnd: '' })).toEqual({
      start: '2026-03-08',
      end: '2026-03-15'
    })
  })

  it('uses custom range when enabled', () => {
    expect(
      resolveInboxAllDateBounds({
        presetDays: 90,
        isCustom: true,
        customStart: '2026-01-01',
        customEnd: '2026-01-31'
      })
    ).toEqual({ start: '2026-01-01', end: '2026-01-31' })
  })
})

describe('createdAtBetweenFilter', () => {
  it('builds a conversations created_at between filter', () => {
    expect(createdAtBetweenFilter('2026-01-01', '2026-01-31')).toEqual({
      model: 'conversations',
      field: 'created_at',
      operator: 'between',
      value: '2026-01-01,2026-01-31'
    })
  })
})

describe('inboxAllCreatedAtListFilter', () => {
  it('defaults to 90 days', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-15T12:00:00Z'))
    expect(inboxAllCreatedAtListFilter({ presetDays: 90, isCustom: false, customStart: '', customEnd: '' })).toEqual({
      model: 'conversations',
      field: 'created_at',
      operator: 'between',
      value: '2025-12-15,2026-03-15'
    })
  })
})