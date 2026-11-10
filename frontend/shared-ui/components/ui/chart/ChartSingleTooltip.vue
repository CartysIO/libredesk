<script setup>
import { VisTooltip } from '@unovis/vue'
import { ChartTooltip } from '.'
import { renderTooltip, seriesFromDatum } from './tooltipSeries'
import { createVersionedTooltipCache } from './tooltipCache'

const props = defineProps({
  selector: { type: String, required: true },
  index: { type: String, required: true },
  items: { type: Array, required: false },
  valueFormatter: { type: Function, required: false, default: (tick) => `${tick}` },
  customTooltip: { type: null, required: false }
})

const tooltipCache = createVersionedTooltipCache(() => props.items)

function template(d, i, elements) {
  if (props.index in d) {
    return tooltipCache.remember(d, () => {
      const TooltipComponent = props.customTooltip ?? ChartTooltip
      return renderTooltip(
        TooltipComponent,
        d[props.index],
        seriesFromDatum(d, props.index, props.items).map((row) => ({
          ...row,
          value: props.valueFormatter(row.value)
        }))
      )
    })
  }

  const data = d.data
  return tooltipCache.remember(data, () => {
    const style = getComputedStyle(elements[i])
    const TooltipComponent = props.customTooltip ?? ChartTooltip
    return renderTooltip(TooltipComponent, d[props.index], [
      {
        name: data.name,
        value: props.valueFormatter(data[props.index]),
        color: style.fill
      }
    ])
  })
}
</script>

<template>
  <VisTooltip
    :horizontal-shift="20"
    :vertical-shift="20"
    class="unovis-tooltip !z-[80]"
    :triggers="{
      [selector]: template
    }"
  />
</template>
