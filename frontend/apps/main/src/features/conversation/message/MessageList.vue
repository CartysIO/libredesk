<template>
  <div class="flex flex-col relative h-full">
    <div ref="threadEl" class="flex-1 overflow-y-auto overscroll-contain [overflow-anchor:none]" @scroll="handleScroll">
      <div ref="contentEl" class="min-h-full relative" :class="isEmailThread ? 'pb-6' : 'px-4 pb-10'">
        <MessagesSkeleton :count="10" v-if="conversationStore.messages.loading" />

        <!-- The key also changes when the first messages land: the conversation often loads before
             its messages, and the group must not animate them in as if they had just arrived. -->
        <TransitionGroup
          v-else
          :key="`${conversationStore.current?.uuid}:${messageRows.length > 0}`"
          :enter-active-class="loadingOlder ? '' : 'animate-slide-in'"
          tag="div"
        >
          <div
            v-for="item in displayRows"
            :key="displayRowKey(item)"
            :data-message-uuid="item.row?.message.uuid"
            :class="displayRowClass(item)"
          >
          <!-- Email threads (Clyr desk): "N earlier messages" stands for the middle of a long folded run. -->
          <button
            v-if="item.kind === 'summary'"
            type="button"
            class="flex w-full items-center gap-3 border-b border-border/70 px-4 py-1.5 text-left text-xs font-semibold text-muted-foreground transition-colors hover:bg-accent/50 hover:text-foreground"
            @click="expandRun(item.key)"
          >
            <span class="flex h-6 w-6 items-center justify-center rounded-full border border-border tabular-nums">{{ item.count }}</span>
            {{ t('conversation.earlierMessages', { count: item.count }, item.count) }}
          </button>
          <MessageFoldRow
            v-else-if="item.kind === 'fold'"
            :message="item.row.message"
            @expand="setOpen(item.row.message.uuid, true)"
          />
          <template v-else>
            <DaySeparator
              v-if="item.row.showDaySeparator && !isEmailThread"
              :date="item.row.message.created_at"
              class="mb-4"
            />
            <div v-if="!item.row.message.private && item.row.message.type !== 'activity'">
              <MessageBubble
                :message="item.row.message"
                :direction="item.row.message.type"
                :group-with-prev="item.row.groupWithPrev"
                :group-with-next="item.row.groupWithNext"
                :thread="isEmailThread"
                @collapse="setOpen(item.row.message.uuid, false)"
              />
            </div>
            <div v-else-if="item.row.message.type === 'outgoing' && item.row.message.private">
              <MessageBubble
                :message="item.row.message"
                direction="outgoing"
                :group-with-prev="item.row.groupWithPrev"
                :group-with-next="item.row.groupWithNext"
                :thread="isEmailThread"
                @collapse="setOpen(item.row.message.uuid, false)"
              />
            </div>
            <div v-else-if="item.row.message.type === 'activity'">
              <ActivityMessageBubble :message="item.row.message" />
            </div>
          </template>
          </div>
        </TransitionGroup>
        <div ref="readMarkerEl" class="h-px" />
      </div>

      <!-- Typing indicator -->
      <div v-if="conversationStore.conversation.isTyping" class="px-4 pb-4">
        <TypingIndicator />
      </div>
    </div>

    <div
      v-if="olderSpinner"
      class="pointer-events-none absolute left-1/2 top-2 z-10 -translate-x-1/2 rounded-full bg-background p-2 text-muted-foreground shadow-sm"
      role="status"
    >
      <Loader2 class="size-4 animate-spin" aria-hidden="true" />
      <span class="sr-only">{{ $t('globals.terms.loading') }}</span>
    </div>

    <!-- Sticky container for the scroll arrow -->
    <ScrollToBottomButton
      :is-at-bottom="!hasUserScrolled"
      :unread-count="unReadMessages"
      @scroll-to-bottom="handleScrollToBottom"
    />

    <!-- Nudge to self-assign after replying to an unassigned conversation -->
    <AssignSelfNudge
      :show="showAssignNudge"
      @assign="assignToSelf"
      @dismiss="showAssignNudge = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useElementSize } from '@vueuse/core'
import { useConversationRead } from '@main/composables/useConversationRead'
import MessageBubble from './MessageBubble.vue'
import MessageFoldRow from './MessageFoldRow.vue'
import { defaultOpenUUIDs, foldThreadRows } from '@main/utils/threadFold'
import { useI18n } from 'vue-i18n'
import ActivityMessageBubble from './ActivityMessageBubble.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { Loader2 } from 'lucide-vue-next'
import ScrollToBottomButton from '@shared-ui/components/ScrollToBottomButton'
import DaySeparator from '@shared-ui/components/DaySeparator'
import { isSameDay } from 'date-fns'
import AssignSelfNudge from './AssignSelfNudge.vue'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { useBulkActionPermissions } from '@main/composables/useBulkActionPermissions'
import MessagesSkeleton from './MessagesSkeleton.vue'
import { TypingIndicator } from '@shared-ui/components/TypingIndicator'
import { useStickyScroll } from '@shared-ui/composables'
import { delayedLoading } from '@main/utils/delayed-loading'

const MENTION_TOP_OFFSET_RATIO = 0.25
const MENTION_SETTLE_FRAMES = 3
const MENTION_MAX_ANCHOR_FRAMES = 90
const HIGHLIGHT_MS = 2500
const OLDER_MESSAGES_SCROLL_DISTANCE = 48

const route = useRoute()
const { t } = useI18n()

const conversationStore = useConversationStore()
const userStore = useUserStore()
const threadEl = ref(null)
const { height: threadHeight } = useElementSize(threadEl)
const contentEl = ref(null)
const readMarkerEl = ref(null)
const readReady = ref(false)
const emitter = useEmitter()
const unReadMessages = ref(0)
const showAssignNudge = ref(false)
const loadingOlder = ref(false)
const olderSpinner = ref(false)

watch(
  () => loadingOlder.value && conversationStore.messages.fetching,
  (loading, _, onCleanup) => {
    if (!loading) {
      olderSpinner.value = false
      return
    }
    const guard = delayedLoading(olderSpinner, 'value', 100 /** delayMs **/, 0 /** minVisibleMs **/)
    onCleanup(guard.cancel)
  }
)
const { canAssignAgent } = useBulkActionPermissions()
let currentConversationUUID = ''
let openScrollDone = false

useConversationRead({
  root: threadEl,
  marker: readMarkerEl,
  conversationUUID: computed(() => conversationStore.current?.uuid),
  message: computed(() => conversationStore.conversationMessages.findLast(message => message.id > 0)),
  loading: computed(() => !readReady.value || conversationStore.conversation.loading || conversationStore.messages.loading || conversationStore.messages.fetching || conversationStore.messages.fetchingLatest || conversationStore.hasPendingMessages),
  acknowledge: (uuid, messageUUID) => conversationStore.updateAssigneeLastSeen(uuid, messageUUID)
})

const assignToSelf = () => {
  conversationStore.updateAssignee('user', { assignee_id: userStore.userID })
}

const { hasUserScrolled, scrollToBottom, scrollToOffset, handleScroll: handleStickyScroll } =
  useStickyScroll(threadEl, contentEl, {
    skipAutoScroll: () => loadingOlder.value,
    onArriveBottom: () => { unReadMessages.value = 0 }
  })

const handleScroll = () => {
  handleStickyScroll()
  if (
    openScrollDone &&
    readReady.value &&
    threadEl.value?.scrollTop <= OLDER_MESSAGES_SCROLL_DISTANCE
  ) {
    loadMore()
  }
}

const handleScrollToBottom = () => {
  hasUserScrolled.value = false
  scrollToBottom()
}

const applyOpenScroll = () => {
  const thread = threadEl.value
  if (!thread) return
  const targetUUID = route.query.scrollTo
  const targetEl = targetUUID ? thread.querySelector(`[data-message-uuid="${targetUUID}"]`) : null
  if (targetEl) {
    hasUserScrolled.value = true
    // Messages above the target collapse to max-h after mount, so re-pin until offsetTop stops moving.
    const anchorConversation = currentConversationUUID
    let lastOffset = -1
    let stableFrames = 0
    let frames = 0
    const anchorToTarget = () => {
      if (!threadEl.value || !targetEl.isConnected || anchorConversation !== currentConversationUUID) return
      const offset = targetEl.offsetTop
      scrollToOffset(Math.max(0, offset - threadEl.value.clientHeight * MENTION_TOP_OFFSET_RATIO))
      stableFrames = offset === lastOffset ? stableFrames + 1 : 0
      lastOffset = offset
      if (stableFrames < MENTION_SETTLE_FRAMES && ++frames < MENTION_MAX_ANCHOR_FRAMES) requestAnimationFrame(anchorToTarget)
      else readReady.value = true
    }
    anchorToTarget()
    targetEl.classList.add('highlight-mention')
    setTimeout(() => targetEl.classList.remove('highlight-mention'), HIGHLIGHT_MS)
  } else {
    hasUserScrolled.value = false
    scrollToBottom()
    readReady.value = true
  }
}

const newMessageHandler = (data) => {
  if (data.conversation_uuid !== conversationStore.current.uuid) return
  const message = data.message
  if (message?.sender_id === userStore.userID) {
    hasUserScrolled.value = false
    if (
      message.type === 'outgoing' &&
      !message.private &&
      !conversationStore.current.assigned_user_id &&
      canAssignAgent.value
    ) {
      showAssignNudge.value = true
    }
    return
  }
  if (hasUserScrolled.value) unReadMessages.value++
}

onMounted(() => {
  emitter.on(EMITTER_EVENTS.NEW_MESSAGE, newMessageHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.NEW_MESSAGE, newMessageHandler)
})

watch(
  () => conversationStore.current?.uuid,
  (newUUID) => {
    if (!newUUID || newUUID === currentConversationUUID) return
    currentConversationUUID = newUUID
    unReadMessages.value = 0
    openScrollDone = false
    readReady.value = false
    showAssignNudge.value = false
  }
)

watch(
  () => conversationStore.current?.assigned_user_id,
  (assignedUserId) => {
    if (assignedUserId) showAssignNudge.value = false
  }
)

watch(
  () => [conversationStore.conversationMessages.length, conversationStore.messages.loading],
  ([len, loading]) => {
    if (openScrollDone || loading || !len) return
    openScrollDone = true
    hasUserScrolled.value = !!route.query.scrollTo
    nextTick(applyOpenScroll)
  },
  { immediate: true }
)

// Watch for typing indicator and auto-scroll if user is at bottom
watch(
  () => conversationStore.conversation.isTyping,
  (isTyping) => {
    if (isTyping && !hasUserScrolled.value) nextTick(scrollToBottom)
  }
)

const GROUP_WINDOW_MS = 60_000

const canGroup = (a, b) => {
  if (!a || !b) return false
  if (a.type === 'activity' || b.type === 'activity') return false
  if (a.type !== b.type) return false
  if (Boolean(a.private) !== Boolean(b.private)) return false
  if (a.status === 'failed' || b.status === 'failed') return false

  const aSenderId = a.author?.id ?? a.sender_id
  const bSenderId = b.author?.id ?? b.sender_id
  if (!aSenderId || aSenderId !== bSenderId) return false

  const aBucket = Math.floor(new Date(a.created_at).getTime() / GROUP_WINDOW_MS)
  const bBucket = Math.floor(new Date(b.created_at).getTime() / GROUP_WINDOW_MS)
  return aBucket === bBucket
}

const getSpacingClass = (index, groupWithPrev) => {
  if (index === 0) return 'pt-4'
  return groupWithPrev ? 'mt-1' : 'mt-4'
}

const loadMore = async () => {
  if (
    loadingOlder.value ||
    conversationStore.messages.loading ||
    conversationStore.messages.fetching ||
    !conversationStore.currentConversationHasMoreMessages
  ) return
  const thread = threadEl.value
  if (!thread) return
  const conversationUUID = conversationStore.current?.uuid
  const previousPage = conversationStore.messages.data.getLastFetchedPage(conversationUUID)
  const threadTop = thread.getBoundingClientRect().top
  const anchor = [...thread.querySelectorAll('[data-message-uuid]')].find(
    element => element.getBoundingClientRect().bottom > threadTop
  )
  const anchorTop = anchor?.getBoundingClientRect().top
  const prevHeight = thread.scrollHeight
  const prevTop = thread.scrollTop
  hasUserScrolled.value = true
  loadingOlder.value = true
  try {
    await conversationStore.fetchNextMessages()
    await nextTick()
    if (conversationUUID !== conversationStore.current?.uuid || thread !== threadEl.value) return
    if (anchor?.isConnected) {
      scrollToOffset(thread.scrollTop + anchor.getBoundingClientRect().top - anchorTop)
    } else {
      scrollToOffset(thread.scrollHeight - prevHeight + prevTop)
    }
    if (thread.scrollHeight <= thread.clientHeight) hasUserScrolled.value = false
  } finally {
    loadingOlder.value = false
    if (
      conversationUUID !== conversationStore.current?.uuid ||
      conversationStore.messages.data.getLastFetchedPage(conversationUUID) > previousPage
    ) fillShortThread()
  }
}

const fillShortThread = () => {
  const thread = threadEl.value
  if (
    readReady.value &&
    thread?.clientHeight > 0 &&
    thread.scrollHeight <= thread.clientHeight
  ) loadMore()
}

watch([threadHeight, readReady], fillShortThread, { flush: 'post' })

const messageRows = computed(() => {
  const messages = conversationStore.conversationMessages
  return messages.map((message, index) => {
    const groupWithPrev = canGroup(messages[index - 1], message)
    const groupWithNext = canGroup(message, messages[index + 1])
    return {
      message,
      groupWithPrev,
      groupWithNext,
      spacingClass: getSpacingClass(index, groupWithPrev),
      showDaySeparator:
        index === 0 ||
        !isSameDay(new Date(messages[index - 1].created_at), new Date(message.created_at))
    }
  })
})

// Clyr desk: email threads fold like Gmail. Live chat keeps its bubbles.
const isEmailThread = computed(() => conversationStore.current?.inbox_channel === 'email')
const openOverrides = ref(new Map())
const openRuns = ref(new Set())
const defaultOpen = computed(() =>
  defaultOpenUUIDs(conversationStore.conversationMessages, route.query.scrollTo || '')
)
const isMessageOpen = (message) =>
  openOverrides.value.has(message.uuid) ? openOverrides.value.get(message.uuid) : defaultOpen.value.has(message.uuid)

const setOpen = (uuid, open) => {
  const next = new Map(openOverrides.value)
  next.set(uuid, open)
  openOverrides.value = next
}
const expandRun = (key) => {
  openRuns.value = new Set(openRuns.value).add(key)
}

const displayRows = computed(() =>
  isEmailThread.value
    ? foldThreadRows(messageRows.value, isMessageOpen, openRuns.value)
    : messageRows.value.map((row) => ({ kind: 'message', row }))
)
const displayRowClass = (item) => {
  if (item.kind !== 'message') return null
  const activity = { 'my-2': item.row.message.type === 'activity' }
  return isEmailThread.value ? activity : [item.row.spacingClass, activity]
}
const displayRowKey = (item) => {
  if (item.kind === 'summary') return `run-${item.key}`
  const key = item.row.message.render_key || item.row.message.uuid
  return item.kind === 'fold' ? `fold-${key}` : key
}

watch(
  () => conversationStore.current?.uuid,
  () => {
    openOverrides.value = new Map()
    openRuns.value = new Set()
  }
)
</script>

<style scoped>
/* Highlight via an opacity-faded overlay, not the element's own background, to avoid a text repaint flicker when it ends. */
.highlight-mention {
  position: relative;
}

.highlight-mention::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 0.5rem;
  background-color: rgb(251 191 36 / 0.35);
  pointer-events: none;
  animation: highlightFade 2.5s ease-out forwards;
}

:global(.dark .highlight-mention::after) {
  background-color: rgb(250 204 21 / 0.2);
}

@keyframes highlightFade {
  from {
    opacity: 1;
  }
  to {
    opacity: 0;
  }
}
</style>
