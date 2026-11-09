<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        type="button"
        size="sm"
        variant="outline"
        :disabled="disabled"
        :class="cn('h-8 min-w-[12rem] justify-start font-normal', !rangeLabel && 'text-muted-foreground')"
      >
        <CalendarIcon class="mr-2 h-4 w-4 shrink-0" aria-hidden="true" />
        <span class="truncate">{{ rangeLabel || $t('globals.terms.pickDate') }}</span>
      </Button>
    </PopoverTrigger>
    <PopoverContent class="w-auto p-0" align="end">
      <RangeCalendar
        :model-value="calendarRange"
        :number-of-months="2"
        :disabled="disabled"
        @update:model-value="onRangePick"
      />
    </PopoverContent>
  </Popover>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Calendar as CalendarIcon } from 'lucide-vue-next'
import { parseDate } from '@internationalized/date'
import { cn } from '@shared-ui/lib/utils.js'
import { Button } from '@shared-ui/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { RangeCalendar } from '@shared-ui/components/ui/range-calendar'
import { formatReportDate } from '@/features/reports/issuesFormat.js'

const props = defineProps({
  disabled: { type: Boolean, default: false }
})

const start = defineModel('start', { type: String, default: '' })
const end = defineModel('end', { type: String, default: '' })

const open = ref(false)

const toCalendarDate = (v) => {
  if (!v) return undefined
  try {
    return parseDate(v)
  } catch {
    return undefined
  }
}

const calendarRange = computed(() => ({
  start: toCalendarDate(start.value),
  end: toCalendarDate(end.value)
}))

const rangeLabel = computed(() => {
  if (!start.value && !end.value) return ''
  if (start.value && end.value) {
    return `${formatReportDate(start.value)} – ${formatReportDate(end.value)}`
  }
  return formatReportDate(start.value || end.value)
})

const onRangePick = (v) => {
  if (!v?.start || !v?.end) return
  start.value = v.start.toString()
  end.value = v.end.toString()
  open.value = false
}

watch(
  () => [start.value, end.value],
  ([s, e]) => {
    if (s && e && s > e) {
      end.value = s
    }
  }
)
</script>
