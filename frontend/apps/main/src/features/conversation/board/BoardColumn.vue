<template>
  <section
    class="flex flex-col h-full w-72 shrink-0 border-r border-border/70 transition-colors"
    :class="{ 'bg-accent/50': isOver }"
    :aria-label="t(column.labelKey)"
    @dragover.prevent="onDragOver"
    @dragleave="onDragLeave"
    @drop.prevent="onDrop"
  >
    <header class="flex items-center gap-2 px-3 py-2 shrink-0">
      <span class="inline-flex items-center gap-2 text-sm font-semibold text-foreground">
        <StatusSwatch :status="column.status" />
        {{ t(column.labelKey) }}
      </span>
      <span class="text-sm text-muted-foreground tabular-nums">{{ state.total }}</span>
    </header>

    <!-- overscroll-y only: a sideways swipe on a card must chain to the board so columns scroll on phones. -->
    <div ref="scroller" class="flex-1 overflow-y-auto overscroll-y-contain px-2 pb-2 space-y-2" @scroll.passive="onScroll">
      <template v-if="state.loading">
        <Skeleton v-for="i in 4" :key="i" class="h-24 w-full" />
      </template>
      <template v-else>
        <BoardCardMenu
          v-for="conversation in state.items"
          :key="conversation.uuid"
          :conversation="conversation"
          @action="$emit('card-action', { action: $event, conversation })"
        >
          <BoardCard
            :conversation="conversation"
            :is-dragging="dragging?.uuid === conversation.uuid"
            :active="activeUuid === conversation.uuid"
            @open="$emit('open', $event)"
            @dragstart="$emit('dragstart', $event)"
            @dragend="$emit('dragend')"
          />
        </BoardCardMenu>
        <p v-if="state.error" class="text-xs text-destructive px-1">{{ state.error }}</p>
        <p
          v-else-if="!state.items.length && !state.fetching"
          class="text-xs text-muted-foreground text-center py-6"
        >
          {{ t('kanban.emptyColumn') }}
        </p>
        <div v-if="state.fetching" class="flex justify-center py-2">
          <Loader2 class="w-4 h-4 animate-spin text-muted-foreground" />
        </div>
      </template>
    </div>
  </section>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loader2 } from 'lucide-vue-next'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import BoardCard from './BoardCard.vue'
import BoardCardMenu from './BoardCardMenu.vue'
import StatusSwatch from '@main/components/brand/StatusSwatch.vue'

const NEAR_BOTTOM_PX = 240

const props = defineProps({
  column: { type: Object, required: true },
  state: { type: Object, required: true },
  dragging: { type: Object, default: null },
  activeUuid: { type: String, default: '' }
})
const emit = defineEmits(['open', 'dragstart', 'dragend', 'drop', 'load-more', 'card-action'])

const { t } = useI18n()
const scroller = ref(null)
const isOver = ref(false)

const onScroll = () => {
  const el = scroller.value
  if (!el || props.state.fetching || !props.state.hasMore) return
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - NEAR_BOTTOM_PX) emit('load-more')
}

const onDragOver = (event) => {
  if (!props.dragging) return
  event.dataTransfer.dropEffect = 'move'
  isOver.value = true
}
const onDragLeave = (event) => {
  if (!event.currentTarget.contains(event.relatedTarget)) isOver.value = false
}
const onDrop = () => {
  isOver.value = false
  if (props.dragging) emit('drop', props.column)
}
</script>
