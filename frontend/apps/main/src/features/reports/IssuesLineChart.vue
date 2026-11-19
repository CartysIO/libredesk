<template>
  <LineChart
    :data="data"
    :index="indexField"
    :categories="categories"
    :x-formatter="resolvedXFormatter"
    :y-formatter="resolvedYFormatter"
    :value-formatter="resolvedValueFormatter"
    :title-formatter="resolvedTitleFormatter"
    :margin="{ top: 10, bottom: 30, left: 10, right: 20 }"
  />
</template>

<script setup>
import { computed } from 'vue'
import { LineChart } from '@shared-ui/components/ui/chart-line'
import {
  formatChartCount,
  formatCompactDuration,
  formatResolutionDaysHours
} from '@/features/reports/issuesFormat.js'
import { useReportChartIndexFormatters } from '@/features/reports/useReportChartIndexFormatters.js'

const props = defineProps({
  data: { type: Array, default: () => [] },
  categories: { type: Array, required: true },
  indexField: { type: String, default: 'date' },
  xFormatter: { type: Function, default: null },
  yFormatter: { type: Function, default: null },
  titleFormatter: { type: Function, default: null },
  valueFormatter: { type: Function, default: null },
  durationAxis: { type: Boolean, default: false },
  formatIndexAsDate: { type: Boolean, default: false },
  formatIndexAsWeek: { type: Boolean, default: false }
})

const { formatIndexAxisValue, formatIndexTitleValue } = useReportChartIndexFormatters(props)

const resolvedXFormatter = computed(() => {
  if (props.xFormatter) return props.xFormatter
  return (tick) => {
    const raw = props.data[tick]?.[props.indexField] ?? ''
    return formatIndexAxisValue(raw)
  }
})

const resolvedTitleFormatter = computed(() => {
  if (props.titleFormatter) return props.titleFormatter
  return (title) => formatIndexTitleValue(title)
})

const resolvedYFormatter = computed(() => {
  if (props.yFormatter) return props.yFormatter
  if (props.durationAxis) return (tick) => formatCompactDuration(tick)
  return (tick) => {
    if (typeof tick !== 'number' || !Number.isFinite(tick)) return ''
    return formatChartCount(tick)
  }
})

const resolvedValueFormatter = computed(() => {
  if (props.valueFormatter) return props.valueFormatter
  if (props.durationAxis) return formatResolutionDaysHours
  return formatChartCount
})
</script>