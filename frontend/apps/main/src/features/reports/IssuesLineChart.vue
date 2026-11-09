<template>
  <LineChart
    :data="data"
    index="date"
    :categories="categories"
    :x-formatter="xFormatter"
    :y-formatter="yFormatter"
  />
</template>

<script setup>
import { LineChart } from '@shared-ui/components/ui/chart-line'

const props = defineProps({
  data: { type: Array, default: () => [] },
  categories: { type: Array, required: true }
})

const xFormatter = (tick) => props.data[tick]?.date ?? ''

const yFormatter = (tick) => (Number.isInteger(tick) ? tick : '')
</script>
