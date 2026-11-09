import { format, parseISO, isValid } from 'date-fns'

export function formatReportDate(value) {
  if (value == null || value === '') return ''
  try {
    const d = typeof value === 'string' ? parseISO(value.slice(0, 10)) : new Date(value)
    if (!isValid(d)) return String(value)
    return format(d, 'MM/dd/yyyy')
  } catch {
    return String(value)
  }
}

export function formatReportDateTime(value) {
  try {
    const d = value instanceof Date ? value : new Date(value)
    if (!isValid(d)) return ''
    return format(d, 'MM/dd/yyyy, h:mm a')
  } catch {
    return ''
  }
}

export function formatReportHour24(hour) {
  const h = Number(hour)
  if (!Number.isFinite(h)) return ''
  const d = new Date()
  d.setHours(h, 0, 0, 0)
  return format(d, 'h:mm a')
}

export function formatCompactDuration(seconds) {
  const total = Math.floor(Number(seconds) || 0)
  if (total <= 0) return '0'
  const days = Math.floor(total / 86400)
  const months = Math.floor(days / 30)
  const daysAfterMonths = days % 30
  const weeks = Math.floor(daysAfterMonths / 7)
  const remDays = daysAfterMonths % 7
  const parts = []
  if (months) parts.push(`${months}m`)
  if (weeks) parts.push(`${weeks}w`)
  if (remDays) parts.push(`${remDays}d`)
  if (!parts.length) {
    const hours = Math.floor(total / 3600)
    const mins = Math.floor((total % 3600) / 60)
    if (hours) parts.push(`${hours}h`)
    if (mins) parts.push(`${mins}m`)
    if (!parts.length) parts.push(`${total}s`)
  }
  return parts.join(' ')
}

export function formatResolutionDaysHours(seconds) {
  const total = Math.floor(Number(seconds) || 0)
  if (total <= 0) return '—'
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  if (days > 0) return `${days}d ${hours}h`
  const mins = Math.floor((total % 3600) / 60)
  if (hours > 0) return mins ? `${hours}h ${mins}m` : `${hours}h`
  return mins ? `${mins}m` : `${total}s`
}

export function formatChartCount(value) {
  const n = Number(value)
  if (!Number.isFinite(n)) return ''
  return Number.isInteger(n) ? String(n) : n.toFixed(1)
}
