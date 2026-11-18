import { subDays, format } from 'date-fns'

export const INBOX_ALL_RANGE_OPTIONS = [7, 30, 90, 120]
export const INBOX_ALL_DEFAULT_RANGE_DAYS = 90

export function resolveInboxAllDateBounds ({ presetDays, isCustom, customStart, customEnd }) {
  if (isCustom && customStart && customEnd) {
    return { start: customStart, end: customEnd }
  }
  const days = presetDays || INBOX_ALL_DEFAULT_RANGE_DAYS
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

export function inboxAllCreatedAtListFilter (rangeState) {
  const { start, end } = resolveInboxAllDateBounds(rangeState)
  return createdAtBetweenFilter(start, end)
}