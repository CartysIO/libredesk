<template>
  <LineChart
    :data="data"
    index="date"
    :categories="[t('report.chart.newConversations'), t('report.chart.resolvedConversations')]"
    :x-formatter="xFormatter"
    :y-formatter="yFormatter"
    :value-formatter="formatChartCount"
    :title-formatter="titleFormatter"
  />
</template>

<script setup>
import { LineChart } from '@shared-ui/components/ui/chart-line'
import { useI18n } from 'vue-i18n'
import { formatChartCount } from '@/features/reports/issuesFormat.js'
import { useReportChartIndexFormatters } from '@/features/reports/useReportChartIndexFormatters.js'

const props = defineProps({
  data: {
    type: Array,
    default: () => []
  }
})
const { t } = useI18n()

const indexFormatProps = { formatIndexAsDate: true, formatIndexAsWeek: false }
const { formatIndexAxisValue, formatIndexTitleValue } =
  useReportChartIndexFormatters(indexFormatProps)

const xFormatter = (tick) => {
  const raw = props.data[tick]?.date ?? ''
  return formatIndexAxisValue(raw)
}

const titleFormatter = (title) => formatIndexTitleValue(title)

const yFormatter = (tick) => {
  if (typeof tick !== 'number' || !Number.isFinite(tick)) return ''
  return formatChartCount(tick)
}
</script>