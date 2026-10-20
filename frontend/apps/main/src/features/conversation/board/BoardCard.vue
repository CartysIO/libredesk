<template>
  <div
    role="button"
    tabindex="0"
    draggable="true"
    :data-uuid="conversation.uuid"
    class="rounded-md border bg-card text-card-foreground px-3 py-2.5 space-y-1 cursor-pointer select-none hover:border-input hover:bg-accent/40 transition-colors"
    :class="{ 'opacity-50': isDragging, 'ring-2 ring-primary bg-accent/60': active }"
    :aria-current="active ? 'true' : undefined"
    @click="$emit('open', conversation)"
    @keydown.enter.self="$emit('open', conversation)"
    @dragstart="onDragStart"
    @dragend="$emit('dragend')"
  >
    <div class="flex items-baseline justify-between gap-2">
      <h3 class="text-sm font-semibold truncate min-w-0">{{ title }}</h3>
      <span class="text-xs text-muted-foreground tabular-nums shrink-0">
        #{{ conversation.reference_number }}
      </span>
    </div>
    <p v-if="conversation.organization_name" class="text-xs text-muted-foreground truncate">
      {{ contactName }}
    </p>
    <p class="text-sm line-clamp-2 break-words" :class="{ 'font-medium': isUnread }">
      {{ summary }}
    </p>

    <div class="flex items-center gap-1.5 pt-1 text-xs text-muted-foreground">
      <Avatar v-if="assignee" class="w-5 h-5">
        <AvatarImage :src="assignee.avatar_url || ''" class="object-cover" />
        <AvatarFallback class="text-[9px]">{{ assigneeInitials }}</AvatarFallback>
      </Avatar>
      <span
        v-else
        class="w-5 h-5 rounded-full border border-dashed border-muted-foreground/60 shrink-0"
        :title="t('globals.terms.unassigned')"
      />
      <span v-if="relativeTime" class="tabular-nums">· {{ relativeTime }}</span>
      <span class="flex-1" />
      <span
        v-if="isUnread"
        class="w-2 h-2 rounded-full bg-destructive shrink-0"
        :aria-label="t('kanban.unread')"
      />
      <component
        :is="channelIcon"
        class="w-3.5 h-3.5 shrink-0"
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
const title = computed(() => props.conversation.organization_name || contactName.value)
// last_interaction skips activity rows ("X marked the conversation as …"), which every auto-move adds.
const summary = computed(
  () =>
    props.conversation.subject ||
    props.conversation.last_interaction ||
    props.conversation.last_message ||
    ''
)
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
