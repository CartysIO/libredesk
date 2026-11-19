<template>
  <div class="min-h-0 flex-1 overflow-y-auto">
    <div
      class="desk-reports w-full px-3 py-4"
      :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
    >
      <Spinner v-if="isLoading" />

      <div class="desk-report-sections flex flex-col gap-6">
        <div class="mb-1 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground">
              {{ $t('globals.terms.lastUpdated') }}: {{ lastUpdateFormatted }}
            </span>
            <span class="text-xs text-muted-foreground">{{ $t('report.workforce.dataDelay') }}</span>
            <span class="text-xs text-muted-foreground">{{ $t('report.workforce.calendarTimeNote') }}</span>
            <template v-if="autoRefreshPaused">
              <span class="text-sm text-muted-foreground">{{ $t('report.autoRefreshPaused') }}</span>
              <Button size="sm" variant="outline" :disabled="isLoading" @click="manualRefresh">
                {{ $t('globals.terms.refresh') }}
              </Button>
            </template>
          </div>
          <div class="desk-report-range flex flex-wrap items-center gap-1">
            <Button
              v-for="option in rangeOptions"
              :key="option"
              size="sm"
              :variant="!isCustom && range === option ? 'default' : 'outline'"
              :disabled="isLoading"
              @click="selectRange(option)"
            >
              {{ $t('globals.messages.nDays', { days: option }) }}
            </Button>
            <Separator orientation="vertical" class="h-6 mx-1" />
            <Button
              size="sm"
              :variant="isCustom ? 'default' : 'outline'"
              :disabled="isLoading"
              @click="enableCustom"
            >
              {{ $t('globals.terms.custom') }}
            </Button>
            <IssuesReportDateRange
              v-if="isCustom"
              v-model:start="customStart"
              v-model:end="customEnd"
              :disabled="isLoading"
            />
          </div>
        </div>

        <div class="desk-report-scalar-grid grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3">
          <div
            v-for="scalar in scalarCards"
            :key="scalar.key"
            class="desk-report-panel p-5"
          >
            <p class="mb-1 text-xs text-muted-foreground">{{ scalar.sublabel }}</p>
            <p class="ui-heading mb-2">{{ scalar.title }}</p>
            <span class="desk-report-metric">{{ scalar.display }}</span>
          </div>
        </div>

        <div :class="REPORT_ROW_CLASS">
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.workforce.firstResponseBuckets') }}</p>
            <IssuesBarChart
              :data="firstResponseBucketChart"
              :categories="[countLabel]"
              index="label"
            />
          </div>
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.workforce.resolutionBuckets') }}</p>
            <IssuesBarChart
              :data="resolutionBucketChart"
              :categories="[countLabel]"
              index="label"
            />
          </div>
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.workforce.medianResponseResolution') }}</p>
          <p v-if="!medianSlaHasData" class="text-sm text-muted-foreground min-h-[200px]">
            {{ $t('report.workforce.noSlaData') }}
          </p>
          <IssuesLineChart
            v-else
            :data="medianSlaChart"
            :categories="[firstResponseLabel, resolutionLabel]"
            index-field="date"
            format-index-as-week
            duration-axis
          />
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.workforce.achievedVsBreached') }}</p>
          <IssuesBarChart
            :data="achievedVsBreachedChart"
            :categories="[achievedLabel, breachedLabel]"
            index="week"
            format-index-as-week
          />
        </div>

        <div :class="REPORT_ROW_CLASS">
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.workforce.breachRateByHour') }}</p>
            <IssuesBarChart
              :data="breachRateByHourChart"
              :categories="[breachRateLabel]"
              index="label"
              :y-formatter="percentYFormatter"
              :value-formatter="percentValueFormatter"
            />
          </div>
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.workforce.breachRateByDow') }}</p>
            <IssuesBarChart
              :data="breachRateByDowChart"
              :categories="[breachRateLabel]"
              index="label"
              :y-formatter="percentYFormatter"
              :value-formatter="percentValueFormatter"
            />
          </div>
        </div>

        <div :class="REPORT_ROW_CLASS">
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.workforce.firstResponseByAssignee') }}</p>
            <IssuesBarChart
              :data="firstResponseByAssigneeChart"
              :categories="[medianFrLabel]"
              index="assignee_name"
              duration-y-axis
            />
          </div>
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.workforce.resolutionByAssignee') }}</p>
            <IssuesBarChart
              :data="resolutionByAssigneeChart"
              :categories="[medianResLabel]"
              index="assignee_name"
              duration-y-axis
            />
          </div>
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.workforce.assigneeOverview') }}</p>
          <p v-if="!assigneeRows.length" class="text-sm text-muted-foreground">
            {{ $t('report.workforce.noAssignees') }}
          </p>
          <IssuesAccountsTable
            v-else
            :rows="assigneeRows"
            :columns="assigneeColumns"
            :sort-by="assigneeSortBy"
            :sort-dir="assigneeSortDir"
            @update:sort="onAssigneeSort"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { format, subDays } from 'date-fns'
import { watchDebounced } from '@vueuse/core'
import { useDocumentVisibility, useIntervalFn } from '@vueuse/core'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import {
  formatReportDateTime,
  formatReportDayOfWeek,
  formatReportHour24,
  formatResolutionDaysHours,
  reportDateFnsLocale
} from '@/features/reports/issuesFormat.js'
import IssuesBarChart from '@/features/reports/IssuesBarChart.vue'
import IssuesLineChart from '@/features/reports/IssuesLineChart.vue'
import IssuesReportDateRange from '@/features/reports/IssuesReportDateRange.vue'
import IssuesAccountsTable from '@/features/reports/IssuesAccountsTable.vue'
import Spinner from '@shared-ui/components/ui/spinner/Spinner.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Separator } from '@shared-ui/components/ui/separator'
import { useI18n } from 'vue-i18n'
import api from '@main/api'

const REPORT_ROW_CLASS = 'flex w-full flex-col gap-4 md:flex-row md:flex-wrap'
const REFRESH_INTERVAL_MS = 5 * 60 * 1000
const MAX_POLL_DURATION_MS = 60 * 60 * 1000

const emitter = useEmitter()
const { t, locale } = useI18n()
const reportLocale = computed(() => reportDateFnsLocale(locale.value))
const isLoading = ref(false)
const lastUpdate = ref(new Date())
const rangeOptions = [7, 30, 90, 120]
const range = ref(120)
const isCustom = ref(false)
const customStart = ref('')
const customEnd = ref('')
const autoRefreshPaused = ref(false)
const assigneeSortBy = ref('total')
const assigneeSortDir = ref('desc')

const scalars = ref({})
const charts = ref({})

const countLabel = computed(() => t('report.workforce.tickets'))
const firstResponseLabel = computed(() => t('report.workforce.firstResponse'))
const resolutionLabel = computed(() => t('report.workforce.resolution'))
const achievedLabel = computed(() => t('report.workforce.achieved'))
const breachedLabel = computed(() => t('report.workforce.breached'))
const breachRateLabel = computed(() => t('report.workforce.breachRate'))
const medianFrLabel = computed(() => t('report.workforce.medianFirstResponse'))
const medianResLabel = computed(() => t('report.workforce.medianResolution'))

const formatPercent = (value) => {
  const n = Number(value)
  if (!Number.isFinite(n)) return '—'
  return `${n.toFixed(1)}%`
}

const percentYFormatter = (tick) => (typeof tick === 'number' ? `${tick}%` : '')
const percentValueFormatter = (value) => formatPercent(value)

const scalarCards = computed(() => [
  {
    key: 'median_first_response',
    title: t('report.workforce.medianFirstResponse'),
    sublabel: t('report.workforce.inRange'),
    display: formatResolutionDaysHours(scalars.value.median_first_response_sec)
  },
  {
    key: 'median_resolution',
    title: t('report.workforce.medianResolution'),
    sublabel: t('report.workforce.inRange'),
    display: formatResolutionDaysHours(scalars.value.median_resolution_sec)
  },
  {
    key: 'sla_achievement',
    title: t('report.workforce.slaAchievementRate'),
    sublabel: t('report.workforce.inRange'),
    display: formatPercent(scalars.value.sla_achievement_rate)
  },
  {
    key: 'sla_breached',
    title: t('report.workforce.slaBreachedTickets'),
    sublabel: t('report.workforce.inRange'),
    display: formatPercent(scalars.value.sla_breached_tickets_percent)
  },
  {
    key: 'active_breached',
    title: t('report.workforce.activeBreached'),
    sublabel: t('report.workforce.now'),
    display: String(scalars.value.active_breached_count ?? 0)
  },
  {
    key: 'fr_breach',
    title: t('report.workforce.firstResponseBreachRate'),
    sublabel: t('report.workforce.inRange'),
    display: formatPercent(scalars.value.first_response_breach_percent)
  },
  {
    key: 'res_breach',
    title: t('report.workforce.resolutionBreachRate'),
    sublabel: t('report.workforce.inRange'),
    display: formatPercent(scalars.value.resolution_breach_percent)
  },
  {
    key: 'nr_breach',
    title: t('report.workforce.nextResponseBreachRate'),
    sublabel: t('report.workforce.inRange'),
    display: formatPercent(scalars.value.next_response_breach_percent)
  }
])

const firstResponseBucketChart = computed(() =>
  (charts.value.first_response_buckets || []).map((row) => ({
    label: row.label,
    [countLabel.value]: row.count
  }))
)

const resolutionBucketChart = computed(() =>
  (charts.value.resolution_buckets || []).map((row) => ({
    label: row.label,
    [countLabel.value]: row.count
  }))
)

const medianSlaChart = computed(() =>
  (charts.value.median_sla_by_week || []).map((row) => ({
    date: row.week,
    [firstResponseLabel.value]: Number(row.median_first_response_sec) || 0,
    [resolutionLabel.value]: Number(row.median_resolution_sec) || 0
  }))
)

const medianSlaHasData = computed(() => {
  const rows = charts.value.median_sla_by_week || []
  return rows.some(
    (row) =>
      (Number(row.median_first_response_sec) || 0) > 0 ||
      (Number(row.median_resolution_sec) || 0) > 0
  )
})

const achievedVsBreachedChart = computed(() =>
  (charts.value.achieved_vs_breached_by_week || []).map((row) => ({
    week: row.week,
    [achievedLabel.value]: row.achieved,
    [breachedLabel.value]: row.breached
  }))
)

const breachRateByHourChart = computed(() =>
  (charts.value.breach_rate_by_hour || []).map((row) => ({
    label: formatReportHour24(row.hour, reportLocale.value),
    [breachRateLabel.value]: Number(row.breach_rate) || 0
  }))
)

const breachRateByDowChart = computed(() =>
  (charts.value.breach_rate_by_dow || []).map((row) => ({
    label:
      row.dow != null
        ? formatReportDayOfWeek(row.dow, reportLocale.value)
        : row.label,
    [breachRateLabel.value]: Number(row.breach_rate) || 0
  }))
)

const firstResponseByAssigneeChart = computed(() =>
  (charts.value.first_response_by_assignee || []).map((row) => ({
    assignee_name: row.assignee_name,
    [medianFrLabel.value]: Number(row.median_first_response_sec) || 0
  }))
)

const resolutionByAssigneeChart = computed(() =>
  (charts.value.resolution_by_assignee || []).map((row) => ({
    assignee_name: row.assignee_name,
    [medianResLabel.value]: Number(row.median_resolution_sec) || 0
  }))
)

const assigneeColumns = computed(() => [
  { key: 'assignee_name', label: t('report.workforce.assignee') },
  { key: 'open_count', label: t('globals.terms.open') },
  { key: 'closed_count', label: t('report.workforce.closed') },
  { key: 'total', label: t('report.workforce.total') },
  {
    key: 'median_first_response_sec',
    label: t('report.workforce.medianFirstResponse'),
    format: (v) => formatResolutionDaysHours(v)
  },
  {
    key: 'median_resolution_sec',
    label: t('report.workforce.medianResolution'),
    format: (v) => formatResolutionDaysHours(v)
  },
  {
    key: 'breach_rate_percent',
    label: t('report.workforce.breachRate'),
    format: (v) => formatPercent(v)
  }
])

const assigneeRows = computed(() => {
  const rows = [...(charts.value.assignee_overview || [])]
  const { by, dir } = { by: assigneeSortBy.value, dir: assigneeSortDir.value }
  rows.sort((a, b) => {
    const av = a[by]
    const bv = b[by]
    if (by === 'assignee_name') {
      return (dir === 'asc' ? 1 : -1) * String(av).localeCompare(String(bv))
    }
    const an = Number(av) || 0
    const bn = Number(bv) || 0
    return dir === 'asc' ? an - bn : bn - an
  })
  return rows
})

const lastUpdateFormatted = computed(() =>
  formatReportDateTime(lastUpdate.value, reportLocale.value)
)

const showError = (error) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })
}

let loadSeq = 0

const appendRangeParams = (params, { start, end, days }) => {
  if (start && end) {
    params.start_date = start
    params.end_date = end
  } else if (days != null) {
    params.days = days
  }
  return params
}

const buildReportParams = () =>
  appendRangeParams(
    {},
    {
      start: isCustom.value ? customStart.value : '',
      end: isCustom.value ? customEnd.value : '',
      days: isCustom.value ? null : range.value
    }
  )

const loadReport = async ({ silent = false } = {}) => {
  const seq = ++loadSeq
  if (!silent) {
    isLoading.value = true
  }
  try {
    const { data } = await api.getWorkforceReport(buildReportParams())
    if (seq !== loadSeq) return
    scalars.value = data.data.scalars || {}
    charts.value = data.data.charts || {}
    lastUpdate.value = new Date()
  } catch (error) {
    if (seq !== loadSeq) return
    showError(error)
  } finally {
    if (seq === loadSeq && !silent) {
      isLoading.value = false
    }
  }
}

const selectRange = (option) => {
  isCustom.value = false
  if (option === range.value) return
  markActive()
  range.value = option
  loadReport()
}

const seedCustomRangeFromPreset = () => {
  const end = new Date()
  const start = subDays(end, range.value)
  customEnd.value = format(end, 'yyyy-MM-dd')
  customStart.value = format(start, 'yyyy-MM-dd')
}

const enableCustom = () => {
  isCustom.value = true
  if (!customStart.value || !customEnd.value) {
    seedCustomRangeFromPreset()
  }
}

const applyCustomRange = () => {
  if (!isCustom.value || !customStart.value || !customEnd.value) return
  if (customStart.value > customEnd.value) return
  markActive()
  loadReport()
}

const onAssigneeSort = ({ by, dir }) => {
  assigneeSortBy.value = by
  assigneeSortDir.value = dir
}

let pollingSince = Date.now()

const scheduledRefresh = () => {
  if (Date.now() - pollingSince >= MAX_POLL_DURATION_MS) {
    pause()
    autoRefreshPaused.value = true
    return
  }
  loadReport({ silent: true })
}

const visibility = useDocumentVisibility()
const { pause, resume } = useIntervalFn(scheduledRefresh, REFRESH_INTERVAL_MS)

const markActive = () => {
  pollingSince = Date.now()
  autoRefreshPaused.value = false
  resume()
}

const manualRefresh = () => {
  markActive()
  loadReport()
}

watch(visibility, (v) => {
  if (v === 'visible') {
    markActive()
    if (Date.now() - lastUpdate.value.getTime() >= REFRESH_INTERVAL_MS) {
      loadReport({ silent: true })
    }
  } else {
    pause()
  }
})

watchDebounced(
  [customStart, customEnd],
  () => {
    if (isCustom.value && customStart.value && customEnd.value) {
      applyCustomRange()
    }
  },
  { debounce: 350 }
)

onMounted(() => {
  loadReport()
})
</script>