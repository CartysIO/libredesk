import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { reportDateFnsLocale } from '@/features/reports/issuesFormat.js'

export function useReportLocale() {
  const { locale } = useI18n()
  return computed(() => reportDateFnsLocale(locale.value))
}