import { computed, watch } from 'vue'
import { watchDebounced } from '@vueuse/core'
import { subDays, format } from 'date-fns'
import { useConversationStore } from '@/stores/conversation'
import { useConversationBoardStore } from '@/stores/conversationBoard'
import { CONVERSATION_LIST_TYPE } from '@/constants/conversation'
import { DATE_RANGE_PRESET_OPTIONS } from '@/constants/dateRangePresets'

export function useInboxAllDateRange () {
  const conversationStore = useConversationStore()
  const boardStore = useConversationBoardStore()

  const applyRange = () => {
    if (conversationStore.conversations.listType === CONVERSATION_LIST_TYPE.ALL) {
      conversationStore.applyAllInboxDateRange()
    }
    if (
      boardStore.context.active &&
      boardStore.context.listType === CONVERSATION_LIST_TYPE.ALL
    ) {
      boardStore.refreshAll()
    }
  }

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
    applyRange()
  }

  const enableCustom = () => {
    const hadRange =
      conversationStore.allInboxDateRange.customStart &&
      conversationStore.allInboxDateRange.customEnd
    isCustom.value = true
    if (!hadRange) {
      seedCustomFromPreset()
      applyRange()
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
        applyRange()
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

  return {
    rangeOptions: DATE_RANGE_PRESET_OPTIONS,
    presetDays,
    isCustom,
    customRange,
    selectPreset,
    enableCustom
  }
}