<template>
  <aside
    ref="panel"
    class="absolute top-12 bottom-0 right-0 z-20 flex flex-col border-l bg-background shadow-2xl max-w-full"
    :style="panelStyle"
    :aria-label="t('kanban.conversationDrawer')"
  >
    <!-- Drag the left edge to resize. -->
    <div
      v-if="!expanded && !isMobile"
      class="absolute inset-y-0 -left-1 w-2 cursor-col-resize z-10 hover:bg-primary/30 active:bg-primary/40"
      role="separator"
      aria-orientation="vertical"
      @pointerdown.prevent="startResize"
    />

    <div class="h-10 shrink-0 px-2 border-b flex items-center gap-1">
      <Button
        variant="ghost"
        size="icon"
        class="h-8 w-8"
        :aria-label="t('kanban.closeDrawer')"
        @click="$emit('close')"
      >
        <X class="w-4 h-4" />
      </Button>
      <Button
        v-if="!isMobile"
        variant="ghost"
        size="icon"
        class="h-8 w-8"
        :aria-label="expanded ? t('kanban.collapseDrawer') : t('kanban.expandDrawer')"
        @click="toggleExpanded"
      >
        <Minimize2 v-if="expanded" class="w-4 h-4" />
        <Maximize2 v-else class="w-4 h-4" />
      </Button>
      <div class="ml-auto flex items-center gap-1">
        <Button
          variant="ghost"
          size="icon"
          class="h-8 w-8"
          :disabled="!previous"
          :aria-label="t('kanban.previousCard')"
          @click="previous && $emit('navigate', previous)"
        >
          <ChevronUp class="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="h-8 w-8"
          :disabled="!next"
          :aria-label="t('kanban.nextCard')"
          @click="next && $emit('navigate', next)"
        >
          <ChevronDown class="w-4 h-4" />
        </Button>
      </div>
    </div>

    <div class="flex-1 min-h-0">
      <ConversationDetailView :uuid="uuid" />
    </div>
  </aside>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { X, Maximize2, Minimize2, ChevronUp, ChevronDown } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useIsMobile } from '@shared-ui/composables'
import ConversationDetailView from '@main/views/conversation/ConversationDetailView.vue'
import { useConversationBoardStore } from '@main/stores/conversationBoard'
import { useBoardDrawer } from '@main/composables/useBoardDrawer'
import {
  adjacentBoardCard,
  clampDrawerWidth,
  isTypingTarget
} from '@main/utils/conversationBoard'

const props = defineProps({
  uuid: { type: String, required: true },
  // The board element. The drawer sizes itself against it.
  container: { type: Object, default: null }
})
const emit = defineEmits(['close', 'navigate'])

const { t } = useI18n()
const isMobile = useIsMobile()
const boardStore = useConversationBoardStore()
const panel = ref(null)
const { expanded, width, occupied, toggleExpanded, resetExpanded, setWidth } = useBoardDrawer()

const previous = computed(() => adjacentBoardCard(boardStore.columns, props.uuid, -1))
const next = computed(() => adjacentBoardCard(boardStore.columns, props.uuid, 1))

const panelStyle = computed(() => {
  if (expanded.value || isMobile.value) return { width: '100%' }
  if (width.value > 0) return { width: `${width.value}px` }
  return { width: '75%', minWidth: 'min(720px, 100%)' }
})

let dragging = false
const onMove = (event) => {
  if (!dragging || !props.container) return
  const rect = props.container.getBoundingClientRect()
  width.value = clampDrawerWidth(rect.right - event.clientX, rect.width)
}
const stopResize = () => {
  if (!dragging) return
  dragging = false
  window.removeEventListener('pointermove', onMove)
  window.removeEventListener('pointerup', stopResize)
  setWidth(width.value)
}
const startResize = () => {
  dragging = true
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', stopResize)
}

// A menu, dialog or popover handles Escape itself.
const overlayOpen = () =>
  !!document.querySelector('[role="dialog"][data-state="open"], [role="menu"][data-state="open"], [role="listbox"][data-state="open"]')

const onKeydown = (event) => {
  if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) return
  if (isTypingTarget(event.target)) return
  if (event.key === 'Escape') {
    if (overlayOpen()) return
    emit('close')
  } else if (event.key === 'f' || event.key === 'F') {
    toggleExpanded()
  }
}

let observer = null
onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  if (typeof ResizeObserver !== 'undefined' && panel.value) {
    observer = new ResizeObserver(() => (occupied.value = panel.value?.offsetWidth || 0))
    observer.observe(panel.value)
  }
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  observer?.disconnect()
  occupied.value = 0
  stopResize()
  resetExpanded()
})
</script>
