import { ref, watch } from 'vue'
import { useDocumentVisibility, useIntervalFn } from '@vueuse/core'

export const REPORT_REFRESH_INTERVAL_MS = 5 * 60 * 1000
export const REPORT_MAX_POLL_DURATION_MS = 60 * 60 * 1000

/**
 * Auto-refresh with visibility pause and hourly poll cap (Issues/Workforce pattern).
 */
export function useReportPolling(refreshFn, lastUpdate) {
  const autoRefreshPaused = ref(false)
  let pollingSince = Date.now()

  const scheduledRefresh = () => {
    if (Date.now() - pollingSince >= REPORT_MAX_POLL_DURATION_MS) {
      pause()
      autoRefreshPaused.value = true
      return
    }
    refreshFn({ silent: true })
  }

  const visibility = useDocumentVisibility()
  const { pause, resume } = useIntervalFn(scheduledRefresh, REPORT_REFRESH_INTERVAL_MS)

  const markActive = () => {
    pollingSince = Date.now()
    autoRefreshPaused.value = false
    resume()
  }

  const manualRefresh = () => {
    markActive()
    refreshFn()
  }

  watch(visibility, (v) => {
    if (v === 'visible') {
      markActive()
      if (lastUpdate?.value && Date.now() - lastUpdate.value.getTime() >= REPORT_REFRESH_INTERVAL_MS) {
        refreshFn({ silent: true })
      }
    } else {
      pause()
    }
  })

  return {
    autoRefreshPaused,
    markActive,
    manualRefresh
  }
}