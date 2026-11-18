<template>
  <div class="min-h-0 flex-1 overflow-y-auto">
    <div
      class="desk-reports w-full px-3 py-4"
      :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
    >
      <Spinner v-if="isLoading && !rows.length" />

      <div class="desk-report-sections flex flex-col gap-6">
        <div class="mb-1 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground">
              {{ $t('globals.terms.lastUpdated') }}: {{ lastUpdateFormatted }}
            </span>
            <span class="text-xs text-muted-foreground">{{ $t('report.slaBreaches.dataDelay') }}</span>
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

        <div class="box w-full p-5">
          <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p class="ui-heading">{{ $t('report.slaBreaches.title') }}</p>
            <Input
              v-model="search"
              class="max-w-xs"
              :placeholder="$t('report.slaBreaches.search')"
              :disabled="isLoading"
            />
          </div>
          <SlaBreachesTable
            v-if="rows.length"
            :rows="rows"
            :columns="columns"
            :sort-by="sortBy"
            :sort-dir="sortDir"
            @update:sort="onSort"
          />
          <p v-else-if="!isLoading" class="text-sm text-muted-foreground">
            {{ $t('report.slaBreaches.empty') }}
          </p>
          <PaginationBar
            v-if="total > 0"
            v-model:page="page"
            v-model:per-page="perPage"
            :total-pages="totalPages"
            :per-page-options="[25, 50, 100]"
            class="mt-4 border-t border-border/70 pt-3"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useDocumentVisibility, useIntervalFn, watchDebounced } from '@vueuse/core'
import { subDays, format } from 'date-fns'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import {
  formatReportDateTime,
  formatReportDateTimeReadable,
  formatResolutionDaysHours
} from '@/features/reports/issuesFormat.js'
import IssuesReportDateRange from '@/features/reports/IssuesReportDateRange.vue'
import SlaBreachesTable from '@/features/reports/SlaBreachesTable.vue'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
import Spinner from '@shared-ui/components/ui/spinner/Spinner.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input'
import { Separator } from '@shared-ui/components/ui/separator'
import { useI18n } from 'vue-i18n'
import api from '@main/api'

const REFRESH_INTERVAL_MS = 5 * 60 * 1000
const MAX_POLL_DURATION_MS = 60 * 60 * 1000

const emitter = useEmitter()
const { t } = useI18n()
const isLoading = ref(false)
const lastUpdate = ref(new Date())
const rangeOptions = [7, 30, 90, 120]
const range = ref(90)
const isCustom = ref(false)
const customStart = ref('')
const customEnd = ref('')
const autoRefreshPaused = ref(false)
const search = ref('')
const sortBy = ref('created_at')
const sortDir = ref('desc')
const page = ref(1)
const perPage = ref(50)
const total = ref(0)
const totalPages = ref(0)
const rows = ref([])

const columns = computed(() => [
  { key: 'reference_number', label: t('report.slaBreaches.conversationNumber') },
  {
    key: 'created_at',
    label: t('report.slaBreaches.conversationCreatedAt'),
    format: (v) => formatReportDateTimeReadable(v)
  },
  {
    key: 'subject',
    label: t('report.slaBreaches.conversationTitle'),
    format: (v) => v || '—'
  },
  {
    key: 'first_response_sec',
    label: t('report.slaBreaches.firstResponseTime'),
    numeric: true,
    format: (v) => formatResolutionDaysHours(v)
  },
  {
    key: 'resolution_sec',
    label: t('report.slaBreaches.resolutionTime'),
    numeric: true,
    format: (v) => formatResolutionDaysHours(v)
  },
  {
    key: 'max_next_response_sec',
    label: t('report.slaBreaches.maxNextResponseTime'),
    numeric: true,
    format: (v) => formatResolutionDaysHours(v)
  },
  {
    key: 'organization_name',
    label: t('globals.terms.organization'),
    format: (v) => v || t('report.slaBreaches.noOrganization')
  }
])

const lastUpdateFormatted = computed(() => formatReportDateTime(lastUpdate.value))

const showError = (error) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })
}

let loadSeq = 0

const appendRangeParams = (params) => {
  if (isCustom.value && customStart.value && customEnd.value) {
    params.start_date = customStart.value
    params.end_date = customEnd.value
  } else {
    params.days = range.value
  }
  return params
}

const buildParams = () =>
  appendRangeParams({
    search: search.value.trim(),
    sort_by: sortBy.value,
    sort_dir: sortDir.value,
    page: page.value,
    page_size: perPage.value
  })

const load = async ({ silent = false } = {}) => {
  const seq = ++loadSeq
  if (!silent) {
    isLoading.value = true
  }
  try {
    const { data } = await api.getSlaBreachesReport(buildParams())
    if (seq !== loadSeq) return
    rows.value = data.data?.results || []
    total.value = data.data?.total ?? 0
    totalPages.value = data.data?.total_pages ?? 0
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
  page.value = 1
  load()
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
  page.value = 1
  load()
}

const onSort = ({ by, dir }) => {
  if (sortBy.value === by && sortDir.value === dir) return
  sortBy.value = by
  sortDir.value = dir
  page.value = 1
  markActive()
  load({ silent: true })
}

let pollingSince = Date.now()

const scheduledRefresh = () => {
  if (Date.now() - pollingSince >= MAX_POLL_DURATION_MS) {
    pause()
    autoRefreshPaused.value = true
    return
  }
  load({ silent: true })
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
  load()
}

watch(visibility, (v) => {
  if (v === 'visible') {
    markActive()
    if (Date.now() - lastUpdate.value.getTime() >= REFRESH_INTERVAL_MS) {
      load({ silent: true })
    }
  } else {
    pause()
  }
})

watchDebounced(
  search,
  () => {
    page.value = 1
    load({ silent: true })
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

watch([page, perPage], () => {
  markActive()
  load({ silent: true })
})

onMounted(() => {
  load()
})
</script>