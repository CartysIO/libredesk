<template>
  <BarChart
    :data="data"
    :index="index"
    :categories="categories"
    :show-grid-line="true"
    :show-x-axis="true"
    :show-y-axis="true"
    :type="type"
    :x-formatter="resolvedXFormatter"
    :y-formatter="resolvedYFormatter"
    :value-formatter="resolvedValueFormatter"
    :title-formatter="resolvedTitleFormatter"
    :margin="margin"
    :rounded-corners="4"
  />
</template>

<script setup>
import { computed } from 'vue'
import { BarChart } from '@shared-ui/components/ui/chart-bar'
import {
  formatChartCount,
  formatCompactDuration,
  formatResolutionDaysHours
} from '@/features/reports/issuesFormat.js'
import { useReportChartIndexFormatters } from '@/features/reports/useReportChartIndexFormatters.js'

const props = defineProps({
  data: { type: Array, required: true, default: () => [] },
  categories: { type: Array, required: true },
  index: { type: String, default: 'date' },
  type: { type: String, default: 'grouped' },
  xFormatter: { type: Function, default: null },
  yFormatter: { type: Function, default: null },
  titleFormatter: { type: Function, default: null },
  valueFormatter: { type: Function, default: null },
  formatIndexAsDate: { type: Boolean, default: false },
  formatIndexAsWeek: { type: Boolean, default: false },
  durationYAxis: { type: Boolean, default: false },
  margin: {
    type: Object,
    default: () => ({ top: 10, bottom: 30, left: 10, right: 10 })
  }
})

const { formatIndexAxisValue, formatIndexTitleValue } = useReportChartIndexFormatters(props)

const resolvedXFormatter = computed(() => {
  if (props.xFormatter) return props.xFormatter
  return (tick) => {
    const row = props.data[tick]
    if (!row) return ''
    const raw = row[props.index] ?? ''
    return formatIndexAxisValue(raw)
  }
})

const resolvedTitleFormatter = computed(() => {
  if (props.titleFormatter) return props.titleFormatter
  return (title) => formatIndexTitleValue(title)
})

const resolvedYFormatter = computed(() => {
  if (props.yFormatter) return props.yFormatter
  if (props.durationYAxis) return (tick) => formatCompactDuration(tick)
  return (tick) => {
    if (typeof tick !== 'number' || !Number.isFinite(tick)) return ''
    return formatChartCount(tick)
  }
})

const resolvedValueFormatter = computed(() => {
  if (props.valueFormatter) return props.valueFormatter
  if (props.durationYAxis) return formatResolutionDaysHours
  return formatChartCount
})
</script>