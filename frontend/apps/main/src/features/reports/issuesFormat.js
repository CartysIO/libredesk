import { addDays, format, parseISO, isValid, setISODay } from 'date-fns'
import { enUS, es, fr, ptBR, ja } from 'date-fns/locale'

const REPORT_DATE_LOCALES = {
  'en-US': enUS,
  en: enUS,
  'es-ES': es,
  es,
  'fr-FR': fr,
  fr,
  'pt-BR': ptBR,
  pt: ptBR,
  'ja-JP': ja,
  ja
}

export function reportDateFnsLocale(i18nLocale) {
  if (!i18nLocale) return enUS
  return REPORT_DATE_LOCALES[i18nLocale] ?? enUS
}

function parseReportDay(value) {
  if (value == null || value === '') return null
  try {
    const d = typeof value === 'string' ? parseISO(value.slice(0, 10)) : new Date(value)
    return isValid(d) ? d : null
  } catch {
    return null
  }
}

/** Short localized calendar date for chart axes (no time). */
export function formatReportDate(value, locale = enUS) {
  const d = parseReportDay(value)
  if (!d) return value == null ? '' : String(value)
  return format(d, 'P', { locale })
}

/** Tooltip title for weekly buckets (week start is Monday in app TZ from API). */
export function formatReportWeekTooltipTitle(weekStart, locale, translate) {
  const start = parseReportDay(weekStart)
  if (!start) return String(weekStart ?? '')
  const end = addDays(start, 6)
  const startLabel = format(start, 'P', { locale })
  const endLabel = format(end, 'P', { locale })
  const weekOf = translate('report.chart.weekOf', { date: startLabel })
  const range = translate('report.chart.weekRange', { start: startLabel, end: endLabel })
  return `${weekOf} (${range})`
}

export function formatReportDateTime(value, locale = enUS) {
  try {
    const d = value instanceof Date ? value : new Date(value)
    if (!isValid(d)) return ''
    return format(d, 'Pp', { locale })
  } catch {
    return ''
  }
}

/** Readable report datetime (e.g. Nov 17, 2026, 8:50 AM). Prefer `created_at_display` from API when set. */
export function formatReportDateTimeReadable(value, locale = enUS) {
  if (value == null || value === '') return ''
  if (typeof value === 'string' && !value.includes('T') && !/^\d{4}-\d{2}-\d{2}/.test(value)) {
    return value
  }
  try {
    const d = value instanceof Date ? value : new Date(value)
    if (!isValid(d)) return ''
    return format(d, 'MMM d, yyyy, h:mm a', { locale })
  } catch {
    return ''
  }
}

/** Duration for report tables: null/absent → em dash; zero → 0s. */
export function formatReportDurationSeconds(seconds) {
  if (seconds == null || seconds === '') return '—'
  const total = Math.floor(Number(seconds))
  if (!Number.isFinite(total) || total < 0) return '—'
  if (total === 0) return '0s'
  return formatResolutionDaysHours(total)
}

/** Hour-of-day bucket label; always 12-hour clock with AM/PM. */
export function formatReportHour24(hour, locale = enUS) {
  const h = Number(hour)
  if (!Number.isFinite(h)) return ''
  const d = new Date()
  d.setHours(h, 0, 0, 0)
  return format(d, 'h:mm a', { locale })
}

/** ISO day-of-week 1 (Mon) … 7 (Sun). */
export function formatReportDayOfWeek(isoDow, locale = enUS) {
  const n = Number(isoDow)
  if (!Number.isFinite(n) || n < 1 || n > 7) return ''
  const ref = setISODay(new Date(2024, 0, 1), n)
  return format(ref, 'EEE', { locale })
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