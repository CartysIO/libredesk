import { subDays, format } from 'date-fns'
import { DEFAULT_DATE_RANGE_DAYS } from '@/constants/dateRangePresets'

export { DEFAULT_DATE_RANGE_DAYS as INBOX_ALL_DEFAULT_RANGE_DAYS }

export const INBOX_ALL_DATE_FIELD_CREATED = 'created_at'
export const INBOX_ALL_DATE_FIELD_RESOLVED = 'resolved_at'

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

export function conversationDateBetweenFilter (field, start, end) {
  return {
    model: 'conversations',
    field,
    operator: 'between',
    value: `${start},${end}`
  }
}

export function createdAtBetweenFilter (start, end) {
  return conversationDateBetweenFilter(INBOX_ALL_DATE_FIELD_CREATED, start, end)
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

export function inboxAllDateListFilter (rangeState, field = INBOX_ALL_DATE_FIELD_CREATED) {
  const { start, end } = resolveInboxAllDateBounds(readInboxAllDateRangeState(rangeState))
  return conversationDateBetweenFilter(field, start, end)
}

export function inboxAllCreatedAtListFilter (rangeState) {
  return inboxAllDateListFilter(rangeState, INBOX_ALL_DATE_FIELD_CREATED)
}