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
          <circle
            r="70"
            fill="hsl(var(--background))"
            stroke="hsl(var(--border))"
            stroke-width="1"
          />
          <path
            v-for="(seg, i) in arcPaths"
            :key="i"
            :d="seg.d"
            :fill="seg.color"
            class="transition-opacity cursor-pointer"
            :opacity="hoverIndex === null || hoverIndex === i ? 1 : 0.35"
            @mouseenter="showTip($event, seg, i)"
            @mouseleave="hideTip"
          />
          <circle r="42" fill="hsl(var(--background))" />
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
        <Card class="text-sm shadow-md border">
          <CardHeader class="p-3 border-b">
            <CardTitle class="text-sm font-medium">{{ tip.label }}</CardTitle>
          </CardHeader>
          <CardContent class="p-3 flex justify-between gap-4">
            <span class="text-muted-foreground">{{ valueLabel }}</span>
            <span class="font-semibold tabular-nums">{{ tip.value }}</span>
          </CardContent>
        </Card>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ChartLegend, defaultColors } from '@shared-ui/components/ui/chart'
import { Card, CardContent, CardHeader, CardTitle } from '@shared-ui/components/ui/card'

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

const total = computed(() =>
  props.data.reduce((sum, row) => sum + (Number(row[props.valueKey]) || 0), 0)
)

const colors = computed(() => defaultColors(Math.max(props.data.length, 1)))

const legendItems = ref([])
watch(
  () => props.data,
  (data) => {
    const cols = defaultColors(Math.max(data.length, 1))
    legendItems.value = data.map((row, i) => ({
      name: row[props.labelKey],
      color: cols[i],
      inactive: false
    }))
  },
  { immediate: true, deep: true }
)

const polarToCartesian = (cx, cy, r, angleDeg) => {
  const rad = ((angleDeg - 90) * Math.PI) / 180
  return { x: cx + r * Math.cos(rad), y: cy + r * Math.sin(rad) }
}

const arcPath = (startAngle, endAngle, outerR = 70, innerR = 42) => {
  const start = polarToCartesian(0, 0, outerR, endAngle)
  const end = polarToCartesian(0, 0, outerR, startAngle)
  const innerStart = polarToCartesian(0, 0, innerR, endAngle)
  const innerEnd = polarToCartesian(0, 0, innerR, startAngle)
  const largeArc = endAngle - startAngle <= 180 ? '0' : '1'
  return [
    `M ${start.x} ${start.y}`,
    `A ${outerR} ${outerR} 0 ${largeArc} 0 ${end.x} ${end.y}`,
    `L ${innerEnd.x} ${innerEnd.y}`,
    `A ${innerR} ${innerR} 0 ${largeArc} 1 ${innerStart.x} ${innerStart.y}`,
    'Z'
  ].join(' ')
}

const arcPaths = computed(() => {
  let cursor = 0
  return props.data.map((row, i) => {
    const value = Number(row[props.valueKey]) || 0
    const slice = total.value ? (value / total.value) * 360 : 0
    const start = cursor
    cursor += slice
    return {
      d: arcPath(start, cursor),
      color: colors.value[i],
      label: row[props.labelKey],
      value
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
