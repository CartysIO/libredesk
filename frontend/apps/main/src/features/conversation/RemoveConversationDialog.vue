<template>
  <AlertDialog :open="open" @update:open="$emit('update:open', $event)">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>
          {{ t(isSpam ? 'conversation.markAsSpam' : 'conversation.deleteConversation') }}
        </AlertDialogTitle>
        <AlertDialogDescription>
          {{
            isSpam
              ? t('conversation.markAsSpamConfirm', { contact })
              : t('conversation.deleteConfirm')
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="$emit('confirm')">
          {{ t(isSpam ? 'conversation.markAsSpam' : 'globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'

// Confirms deleting a conversation or marking it as spam.
const props = defineProps({
  open: { type: Boolean, default: false },
  // 'delete' or 'spam'
  action: { type: String, default: 'delete' },
  // Named in the spam warning.
  contact: { type: String, default: '' }
})
defineEmits(['update:open', 'confirm'])

const { t } = useI18n()
const isSpam = computed(() => props.action === 'spam')
</script>
