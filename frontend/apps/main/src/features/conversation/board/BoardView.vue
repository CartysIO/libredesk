<template>
  <div ref="root" class="relative h-full flex flex-col overflow-hidden">
    <div class="flex items-center gap-4 px-2 h-12 border-b shrink-0">
      <SidebarTrigger class="cursor-pointer" />
      <span class="text-xl font-semibold truncate">{{ title }}</span>
      <div class="ml-auto">
        <ConversationViewToggle />
      </div>
    </div>

    <div
      ref="scroller"
      class="flex-1 min-h-0 overflow-x-auto overflow-y-hidden p-3"
      :style="openUUID && !expanded ? { paddingRight: `${occupied + 12}px` } : undefined"
    >
      <div class="flex gap-3 h-full min-w-max">
        <BoardColumn
          v-for="column in CONVERSATION_BOARD_COLUMNS"
          :key="column.key"
          :column="column"
          :state="boardStore.columns[column.status]"
          :dragging="boardStore.dragging"
          :active-uuid="openUUID"
          @open="openCard"
          @dragstart="boardStore.dragging = $event"
          @dragend="boardStore.dragging = null"
          @drop="onDrop"
          @load-more="boardStore.fetchColumn(column.status)"
        />
      </div>
    </div>

    <!-- Non-modal: the board stays clickable beside it. -->
    <Transition
      enter-active-class="transition-transform duration-200 ease-out motion-reduce:transition-none"
      enter-from-class="translate-x-full"
      leave-active-class="transition-transform duration-150 ease-in motion-reduce:transition-none"
      leave-to-class="translate-x-full"
    >
      <BoardDrawer
        v-if="openUUID"
        :key="'drawer'"
        :uuid="openUUID"
        :container="root"
        @close="closeDrawer"
        @navigate="goToConversation"
      />
    </Transition>

    <SnoozeDatePicker
      external
      :open="snoozeOpen"
      @update:open="onSnoozeOpenChange"
      @snoozed="onSnoozed"
    />
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useIntervalFn, useDocumentVisibility } from '@vueuse/core'
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useConversationBoardStore } from '@main/stores/conversationBoard'
import { useUsersStore } from '@main/stores/users'
import {
  CONVERSATION_BOARD_COLUMNS,
  CONVERSATION_DEFAULT_STATUSES,
  CONVERSATION_LIST_TYPE
} from '@main/constants/conversation'
import { useBoardDrawer } from '@main/composables/useBoardDrawer'
import { findBoardCard, isConversationRoute } from '@main/utils/conversationBoard'
import SnoozeDatePicker from '@main/features/command/SnoozeDatePicker.vue'
import BoardColumn from './BoardColumn.vue'
import BoardDrawer from './BoardDrawer.vue'
import ConversationViewToggle from './ConversationViewToggle.vue'

const REFRESH_INTERVAL_MS = 120000

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const root = ref(null)
const scroller = ref(null)
const { expanded, occupied } = useBoardDrawer()
const boardStore = useConversationBoardStore()
const usersStore = useUsersStore()

const title = computed(() => {
  const typeKey = route.meta?.typeKey?.(route)
  if (typeKey) return t(typeKey)
  const key = route.meta?.titleKey
  return key ? t(key, route.meta?.titleCount || 1) : ''
})

// Same list context the list view derives from the route.
const listContext = computed(() => {
  if (route.params.viewID) {
    return { listType: CONVERSATION_LIST_TYPE.VIEW, viewID: route.params.viewID }
  }
  if (route.params.teamID) {
    return { listType: CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED, teamID: route.params.teamID }
  }
  return { listType: route.params.type }
})

const activate = () => {
  if (listContext.value.listType) boardStore.activate(listContext.value)
}

onMounted(activate)
onUnmounted(() => boardStore.deactivate())
// The drawer follows the URL: a conversation route opens it, a list route closes it.
const openUUID = computed(() =>
  isConversationRoute(route.name) ? String(route.params.uuid || '') : ''
)
watch(openUUID, (uuid) => (boardStore.openUUID = uuid), { immediate: true })

// Keep the open card clear of the drawer.
const revealOpenCard = async () => {
  await nextTick()
  const box = scroller.value
  if (!box || !openUUID.value || expanded.value) return
  const card = box.querySelector(`[data-uuid="${CSS.escape(openUUID.value)}"]`)
  if (!card) return
  const b = box.getBoundingClientRect()
  const c = card.getBoundingClientRect()
  const visibleRight = b.right - occupied.value
  if (c.right > visibleRight - 12) box.scrollLeft += c.right - visibleRight + 12
  else if (c.left < b.left + 12) box.scrollLeft -= b.left + 12 - c.left
}
// The card may arrive after the drawer opens (deep link) or move column.
const openCardStatus = computed(() => findBoardCard(boardStore.columns, openUUID.value)?.status)
watch([openUUID, occupied, openCardStatus], revealOpenCard)

watch(() => [route.params.type, route.params.teamID, route.params.viewID], activate)

// Agent avatars on the cards.
watch(
  () =>
    CONVERSATION_BOARD_COLUMNS.flatMap((c) =>
      boardStore.columns[c.status].items.map((i) => i.assigned_user_id)
    )
      .filter(Boolean)
      .join(','),
  (ids) => ids && usersStore.ensureUserIDs(ids.split(',')),
  { immediate: true }
)

const visibility = useDocumentVisibility()
const { pause, resume } = useIntervalFn(() => boardStore.refreshAll(), REFRESH_INTERVAL_MS)
watch(visibility, (v) => {
  if (v === 'visible') {
    boardStore.refreshAll()
    resume()
  } else {
    pause()
  }
})

const routeParams = () => ({
  ...(route.params.teamID && { teamID: route.params.teamID }),
  ...(route.params.viewID && { viewID: route.params.viewID }),
  ...(route.params.type && { type: route.params.type })
})

const conversationRouteName = () =>
  route.params.teamID
    ? 'team-inbox-conversation'
    : route.params.viewID
      ? 'view-inbox-conversation'
      : 'inbox-conversation'

// Opening a card from the board pushes. Swapping the drawer replaces, so Back closes it.
const goToConversation = (conversation) => {
  const target = {
    name: conversationRouteName(),
    params: { ...routeParams(), uuid: conversation.uuid }
  }
  if (openUUID.value) router.replace(target)
  else router.push(target)
}
const openCard = goToConversation

const closeDrawer = () => {
  const listName = String(route.name).replace(/-conversation$/, '')
  const target = router.resolve({ name: listName, params: routeParams() })
  if (window.history.state?.back?.split('?')[0] === target.path) router.back()
  else router.push(target)
}

const snoozeOpen = ref(false)
const pendingSnoozeUUID = ref('')

const onDrop = (column) => {
  const card = boardStore.dragging
  boardStore.dragging = null
  if (!card || card.status === column.status) return
  if (column.status === CONVERSATION_DEFAULT_STATUSES.SNOOZED) {
    // Ask for the duration first. The card moves once one is picked.
    pendingSnoozeUUID.value = card.uuid
    snoozeOpen.value = true
    return
  }
  boardStore.moveCard(card.uuid, column.status)
}

const onSnoozeOpenChange = (open) => {
  snoozeOpen.value = open
}

const onSnoozed = (duration) => {
  const uuid = pendingSnoozeUUID.value
  pendingSnoozeUUID.value = ''
  if (uuid) boardStore.moveCard(uuid, CONVERSATION_DEFAULT_STATUSES.SNOOZED, duration)
}
</script>
