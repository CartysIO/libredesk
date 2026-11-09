<script setup>
import { VisCrosshair, VisTooltip } from '@unovis/vue'
import { watch } from 'vue'
import { ChartTooltip } from '.'
import { renderTooltip, seriesFromDatum } from './tooltipSeries'

const props = defineProps({
  colors: { type: Array, required: true, default: () => [] },
  index: { type: String, required: true },
  items: { type: Array, required: true },
  customTooltip: { type: null, required: false },
  valueFormatter: {
    type: Function,
    required: false,
    default: (value) => (value == null ? '' : value)
  },
  titleFormatter: {
    type: Function,
    required: false,
    default: (title) => (title == null ? '' : String(title))
  }
})

// Drop a cached tooltip when the legend catches up, so a hover during load
// cannot keep the series name blank.
const wm = new WeakMap()
let cacheVersion = 0
watch(
  () => props.items,
  () => {
    cacheVersion += 1
  }
)

function template(d) {
  const cached = wm.get(d)
  if (cached?.version === cacheVersion) return cached.html

  const TooltipComponent = props.customTooltip ?? ChartTooltip
  const html = renderTooltip(
    TooltipComponent,
    props.titleFormatter(d[props.index]),
    seriesFromDatum(d, props.index, props.items, props.colors).map((row) => ({
      ...row,
      value: props.valueFormatter(row.value)
    }))
  )
  wm.set(d, { version: cacheVersion, html })
  return html
}

function color(d, i) {
  return props.colors[i] ?? 'transparent'
}
</script>

<template>
  <VisTooltip :horizontal-shift="20" :vertical-shift="20" class="unovis-tooltip !z-[80]" />
  <VisCrosshair :template="template" :color="color" />
</template>
