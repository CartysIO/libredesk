<template>
  <button
    type="button"
    class="group/fold flex w-full items-center gap-3 border-b border-border/70 px-4 py-2 text-left transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:bg-accent"
    :class="{ 'bg-private/60': message.private }"
    :aria-expanded="false"
    @click="$emit('expand')"
  >
    <Avatar class="h-6 w-6 shrink-0 text-[10px]">
      <AvatarImage :src="message.author?.avatar_url || ''" />
      <AvatarFallback class="font-semibold">{{ initials }}</AvatarFallback>
    </Avatar>
    <span class="w-24 shrink-0 truncate text-sm font-semibold sm:w-36">{{ name }}</span>
    <Lock v-if="message.private" class="h-3 w-3 shrink-0 text-muted-foreground" :aria-label="t('globals.terms.privateNote')" />
    <Paperclip
      v-if="message.attachments?.length"
      class="h-3 w-3 shrink-0 text-muted-foreground"
      :aria-label="t('globals.terms.attachment', message.attachments.length)"
    />
    <span class="min-w-0 flex-1 truncate text-sm text-muted-foreground">{{ snippet }}</span>
    <span class="shrink-0 text-xs tabular-nums text-muted-foreground">{{ time }}</span>
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Lock, Paperclip } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { formatMessageTimestamp } from '@shared-ui/utils/datetime.js'
import { messageSnippet } from '@main/utils/threadFold'

const props = defineProps({
  message: { type: Object, required: true }
})
defineEmits(['expand'])

const { t } = useI18n()

const name = computed(() => {
  const author = props.message.author ?? {}
  return `${author.first_name ?? ''} ${author.last_name ?? ''}`.trim() || author.email || ''
})
const initials = computed(() => (props.message.author?.first_name || '?').substring(0, 2).toUpperCase())
const snippet = computed(() => messageSnippet(props.message))
const time = computed(() => formatMessageTimestamp(props.message.created_at))
</script>
