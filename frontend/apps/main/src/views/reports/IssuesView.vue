<template>
  <div class="overflow-y-auto">
    <div
      class="p-4 md:p-6 w-full md:w-[calc(100%-3rem)]"
      :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
    >
      <Spinner v-if="isLoading" />

      <div class="space-y-6">
        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
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
          <div class="flex flex-wrap items-center gap-1">
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
            <Input
              v-if="isCustom"
              v-model="customDays"
              :aria-label="$t('report.customRangeDays')"
              type="number"
              min="1"
              max="365"
              class="w-20 h-8"
              :disabled="isLoading"
              @blur="applyCustom"
              @keyup.enter="applyCustom"
            />
          </div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
          <div
            v-for="scalar in scalarCards"
            :key="scalar.key"
            class="box p-5"
          >
            <p class="text-xs text-muted-foreground mb-1">{{ scalar.sublabel }}</p>
            <p class="ui-heading mb-2 text-base">{{ scalar.title }}</p>
            <span class="text-3xl font-bold tabular-nums">{{
              formatCompactNumber(scalar.value)
            }}</span>
          </div>
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.issues.volumeByDay') }}</p>
          <IssuesBarChart :data="volumeByDayChart" :categories="[countLabel]" index="date" />
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
            <IssuesBarChart :data="bySourceChart" :categories="[countLabel]" index="label" />
          </div>
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.issues.meanTouchesBySource') }}</p>
            <IssuesBarChart :data="touchesBySourceChart" :categories="[touchesLabel]" index="label" />
          </div>
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.issues.volumeOverTime') }}</p>
          <IssuesLineChart
            :data="volumeOverTimeChart"
            :categories="[createdLabel, closedLabel]"
          />
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.issues.tagsByWeek') }}</p>
          <IssuesBarChart
            :data="tagsByWeekChart.data"
            :categories="tagsByWeekChart.categories"
            index="week"
            type="stacked"
          />
        </div>

        <div :class="REPORT_ROW_CLASS">
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.issues.medianResponseResolution') }}</p>
            <IssuesLineChart
              :data="medianSlaChart"
              :categories="[firstResponseLabel, resolutionLabel]"
            />
          </div>
          <div class="flex-1 box p-5">
            <p class="ui-heading mb-4">{{ $t('report.issues.solvedWithinWeek') }}</p>
            <IssuesBarChart
              :data="solvedWithinWeekChart"
              :categories="[withinWeekLabel, overWeekLabel]"
              index="week"
              type="stacked"
            />
          </div>
        </div>

        <div class="box w-full p-5">
          <p class="ui-heading mb-4">{{ $t('report.issues.medianTimeInStatus') }}</p>
          <IssuesLineChart
            :data="medianTimeInStatusChart.data"
            :categories="medianTimeInStatusChart.categories"
          />
        </div>

        <div class="box w-full p-5">
          <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mb-4">
            <p class="ui-heading">{{ $t('report.issues.topAccounts') }}</p>
            <Input
              v-model="accountSearch"
              class="max-w-xs"
              :placeholder="$t('report.issues.searchAccounts')"
              :disabled="isLoading"
            />
          </div>
          <IssuesAccountsTable
            v-if="topAccounts.length"
            :rows="topAccounts"
            :columns="accountColumns"
          />
          <p v-else class="text-sm text-muted-foreground">{{ $t('report.issues.noAccounts') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
const REPORT_ROW_CLASS = 'flex flex-col w-full gap-4 md:flex-row md:flex-wrap'

import { ref, computed, watch, onMounted } from 'vue'
import { useIntervalFn, useDocumentVisibility, watchDebounced } from '@vueuse/core'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { formatDuration } from '@shared-ui/utils/datetime.js'
import IssuesBarChart from '@/features/reports/IssuesBarChart.vue'
import IssuesLineChart from '@/features/reports/IssuesLineChart.vue'
import IssuesAccountsTable from '@/features/reports/IssuesAccountsTable.vue'
import Spinner from '@shared-ui/components/ui/spinner/Spinner.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input'
import { Separator } from '@shared-ui/components/ui/separator'
import { useI18n } from 'vue-i18n'
import api from '@main/api'

const emitter = useEmitter()
const { t } = useI18n()
const isLoading = ref(false)
const lastUpdate = ref(new Date())
const rangeOptions = [7, 30, 90, 120]
const range = ref(120)
const isCustom = ref(false)
const customDays = ref('')
const accountSearch = ref('')
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

const byHourChart = computed(() => {
  const rows = charts.value.by_hour || []
  return rows.map((row) => ({
    label: String(row.hour).padStart(2, '0') + ':00',
    [countLabel.value]: row.count
  }))
})

const byDowChart = computed(() => {
  const rows = charts.value.by_dow || []
  return rows.map((row) => ({
    label: row.label,
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
    [firstResponseLabel.value]: Math.round(row.median_first_response_sec || 0),
    [resolutionLabel.value]: Math.round(row.median_resolution_sec || 0)
  }))
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
      point[status] = Math.round(match?.median_seconds || 0)
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
    format: (v) => formatDuration(v, false)
  },
  { key: 'avg_touches', label: t('report.issues.avgTouches') }
])

const lastUpdateFormatted = computed(() => lastUpdate.value.toLocaleTimeString())

const showError = (error) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })
}

const loadReport = async () => {
  isLoading.value = true
  try {
    const { data } = await api.getIssuesReport({
      days: range.value,
      account_search: accountSearch.value.trim()
    })
    scalars.value = data.data.scalars || {}
    charts.value = data.data.charts || {}
    topAccounts.value = (data.data.top_accounts || []).map((row) => ({
      ...row,
      account_name: row.account_name || t('report.issues.noAccount')
    }))
    lastUpdate.value = new Date()
  } catch (error) {
    showError(error)
  } finally {
    isLoading.value = false
  }
}

const selectRange = (option) => {
  isCustom.value = false
  if (option === range.value) return
  markActive()
  range.value = option
  loadReport()
}

const enableCustom = () => {
  isCustom.value = true
  customDays.value = String(range.value)
}

const applyCustom = () => {
  let days = parseInt(customDays.value, 10)
  if (!days || days < 1 || days > 365) {
    days = 120
    customDays.value = '120'
  }
  if (days === range.value) return
  markActive()
  range.value = days
  loadReport()
}

let pollingSince = Date.now()

const scheduledRefresh = () => {
  if (Date.now() - pollingSince >= MAX_POLL_DURATION_MS) {
    pause()
    autoRefreshPaused.value = true
    return
  }
  loadReport()
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
    if (Date.now() - lastUpdate.value.getTime() >= REFRESH_INTERVAL_MS) loadReport()
  } else {
    pause()
  }
})

watchDebounced(
  accountSearch,
  () => {
    if (!isLoading.value) loadReport()
  },
  { debounce: 400 }
)

onMounted(loadReport)
</script>
