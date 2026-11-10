<template>
  <div class="w-full h-[400px] flex flex-col items-end">
    <ChartLegend v-if="showLegend" v-model:items="legendItems" />

    <div class="relative flex-1 w-full flex items-center justify-center min-h-0">
      <svg
        v-if="total > 0"
        viewBox="0 0 200 200"
        class="max-h-full max-w-full w-auto h-auto"
        role="img"
        :aria-label="ariaLabel"
      >
        <g transform="translate(100, 100)">
          <path
            v-for="(seg, i) in arcPaths"
            :key="i"
            :d="seg.d"
            :style="{ fill: seg.color }"
            class="transition-opacity cursor-pointer"
            :opacity="hoverIndex === null || hoverIndex === i ? 1 : 0.35"
            @mouseenter="showTip($event, seg, i)"
            @mouseleave="hideTip"
          />
          <circle r="42" fill="hsl(var(--background))" />
          <text
            v-for="(seg, i) in arcPaths"
            v-show="seg.showLabel"
            :key="`lbl-${i}`"
            :x="seg.labelX"
            :y="seg.labelY"
            text-anchor="middle"
            dominant-baseline="middle"
            class="fill-white text-[7px] font-semibold pointer-events-none"
          >
            {{ seg.shortLabel }}
          </text>
          <text
            text-anchor="middle"
            dominant-baseline="middle"
            class="fill-foreground text-[11px] font-semibold"
            y="-4"
          >
            {{ total }}
          </text>
          <text
            text-anchor="middle"
            dominant-baseline="middle"
            class="fill-muted-foreground text-[8px]"
            y="10"
          >
            {{ centerLabel }}
          </text>
        </g>
      </svg>
      <p v-else class="text-sm text-muted-foreground">{{ emptyLabel }}</p>

      <div
        v-if="tip"
        class="pointer-events-none absolute z-50"
        :style="{ left: `${tip.x}px`, top: `${tip.y}px` }"
      >
        <div
          class="min-w-[10rem] rounded-sm border border-border bg-popover px-3 py-2 text-sm shadow-lg"
        >
          <p class="font-medium">{{ tip.label }}</p>
          <p class="mt-1 flex justify-between gap-4 text-muted-foreground">
            <span>{{ valueLabel }}</span>
            <span class="font-semibold tabular-nums text-foreground">{{ tip.value }}</span>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { ChartLegend, useLegendItems } from '@shared-ui/components/ui/chart'
import { donutArcPath, donutSegmentColors, polarToCartesian } from './donutSegments.js'

const OUTER_R = 70
const INNER_R = 42
const LABEL_MIN_DEG = 22

const props = defineProps({
  data: { type: Array, default: () => [] },
  valueKey: { type: String, required: true },
  labelKey: { type: String, default: 'label' },
  valueLabel: { type: String, default: '' },
  centerLabel: { type: String, default: '' },
  emptyLabel: { type: String, default: '' },
  ariaLabel: { type: String, default: '' },
  showLegend: { type: Boolean, default: true }
})

const hoverIndex = ref(null)
const tip = ref(null)

const segments = computed(() =>
  props.data
    .map((row, index) => ({
      index,
      label: row[props.labelKey],
      value: Number(row[props.valueKey]) || 0
    }))
    .filter((row) => row.value > 0)
)

const total = computed(() => segments.value.reduce((sum, row) => sum + row.value, 0))

const colors = computed(() => donutSegmentColors(segments.value.length))

const legendCategories = computed(() => segments.value.map((row) => row.label))
const legendItems = useLegendItems(() => legendCategories.value, colors)

const arcPaths = computed(() => {
  const t = total.value
  if (!t) return []

  let cursor = 0
  const labelR = (OUTER_R + INNER_R) / 2

  return segments.value.map((row, i) => {
    const slice = (row.value / t) * 360
    const start = cursor
    const end = cursor + slice
    cursor = end
    const mid = start + slice / 2
    const pct = Math.round((row.value / t) * 100)
    const labelPos = polarToCartesian(0, 0, labelR, mid)

    return {
      d: donutArcPath(start, end, OUTER_R, INNER_R),
      color: colors.value[i],
      label: row.label,
      value: row.value,
      showLabel: slice >= LABEL_MIN_DEG,
      labelX: labelPos.x,
      labelY: labelPos.y,
      shortLabel: `${pct}%`
    }
  })
})

const showTip = (event, seg, index) => {
  hoverIndex.value = index
  const rect = event.target.ownerSVGElement?.closest('.relative')?.getBoundingClientRect()
  const x = event.clientX - (rect?.left ?? 0) + 12
  const y = event.clientY - (rect?.top ?? 0) + 12
  tip.value = { label: seg.label, value: seg.value, x, y }
}

const hideTip = () => {
  hoverIndex.value = null
  tip.value = null
}
</script>
