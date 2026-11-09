<template>
  <BarChart
    :data="data"
    :index="index"
    :categories="categories"
    :show-grid-line="true"
    :show-x-axis="true"
    :show-y-axis="true"
    :type="type"
    :x-formatter="xFormatter"
    :y-formatter="yFormatter"
    :rounded-corners="4"
  />
</template>

<script setup>
import { BarChart } from '@shared-ui/components/ui/chart-bar'

const props = defineProps({
  data: { type: Array, required: true, default: () => [] },
  categories: { type: Array, required: true },
  index: { type: String, default: 'date' },
  type: { type: String, default: 'grouped' }
})

const xFormatter = (tick) => {
  const row = props.data[tick]
  return row ? row[props.index] ?? '' : ''
}

const yFormatter = (tick) => (Number.isInteger(tick) ? tick : '')
</script>
