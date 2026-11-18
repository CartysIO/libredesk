<template>
  <div class="desk-inbox-date-range flex flex-wrap items-center gap-1">
    <span class="text-muted-foreground mr-1 text-xs">{{ $t('globals.terms.conversationDate') }}</span>
    <Button
      v-for="option in rangeOptions"
      :key="option"
      type="button"
      size="xs"
      :variant="!isCustom && presetDays === option ? 'secondary' : 'ghost'"
      class="h-7 px-2 text-xs"
      @click="selectPreset(option)"
    >
      {{ $t('globals.messages.nDays', { days: option }) }}
    </Button>
    <Button
      type="button"
      size="xs"
      :variant="isCustom ? 'secondary' : 'ghost'"
      class="h-7 px-2 text-xs"
      @click="enableCustom"
    >
      {{ $t('globals.terms.custom') }}
    </Button>
    <DateFilterValue
      v-if="isCustom"
      v-model="customRange"
      range
      :placeholder="$t('globals.terms.pickDate')"
      class="min-w-[10rem] flex-1 [&_button]:h-7 [&_button]:text-xs"
    />
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import { watchDebounced } from '@vueuse/core'
import { subDays, format } from 'date-fns'
import { Button } from '@shared-ui/components/ui/button'
import DateFilterValue from '@/components/filter/DateFilterValue.vue'
import { INBOX_ALL_RANGE_OPTIONS } from '@/utils/inboxAllDateRange'
import { useConversationStore } from '@/stores/conversation'

const conversationStore = useConversationStore()

const rangeOptions = INBOX_ALL_RANGE_OPTIONS
const presetDays = computed({
  get: () => conversationStore.allInboxDateRange.presetDays,
  set: (v) => { conversationStore.allInboxDateRange.presetDays = v }
})
const isCustom = computed({
  get: () => conversationStore.allInboxDateRange.isCustom,
  set: (v) => { conversationStore.allInboxDateRange.isCustom = v }
})

const customRange = computed({
  get: () => {
    const { customStart, customEnd } = conversationStore.allInboxDateRange
    if (!customStart && !customEnd) return ''
    return customStart && customEnd ? `${customStart},${customEnd}` : ''
  },
  set: (v) => {
    const [start = '', end = ''] = (v || '').split(',')
    conversationStore.allInboxDateRange.customStart = start.trim()
    conversationStore.allInboxDateRange.customEnd = end.trim()
  }
})

const seedCustomFromPreset = () => {
  const end = new Date()
  const start = subDays(end, presetDays.value)
  conversationStore.allInboxDateRange.customEnd = format(end, 'yyyy-MM-dd')
  conversationStore.allInboxDateRange.customStart = format(start, 'yyyy-MM-dd')
}

const selectPreset = (option) => {
  isCustom.value = false
  if (option === presetDays.value) return
  presetDays.value = option
  conversationStore.applyAllInboxDateRange()
}

const enableCustom = () => {
  isCustom.value = true
  if (!conversationStore.allInboxDateRange.customStart || !conversationStore.allInboxDateRange.customEnd) {
    seedCustomFromPreset()
  }
}

watchDebounced(
  () => [
    conversationStore.allInboxDateRange.customStart,
    conversationStore.allInboxDateRange.customEnd,
    conversationStore.allInboxDateRange.isCustom
  ],
  () => {
    if (!isCustom.value) return
    const { customStart, customEnd } = conversationStore.allInboxDateRange
    if (customStart && customEnd && customStart <= customEnd) {
      conversationStore.applyAllInboxDateRange()
    }
  },
  { debounce: 350 }
)

watch(
  () => [conversationStore.allInboxDateRange.customStart, conversationStore.allInboxDateRange.customEnd],
  ([s, e]) => {
    if (s && e && s > e) {
      conversationStore.allInboxDateRange.customEnd = s
    }
  }
)
</script>