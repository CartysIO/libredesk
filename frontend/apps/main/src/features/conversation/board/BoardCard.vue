<template>
  <div
    role="button"
    tabindex="0"
    draggable="true"
    :data-uuid="conversation.uuid"
    class="board-card rounded-md border border-border bg-card text-card-foreground px-3 py-2.5 cursor-pointer select-none transition-colors duration-150 hover:border-input hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
    :class="{
      'opacity-50': isDragging,
      'board-card--active border-input bg-accent/40': active
    }"
    :aria-current="active ? 'true' : undefined"
    @click="$emit('open', conversation)"
    @keydown.enter.self="$emit('open', conversation)"
    @dragstart="onDragStart"
    @dragend="$emit('dragend')"
  >
    <!-- Line 1: whose-move swatch, reference, neutral unread dot (DESIGN.md board card). -->
    <div class="flex items-center gap-2 text-xs tabular-nums">
      <StatusSwatch :status="conversation.status" />
      <span class="font-display shrink-0 font-bold tabular-nums tracking-tight"
        >#{{ conversation.reference_number }}</span
      >
      <span class="min-w-0 flex-1" />
      <span
        v-if="isUnread"
        class="h-2 w-2 shrink-0 rounded-full bg-foreground"
        :aria-label="t('kanban.unread')"
      />
    </div>

    <p class="mt-1 text-sm font-semibold line-clamp-2 break-words text-foreground">
      {{ headline }}
    </p>
    <p
      v-if="preview && preview !== headline"
      class="mt-0.5 text-sm line-clamp-2 break-words text-muted-foreground"
      :class="{ 'font-medium text-foreground': isUnread }"
    >
      {{ preview }}
    </p>

    <div class="mt-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
      <Avatar v-if="assignee" class="h-5 w-5">
        <AvatarImage :src="assignee.avatar_url || ''" class="object-cover" />
        <AvatarFallback class="text-[9px]">{{ assigneeInitials }}</AvatarFallback>
      </Avatar>
      <span
        v-else
        class="h-5 w-5 shrink-0 rounded-full border border-dashed border-muted-foreground/60"
        :title="t('globals.terms.unassigned')"
      />
      <span v-if="metaLabel" class="truncate">{{ metaLabel }}</span>
      <span v-if="relativeTime" class="shrink-0 tabular-nums">· {{ relativeTime }}</span>
      <span class="flex-1" />
      <component
        :is="channelIcon"
        class="h-3.5 w-3.5 shrink-0"
        role="img"
        :aria-label="conversation.inbox_name"
      />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Mail } from 'lucide-vue-next'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import StatusSwatch from '@main/components/brand/StatusSwatch.vue'
import { CHANNEL_ICONS } from '@main/constants/channelIcons.js'
import { useUsersStore } from '@main/stores/users'

const props = defineProps({
  conversation: { type: Object, required: true },
  isDragging: { type: Boolean, default: false },
  active: { type: Boolean, default: false }
})
const emit = defineEmits(['open', 'dragstart', 'dragend'])

const { t } = useI18n()
const usersStore = useUsersStore()

const contactName = computed(() => {
  const c = props.conversation.contact || {}
  return `${c.first_name || ''} ${c.last_name || ''}`.trim()
})
// last_interaction skips activity rows ("X marked the conversation as …"), which every auto-move adds.
const summary = computed(
  () =>
    props.conversation.last_interaction ||
    props.conversation.last_message ||
    ''
)
const headline = computed(
  () => props.conversation.subject || summary.value || contactName.value || '—'
)
const preview = computed(() => {
  if (props.conversation.subject) return summary.value
  return ''
})
const metaLabel = computed(() => {
  if (props.conversation.organization_name) return props.conversation.organization_name
  if (props.conversation.subject && contactName.value) return contactName.value
  return ''
})
const isUnread = computed(() => props.conversation.unread_message_count > 0)
const channelIcon = computed(() => CHANNEL_ICONS[props.conversation.inbox_channel] || Mail)

const assignee = computed(() => {
  const id = props.conversation.assigned_user_id
  if (!id) return null
  return usersStore.users.find((u) => u.id === id) || { id, first_name: '', last_name: '' }
})
const assigneeInitials = computed(() => {
  const a = assignee.value
  return a ? `${a.first_name?.[0] || ''}${a.last_name?.[0] || ''}`.toUpperCase() || '?' : ''
})

const relativeTime = computed(() => {
  const at = props.conversation.last_interaction_at || props.conversation.last_message_at
  return at ? getRelativeTime(at) : ''
})

const onDragStart = (event) => {
  event.dataTransfer.effectAllowed = 'move'
  event.dataTransfer.setData('text/plain', props.conversation.uuid)
  emit('dragstart', props.conversation)
}
</script>
