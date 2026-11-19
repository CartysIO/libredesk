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
            <span class="text-xs text-muted-foreground">{{ $t('report.issues.dataDelay') }}</span>
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
            <span class="desk-report-metric">{{ formatCompactNumber(scalar.value) }}</span>
          </div>
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.issues.volumeByDay') }}</p>
          <IssuesBarChart
            :data="volumeByDayChart"
            :categories="[countLabel]"
            index="date"
            format-index-as-date
          />
        </div>

        <div :class="REPORT_ROW_CLASS">
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.issues.byHour') }}</p>
            <IssuesBarChart :data="byHourChart" :categories="[countLabel]" index="label" />
          </div>
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.issues.byDayOfWeek') }}</p>
            <IssuesBarChart :data="byDowChart" :categories="[countLabel]" index="label" />
          </div>
        </div>

        <div :class="REPORT_ROW_CLASS">
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.issues.bySource') }}</p>
            <IssuesDonutChart
              :data="bySourceChart"
              :value-key="countLabel"
              label-key="label"
              :value-label="countLabel"
              :center-label="countLabel"
              :empty-label="$t('report.issues.noSourceData')"
              :aria-label="$t('report.issues.bySource')"
            />
          </div>
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.issues.meanTouchesBySource') }}</p>
            <IssuesBarChart :data="touchesBySourceChart" :categories="[touchesLabel]" index="label" />
          </div>
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.issues.volumeOverTime') }}</p>
          <IssuesBarChart
            :data="volumeOverTimeChart"
            :categories="[createdLabel, closedLabel]"
            index="date"
            format-index-as-date
          />
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.issues.tagsByWeek') }}</p>
          <IssuesBarChart
            :data="tagsByWeekChart.data"
            :categories="tagsByWeekChart.categories"
            index="week"
            type="stacked"
            format-index-as-week
          />
        </div>

        <div :class="REPORT_ROW_CLASS">
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.issues.medianResponseResolution') }}</p>
            <p v-if="!medianSlaHasData" class="text-sm text-muted-foreground min-h-[200px]">
              {{ $t('report.issues.noSlaData') }}
            </p>
            <IssuesLineChart
              v-else
              :data="medianSlaChart"
              :categories="[firstResponseLabel, resolutionLabel]"
              format-index-as-week
              duration-axis
            />
          </div>
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.issues.solvedWithinWeek') }}</p>
            <IssuesBarChart
              :data="solvedWithinWeekChart"
              :categories="[withinWeekLabel, overWeekLabel]"
              index="week"
              type="stacked"
              format-index-as-week
            />
          </div>
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.issues.medianTimeInStatus') }}</p>
          <IssuesLineChart
            :data="medianTimeInStatusChart.data"
            :categories="medianTimeInStatusChart.categories"
            format-index-as-week
            duration-axis
          />
        </div>

        <div
          class="box w-full p-5 transition-opacity duration-300"
          :class="{ 'opacity-50': accountsLoading }"
        >
          <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mb-4">
            <p class="ui-heading">{{ $t('report.issues.topAccounts') }}</p>
            <div class="flex flex-wrap items-center gap-2">
              <IssuesReportDateRange
                v-model:start="accountsStart"
                v-model:end="accountsEnd"
                :disabled="accountsLoading"
              />
              <Input
                v-model="accountSearch"
                class="max-w-xs"
                :placeholder="$t('report.issues.searchAccounts')"
                :disabled="accountsLoading"
              />
            </div>
          </div>
          <Spinner v-if="accountsLoading && !topAccounts.length" class="mx-auto my-6" />
          <IssuesAccountsTable
            v-else-if="topAccounts.length"
            :rows="topAccounts"
            :columns="accountColumns"
            :sort-by="accountsSortBy"
            :sort-dir="accountsSortDir"
            @update:sort="onAccountsSort"
          />
          <p v-else-if="!accountsLoading" class="text-sm text-muted-foreground">
            {{ $t('report.issues.noAccounts') }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
const REPORT_ROW_CLASS = 'flex w-full flex-col gap-4 md:flex-row md:flex-wrap'

import { ref, computed, watch, onMounted } from 'vue'
import { useIntervalFn, useDocumentVisibility, watchDebounced } from '@vueuse/core'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { subDays, format } from 'date-fns'
import {
  formatReportDateTime,
  formatReportDayOfWeek,
  formatReportHour24,
  formatResolutionDaysHours,
  reportDateFnsLocale
} from '@/features/reports/issuesFormat.js'
import IssuesBarChart from '@/features/reports/IssuesBarChart.vue'
import IssuesLineChart from '@/features/reports/IssuesLineChart.vue'
import IssuesDonutChart from '@/features/reports/IssuesDonutChart.vue'
import IssuesReportDateRange from '@/features/reports/IssuesReportDateRange.vue'
import IssuesAccountsTable from '@/features/reports/IssuesAccountsTable.vue'
import Spinner from '@shared-ui/components/ui/spinner/Spinner.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input'
import { Separator } from '@shared-ui/components/ui/separator'
import { useI18n } from 'vue-i18n'
import api from '@main/api'

const emitter = useEmitter()
const { t, locale } = useI18n()
const isLoading = ref(false)
const accountsLoading = ref(false)
const lastUpdate = ref(new Date())
const rangeOptions = [7, 30, 90, 120]
const range = ref(120)
const isCustom = ref(false)
const customStart = ref('')
const customEnd = ref('')
const accountSearch = ref('')
const accountsStart = ref('')
const accountsEnd = ref('')
const accountsSortBy = ref('total')
const accountsSortDir = ref('desc')
const ACCOUNTS_DEFAULT_DAYS = 120
const REFRESH_INTERVAL_MS = 5 * 60 * 1000
const MAX_POLL_DURATION_MS = 60 * 60 * 1000
const autoRefreshPaused = ref(false)

const scalars = ref({})
const charts = ref({})
const topAccounts = ref([])

const countLabel = computed(() => t('report.issues.count'))
const touchesLabel = computed(() => t('report.issues.touches'))
const firstResponseLabel = computed(() => t('report.issues.firstResponse'))
const resolutionLabel = computed(() => t('report.issues.resolution'))
const withinWeekLabel = computed(() => t('report.issues.withinWeek'))
const overWeekLabel = computed(() => t('report.issues.overWeek'))

const createdLabel = computed(() => t('report.issues.created'))
const closedLabel = computed(() => t('report.issues.closed'))

const sourceLabels = {
  email: () => t('report.issues.sourceEmail'),
  livechat: () => t('report.issues.sourceChat'),
  whatsapp: () => t('report.issues.sourceWhatsapp')
}

const formatCompactNumber = (value) => {
  const n = Number(value) || 0
  if (n < 1000) return n
  return new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 }).format(n)
}

const scalarCards = computed(() => [
  {
    key: 'all_time_accounts',
    title: t('report.issues.allTimeAccounts'),
    sublabel: t('report.issues.allTime'),
    value: scalars.value.all_time_accounts
  },
  {
    key: 'all_time_issues',
    title: t('report.issues.allTimeIssues'),
    sublabel: t('report.issues.allTime'),
    value: scalars.value.all_time_issues
  },
  {
    key: 'issues_created',
    title: t('report.issues.created'),
    sublabel: t('report.issues.inRange'),
    value: scalars.value.issues_created
  },
  {
    key: 'issues_solved',
    title: t('report.issues.solved'),
    sublabel: t('report.issues.inRange'),
    value: scalars.value.issues_solved
  },
  {
    key: 'issues_unsolved',
    title: t('report.issues.unsolved'),
    sublabel: t('report.issues.inRange'),
    value: scalars.value.issues_unsolved
  },
  {
    key: 'reopened',
    title: t('report.issues.reopened'),
    sublabel: t('report.issues.inRange'),
    value: scalars.value.reopened
  }
])

const mapSourceLabel = (channel) => sourceLabels[channel]?.() ?? channel

const volumeByDayChart = computed(() => {
  const rows = charts.value.volume_by_day || []
  return rows.map((row) => ({
    date: row.date,
    [countLabel.value]: row.count
  }))
})

const reportLocale = computed(() => reportDateFnsLocale(locale.value))

const byHourChart = computed(() => {
  const rows = charts.value.by_hour || []
  return rows.map((row) => ({
    label: formatReportHour24(row.hour, reportLocale.value),
    [countLabel.value]: row.count
  }))
})

const byDowChart = computed(() => {
  const rows = charts.value.by_dow || []
  return rows.map((row) => ({
    label:
      row.dow != null
        ? formatReportDayOfWeek(row.dow, reportLocale.value)
        : row.label,
    [countLabel.value]: row.count
  }))
})

const bySourceChart = computed(() => {
  const rows = charts.value.by_source || []
  return rows.map((row) => ({
    label: mapSourceLabel(row.source),
    [countLabel.value]: row.count
  }))
})

const touchesBySourceChart = computed(() => {
  const rows = charts.value.mean_touches_by_source || []
  return rows.map((row) => ({
    label: mapSourceLabel(row.source),
    [touchesLabel.value]: Number(row.mean_touches) || 0
  }))
})

const volumeOverTimeChart = computed(() => {
  const created = charts.value.volume_over_time?.created || []
  const closed = charts.value.volume_over_time?.closed || []
  const dateMap = new Map()
  created.forEach((item) => {
    dateMap.set(item.date, {
      date: item.date,
      [createdLabel.value]: item.count,
      [closedLabel.value]: 0
    })
  })
  closed.forEach((item) => {
    const existing = dateMap.get(item.date)
    if (existing) {
      existing[closedLabel.value] = item.count
    } else {
      dateMap.set(item.date, {
        date: item.date,
        [createdLabel.value]: 0,
        [closedLabel.value]: item.count
      })
    }
  })
  return Array.from(dateMap.values()).sort((a, b) => new Date(a.date) - new Date(b.date))
})

const tagsByWeekChart = computed(() => {
  const rows = charts.value.tags_by_week || []
  const tagTotals = {}
  rows.forEach((row) => {
    tagTotals[row.tag] = (tagTotals[row.tag] || 0) + row.count
  })
  const topTags = Object.entries(tagTotals)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 8)
    .map(([tag]) => tag)
  const weeks = [...new Set(rows.map((r) => r.week))].sort()
  const categories = topTags
  const data = weeks.map((week) => {
    const point = { week }
    topTags.forEach((tag) => {
      const match = rows.find((r) => r.week === week && r.tag === tag)
      point[tag] = match?.count ?? 0
    })
    return point
  })
  return { data, categories }
})

const medianSlaChart = computed(() => {
  const rows = charts.value.median_sla_by_week || []
  return rows.map((row) => ({
    date: row.week,
    [firstResponseLabel.value]: Number(row.median_first_response_sec) || 0,
    [resolutionLabel.value]: Number(row.median_resolution_sec) || 0
  }))
})

const medianSlaHasData = computed(() => {
  const rows = charts.value.median_sla_by_week || []
  return rows.some(
    (row) =>
      (Number(row.median_first_response_sec) || 0) > 0 ||
      (Number(row.median_resolution_sec) || 0) > 0
  )
})

const solvedWithinWeekChart = computed(() => {
  const rows = charts.value.solved_within_week || []
  return rows.map((row) => ({
    week: row.week,
    [withinWeekLabel.value]: row.within_week,
    [overWeekLabel.value]: row.over_week
  }))
})

const medianTimeInStatusChart = computed(() => {
  const rows = charts.value.median_time_in_status || []
  const statuses = [...new Set(rows.map((r) => r.status))].sort()
  const weeks = [...new Set(rows.map((r) => r.week))].sort()
  const data = weeks.map((week) => {
    const point = { date: week }
    statuses.forEach((status) => {
      const match = rows.find((r) => r.week === week && r.status === status)
      point[status] = Number(match?.median_seconds) || 0
    })
    return point
  })
  return { data, categories: statuses }
})

const accountColumns = computed(() => [
  { key: 'account_name', label: t('report.issues.accountName') },
  { key: 'open', label: t('globals.terms.open') },
  { key: 'closed', label: t('report.issues.closed') },
  { key: 'total', label: t('report.issues.total') },
  {
    key: 'avg_resolution_sec',
    label: t('report.issues.avgResolution'),
    format: (v) => formatResolutionDaysHours(v)
  },
  { key: 'avg_touches', label: t('report.issues.avgTouches') }
])

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
let accountsLoadSeq = 0

const appendIssuesRangeParams = (params, { start, end, days }) => {
  if (start && end) {
    params.start_date = start
    params.end_date = end
  } else if (days != null) {
    params.days = days
  }
  return params
}

const buildReportParams = () =>
  appendIssuesRangeParams(
    {},
    {
      start: isCustom.value ? customStart.value : '',
      end: isCustom.value ? customEnd.value : '',
      days: isCustom.value ? null : range.value
    }
  )

const buildTopAccountsParams = () =>
  appendIssuesRangeParams(
    {
      account_search: accountSearch.value.trim(),
      sort_by: accountsSortBy.value,
      sort_dir: accountsSortDir.value
    },
    {
      start: accountsStart.value,
      end: accountsEnd.value,
      days: accountsStart.value && accountsEnd.value ? null : ACCOUNTS_DEFAULT_DAYS
    }
  )

const mapTopAccountRows = (rows) =>
  (rows || []).map((row) => ({
    ...row,
    account_name: row.account_name || t('report.issues.noAccount')
  }))

const loadReport = async ({ silent = false } = {}) => {
  const seq = ++loadSeq
  if (!silent) {
    isLoading.value = true
  }
  try {
    const { data } = await api.getIssuesReport(buildReportParams())
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

const loadTopAccounts = async ({ silent = false } = {}) => {
  const seq = ++accountsLoadSeq
  if (!silent) {
    accountsLoading.value = true
  }
  try {
    const { data } = await api.getIssuesTopAccounts(buildTopAccountsParams())
    if (seq !== accountsLoadSeq) return
    topAccounts.value = mapTopAccountRows(data.data)
  } catch (error) {
    if (seq !== accountsLoadSeq) return
    showError(error)
  } finally {
    if (seq === accountsLoadSeq) {
      accountsLoading.value = false
    }
  }
}

const refreshAll = ({ silent = false } = {}) =>
  Promise.all([loadReport({ silent }), loadTopAccounts({ silent })])

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

const seedAccountsRange = () => {
  const end = new Date()
  const start = subDays(end, ACCOUNTS_DEFAULT_DAYS)
  accountsEnd.value = format(end, 'yyyy-MM-dd')
  accountsStart.value = format(start, 'yyyy-MM-dd')
}

const onAccountsSort = ({ by, dir }) => {
  if (accountsSortBy.value === by && accountsSortDir.value === dir) return
  accountsSortBy.value = by
  accountsSortDir.value = dir
  markActive()
  loadTopAccounts({ silent: true })
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

let pollingSince = Date.now()

const scheduledRefresh = () => {
  if (Date.now() - pollingSince >= MAX_POLL_DURATION_MS) {
    pause()
    autoRefreshPaused.value = true
    return
  }
  refreshAll({ silent: true })
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
  refreshAll()
}

watch(visibility, (v) => {
  if (v === 'visible') {
    markActive()
    if (Date.now() - lastUpdate.value.getTime() >= REFRESH_INTERVAL_MS) {
      refreshAll({ silent: true })
    }
  } else {
    pause()
  }
})

watchDebounced(
  accountSearch,
  () => {
    loadTopAccounts({ silent: true })
  },
  { debounce: 400 }
)

watchDebounced(
  [customStart, customEnd],
  () => {
    if (isCustom.value && customStart.value && customEnd.value) {
      applyCustomRange()
    }
  },
  { debounce: 350 }
)

watchDebounced(
  [accountsStart, accountsEnd],
  () => {
    if (accountsStart.value && accountsEnd.value && accountsStart.value <= accountsEnd.value) {
      markActive()
      loadTopAccounts({ silent: true })
    }
  },
  { debounce: 350 }
)

onMounted(() => {
  seedAccountsRange()
  refreshAll()
})
</script>
