import { ref } from 'vue'
import { subDays, format } from 'date-fns'
import { watchDebounced } from '@vueuse/core'

export const REPORT_RANGE_OPTIONS = [7, 30, 90, 120]

/**
 * Shared preset/custom date range state for report views.
 */
export function useReportDateRange({ defaultDays = 90, onRangeApply }) {
  const range = ref(defaultDays)
  const isCustom = ref(false)
  const customStart = ref('')
  const customEnd = ref('')

  const seedCustomRangeFromPreset = () => {
    const end = new Date()
    const start = subDays(end, range.value)
    customEnd.value = format(end, 'yyyy-MM-dd')
    customStart.value = format(start, 'yyyy-MM-dd')
  }

  const appendRangeParams = (params) => {
    if (isCustom.value && customStart.value && customEnd.value) {
      params.start_date = customStart.value
      params.end_date = customEnd.value
    } else {
      params.days = range.value
    }
    return params
  }

  const selectRange = (option) => {
    isCustom.value = false
    if (option === range.value) return false
    range.value = option
    onRangeApply?.()
    return true
  }

  const enableCustom = () => {
    isCustom.value = true
    if (!customStart.value || !customEnd.value) {
      seedCustomRangeFromPreset()
    }
  }

  const applyCustomRange = () => {
    if (!isCustom.value || !customStart.value || !customEnd.value) return false
    if (customStart.value > customEnd.value) return false
    onRangeApply?.()
    return true
  }

  watchDebounced(
    [customStart, customEnd],
    () => {
      if (isCustom.value && customStart.value && customEnd.value) {
        applyCustomRange()
      }
    },
    { debounce: 350 }
  )

  return {
    range,
    isCustom,
    customStart,
    customEnd,
    appendRangeParams,
    selectRange,
    enableCustom,
    applyCustomRange
  }
}