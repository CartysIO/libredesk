<script setup>
import { VisCrosshair, VisTooltip } from '@unovis/vue'
import { ChartTooltip } from '.'
import { renderTooltip, seriesFromDatum } from './tooltipSeries'
import { createVersionedTooltipCache } from './tooltipCache'

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

const tooltipCache = createVersionedTooltipCache(() => props.items)

function template(d) {
  return tooltipCache.remember(d, () => {
    const TooltipComponent = props.customTooltip ?? ChartTooltip
    return renderTooltip(
      TooltipComponent,
      props.titleFormatter(d[props.index]),
      seriesFromDatum(d, props.index, props.items, props.colors).map((row) => ({
        ...row,
        value: props.valueFormatter(row.value)
      }))
    )
  })
}

function color(d, i) {
  return props.colors[i] ?? 'transparent'
}
</script>

<template>
  <VisTooltip :horizontal-shift="20" :vertical-shift="20" class="unovis-tooltip !z-[80]" />
  <VisCrosshair :template="template" :color="color" />
</template>
