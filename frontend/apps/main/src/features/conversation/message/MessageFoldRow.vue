<template>
  <button
    type="button"
    class="desk-fold-row group/fold"
    :class="{
      'desk-fold-row--private': message.private,
      'desk-fold-row--outgoing': message.type === 'outgoing' && !message.private
    }"
    :aria-expanded="false"
    @click="$emit('expand')"
  >
    <Avatar class="desk-fold-row__avatar h-[26px] w-[26px] shrink-0 text-[10px]">
      <AvatarImage :src="message.author?.avatar_url || ''" />
      <AvatarFallback class="font-semibold">{{ initials }}</AvatarFallback>
    </Avatar>
    <span class="desk-fold-row__name">{{ name }}</span>
    <span v-if="message.private" class="desk-fold-row__note">
      <Lock class="desk-fold-row__icon" :aria-label="t('globals.terms.privateNote')" />
      <span class="desk-fold-row__note-text" aria-hidden="true">{{ t('globals.terms.privateNote') }}</span>
    </span>
    <Paperclip
      v-if="message.attachments?.length"
      class="desk-fold-row__icon"
      :aria-label="t('globals.terms.attachment', message.attachments.length)"
    />
    <span class="desk-fold-row__snippet">{{ snippet }}</span>
    <time class="desk-fold-row__time" :datetime="message.created_at" :title="fullTime">
      <span class="desk-fold-row__date">{{ time.date }}</span>{{ time.time }}
    </time>
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { isToday } from 'date-fns'
import { Lock, Paperclip } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { formatMessageTimestamp, formatFullTimestamp } from '@shared-ui/utils/datetime.js'
import { messageSnippet } from '@main/utils/threadFold'
import { initials as toInitials } from '@main/utils/initials'

const props = defineProps({
  message: { type: Object, required: true }
})
defineEmits(['expand'])

const { t } = useI18n()

const name = computed(() => {
  const author = props.message.author ?? {}
  return `${author.first_name ?? ''} ${author.last_name ?? ''}`.trim() || author.email || ''
})
const initials = computed(() => toInitials(props.message.author?.first_name, props.message.author?.last_name))
const snippet = computed(() => messageSnippet(props.message))
const fullTime = computed(() => formatFullTimestamp(props.message.created_at))
// Date and time are split so narrow panes can show the time only (the full stamp stays in `title`).
const time = computed(() => {
  const full = formatMessageTimestamp(props.message.created_at)
  const cut = full.lastIndexOf(', ')
  if (cut < 0) return { full, date: '', time: full }
  // A message from today shows the time only, like the open card's header.
  const date = isToday(new Date(props.message.created_at)) ? '' : full.slice(0, cut + 2)
  return { full, date, time: full.slice(cut + 2) }
})
</script>
