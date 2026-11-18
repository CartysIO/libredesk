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
import { Button } from '@shared-ui/components/ui/button'
import DateFilterValue from '@/components/filter/DateFilterValue.vue'
import { useInboxAllDateRange } from '@/composables/useInboxAllDateRange'

const {
  rangeOptions,
  presetDays,
  isCustom,
  customRange,
  selectPreset,
  enableCustom
} = useInboxAllDateRange()
</script>