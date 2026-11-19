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
import { formatChartCount, formatReportDate, reportDateFnsLocale } from '@/features/reports/issuesFormat.js'

const props = defineProps({
  data: {
    type: Array,
    default: () => []
  }
})
const { t, locale } = useI18n()

const dateLocale = () => reportDateFnsLocale(locale.value)

const xFormatter = (tick) => {
  const raw = props.data[tick]?.date ?? ''
  return formatReportDate(raw, dateLocale())
}

const titleFormatter = (title) => formatReportDate(title, dateLocale())

const yFormatter = (tick) => {
  if (typeof tick !== 'number' || !Number.isFinite(tick)) return ''
  return formatChartCount(tick)
}
</script>