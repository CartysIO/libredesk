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
    :value-formatter="valueFormatter"
    :title-formatter="resolvedTitleFormatter"
    :margin="margin"
    :rounded-corners="4"
  />
</template>

<script setup>
import { computed } from 'vue'
import { BarChart } from '@shared-ui/components/ui/chart-bar'
import { formatChartCount, formatReportDate } from '@/features/reports/issuesFormat.js'

const props = defineProps({
  data: { type: Array, required: true, default: () => [] },
  categories: { type: Array, required: true },
  index: { type: String, default: 'date' },
  type: { type: String, default: 'grouped' },
  xFormatter: { type: Function, default: null },
  yFormatter: { type: Function, default: null },
  titleFormatter: { type: Function, default: null },
  valueFormatter: { type: Function, default: formatChartCount },
  formatIndexAsDate: { type: Boolean, default: false },
  margin: {
    type: Object,
    default: () => ({ top: 10, bottom: 30, left: 10, right: 10 })
  }
})

const resolvedXFormatter = computed(() => {
  if (props.xFormatter) return props.xFormatter
  return (tick) => {
    const row = props.data[tick]
    if (!row) return ''
    const raw = row[props.index] ?? ''
    return props.formatIndexAsDate ? formatReportDate(raw) : String(raw)
  }
})

const resolvedTitleFormatter = computed(() => {
  if (props.titleFormatter) return props.titleFormatter
  return (title) => (props.formatIndexAsDate ? formatReportDate(title) : String(title ?? ''))
})

const resolvedYFormatter = computed(() => {
  if (props.yFormatter) return props.yFormatter
  return (tick) => {
    if (typeof tick !== 'number' || !Number.isFinite(tick)) return ''
    return formatChartCount(tick)
  }
})
</script>
