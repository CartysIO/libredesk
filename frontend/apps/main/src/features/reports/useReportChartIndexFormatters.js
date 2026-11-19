import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  reportDateFnsLocale,
  formatReportDate,
  formatReportWeekTooltipTitle
} from '@/features/reports/issuesFormat.js'

/** Shared X-axis / crosshair title formatters for report charts. */
export function useReportChartIndexFormatters(props) {
  const { t, locale } = useI18n()
  const dateLocale = computed(() => reportDateFnsLocale(locale.value))

  const formatIndexAxisValue = (raw) => {
    if (props.formatIndexAsWeek || props.formatIndexAsDate) {
      return formatReportDate(raw, dateLocale.value)
    }
    return String(raw ?? '')
  }

  const formatIndexTitleValue = (raw) => {
    if (props.formatIndexAsWeek) {
      return formatReportWeekTooltipTitle(raw, dateLocale.value, t)
    }
    if (props.formatIndexAsDate) {
      return formatReportDate(raw, dateLocale.value)
    }
    return String(raw ?? '')
  }

  return { formatIndexAxisValue, formatIndexTitleValue }
}