<template>
  <div class="h-full flex flex-col">
    <div class="flex items-center gap-4 px-2 h-12 border-b shrink-0">
      <SidebarTrigger class="cursor-pointer" />
      <span class="text-xl font-semibold truncate">{{ title }}</span>
      <div class="ml-auto">
        <ConversationViewToggle />
      </div>
    </div>

    <div class="flex-1 min-h-0 overflow-x-auto overflow-y-hidden p-3">
      <div class="flex gap-3 h-full min-w-max">
        <BoardColumn
          v-for="column in CONVERSATION_BOARD_COLUMNS"
          :key="column.key"
          :column="column"
          :state="boardStore.columns[column.status]"
          :dragging="boardStore.dragging"
          @open="openCard"
          @dragstart="boardStore.dragging = $event"
          @dragend="boardStore.dragging = null"
          @drop="onDrop"
          @load-more="boardStore.fetchColumn(column.status)"
        />
      </div>
    </div>

    <SnoozeDatePicker
      external
      :open="snoozeOpen"
      @update:open="onSnoozeOpenChange"
      @snoozed="onSnoozed"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useIntervalFn, useDocumentVisibility } from '@vueuse/core'
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useConversationStore } from '@main/stores/conversation'
import { useConversationBoardStore } from '@main/stores/conversationBoard'
import { useUsersStore } from '@main/stores/users'
import {
  CONVERSATION_BOARD_COLUMNS,
  CONVERSATION_DEFAULT_STATUSES,
  CONVERSATION_LIST_TYPE
} from '@main/constants/conversation'
import SnoozeDatePicker from '@main/features/command/SnoozeDatePicker.vue'
import BoardColumn from './BoardColumn.vue'
import ConversationViewToggle from './ConversationViewToggle.vue'

const REFRESH_INTERVAL_MS = 120000

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const conversationStore = useConversationStore()
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

const openCard = (conversation) => {
  // The split view shows the list of this card's column.
  if (!route.params.viewID) {
    conversationStore.setListStatus(conversation.status, false)
    conversationStore.resetConversations()
  }
  const name = route.params.teamID
    ? 'team-inbox-conversation'
    : route.params.viewID
      ? 'view-inbox-conversation'
      : 'inbox-conversation'
  router.push({
    name,
    params: {
      uuid: conversation.uuid,
      ...(route.params.teamID && { teamID: route.params.teamID }),
      ...(route.params.viewID && { viewID: route.params.viewID })
    }
  })
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
