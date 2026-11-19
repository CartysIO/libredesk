import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { reportDateFnsLocale } from '@/features/reports/issuesFormat.js'

const DEFAULT_APP_TIMEZONE = 'UTC'

/** Viewer locale + app timezone for report formatting. */
export function useReportFormatContext() {
  const { locale: i18nLocale } = useI18n()
  const settingsStore = useAppSettingsStore()

  const dateLocale = computed(() => reportDateFnsLocale(i18nLocale.value))

  const appTimezone = computed(() => {
    const tz = settingsStore.settings?.['app.timezone']
    return typeof tz === 'string' && tz.trim() ? tz.trim() : DEFAULT_APP_TIMEZONE
  })

  onMounted(() => {
    if (!settingsStore.settings?.['app.timezone']) {
      settingsStore.fetchSettings('general')
    }
  })

  return { dateLocale, appTimezone, i18nLocale }
}

export function useReportLocale() {
  const { dateLocale } = useReportFormatContext()
  return dateLocale
}