<template>
  <div class="desk-inbox-date-range flex flex-wrap items-center gap-1">
    <span class="text-muted-foreground mr-1 text-xs">{{ dateFieldLabel }}</span>
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
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import DateFilterValue from '@/components/filter/DateFilterValue.vue'
import { useInboxAllDateRange } from '@/composables/useInboxAllDateRange'
import { useConversationStore } from '@/stores/conversation'
import { CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'

const { t } = useI18n()
const conversationStore = useConversationStore()
const dateFieldLabel = computed(() =>
  conversationStore.getListStatus === CONVERSATION_DEFAULT_STATUSES.RESOLVED
    ? t('globals.terms.resolvedAt')
    : t('globals.terms.createdAt')
)

const {
  rangeOptions,
  presetDays,
  isCustom,
  customRange,
  selectPreset,
  enableCustom
} = useInboxAllDateRange()
</script>