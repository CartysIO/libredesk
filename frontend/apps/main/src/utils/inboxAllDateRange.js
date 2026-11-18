import { subDays, format } from 'date-fns'
import { DEFAULT_DATE_RANGE_DAYS } from '@/constants/dateRangePresets'

export { DEFAULT_DATE_RANGE_DAYS as INBOX_ALL_DEFAULT_RANGE_DAYS }

export function resolveInboxAllDateBounds ({ presetDays, isCustom, customStart, customEnd }) {
  if (isCustom && customStart && customEnd) {
    return { start: customStart, end: customEnd }
  }
  const days = presetDays || DEFAULT_DATE_RANGE_DAYS
  const end = new Date()
  const start = subDays(end, days)
  return {
    start: format(start, 'yyyy-MM-dd'),
    end: format(end, 'yyyy-MM-dd')
  }
}

export function createdAtBetweenFilter (start, end) {
  return {
    model: 'conversations',
    field: 'created_at',
    operator: 'between',
    value: `${start},${end}`
  }
}

/** Reads range fields from the reactive store bucket (refs are unwrapped via property access). */
export function readInboxAllDateRangeState (rangeState) {
  return {
    presetDays: rangeState.presetDays,
    isCustom: rangeState.isCustom,
    customStart: rangeState.customStart,
    customEnd: rangeState.customEnd
  }
}

export function inboxAllCreatedAtListFilter (rangeState) {
  const { start, end } = resolveInboxAllDateBounds(readInboxAllDateRangeState(rangeState))
  return createdAtBetweenFilter(start, end)
}