<template>
  <button
    type="button"
    class="desk-fold-row group/fold"
    :class="{ 'desk-fold-row--private': message.private }"
    :aria-expanded="false"
    @click="$emit('expand')"
  >
    <Avatar class="desk-fold-row__avatar h-6 w-6 shrink-0 text-[10px]">
      <AvatarImage :src="message.author?.avatar_url || ''" />
      <AvatarFallback class="font-semibold">{{ initials }}</AvatarFallback>
    </Avatar>
    <span class="desk-fold-row__name">{{ name }}</span>
    <Lock v-if="message.private" class="desk-fold-row__icon" :aria-label="t('globals.terms.privateNote')" />
    <Paperclip
      v-if="message.attachments?.length"
      class="desk-fold-row__icon"
      :aria-label="t('globals.terms.attachment', message.attachments.length)"
    />
    <span class="desk-fold-row__snippet">{{ snippet }}</span>
    <time class="desk-fold-row__time" :datetime="message.created_at">{{ time }}</time>
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Lock, Paperclip } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { formatMessageTimestamp } from '@shared-ui/utils/datetime.js'
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
const time = computed(() => formatMessageTimestamp(props.message.created_at))
</script>
