<template>
  <div class="desk-report-panel flex flex-1 flex-col gap-4 p-5">
    <div class="flex items-center justify-between">
      <h3 class="ui-heading">{{ title }}</h3>
      <slot name="header-right"></slot>
    </div>
    <div :class="gridClass">
      <div
        v-for="(item, key) in filteredCounts"
        :key="key"
        class="flex flex-col items-center gap-1 text-center"
      >
        <span :class="valueClass">{{ item }}</span>
        <span class="text-xs text-muted-foreground">{{ labels[key] }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  counts: { type: Object, required: true },
  labels: { type: Object, required: true },
  title: { type: String, required: true },
  size: { type: String, default: 'default' }, // 'default' | 'large'
  columns: { type: Number, default: 4 }
})

// Filter out counts that don't have a label
const filteredCounts = computed(() => {
  return Object.fromEntries(Object.entries(props.counts).filter(([key]) => props.labels[key]))
})

const gridClass = computed(() => {
  const cols = {
    2: 'grid-cols-1 sm:grid-cols-2',
    3: 'grid-cols-1 sm:grid-cols-3',
    4: 'grid-cols-1 sm:grid-cols-2 md:grid-cols-4',
    5: 'grid-cols-1 sm:grid-cols-2 md:grid-cols-5'
  }
  return `grid gap-6 ${cols[props.columns] || cols[4]}`
})

const valueClass = computed(() => {
  const sizes = {
    default: 'text-sm font-bold tabular-nums',
    large: 'desk-report-metric tracking-tight'
  }
  return sizes[props.size] || sizes.default
})
</script>
