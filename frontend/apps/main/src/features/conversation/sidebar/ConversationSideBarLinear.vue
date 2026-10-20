<template>
  <!-- Linear issues linked to this conversation (Clyr fork). Hidden when Linear is not set up. -->
  <AccordionItem
    v-if="enabled"
    value="linear_issues"
    class="accordion-item"
    data-cy="sidebar-linear"
  >
    <AccordionTrigger class="accordion-trigger">
      {{ t('linear.issues') }}
    </AccordionTrigger>
    <AccordionContent class="accordion-content space-y-2">
      <p v-if="issues.length === 0" class="text-sm text-muted-foreground">
        {{ t('linear.noIssues') }}
      </p>
      <div
        v-for="issue in issues"
        :key="issue.id"
        class="flex items-start gap-2 text-sm"
        data-cy="linear-issue"
      >
        <a
          :href="issue.url"
          target="_blank"
          rel="noopener noreferrer"
          class="min-w-0 flex-1 hover:underline"
        >
          <span class="font-medium">{{ issue.identifier }}</span>
          <span v-if="issue.title"> {{ issue.title }}</span>
        </a>
        <span
          v-if="issue.state"
          class="shrink-0 rounded bg-muted px-1.5 text-xs text-muted-foreground"
        >
          {{ issue.state }}
        </span>
        <button
          v-if="canEdit"
          type="button"
          class="shrink-0 text-muted-foreground hover:text-foreground"
          :aria-label="t('linear.unlink', { identifier: issue.identifier })"
          :title="t('linear.unlink', { identifier: issue.identifier })"
          @click="unlink(issue)"
        >
          <X size="14" />
        </button>
      </div>
      <form v-if="canEdit" class="flex gap-2" @submit.prevent="link">
        <Input
          v-model="ref_"
          class="h-7 text-sm"
          :placeholder="t('linear.refPlaceholder')"
          :disabled="saving"
          data-cy="linear-ref"
        />
        <Button type="submit" size="sm" class="h-7" :disabled="saving || !ref_.trim()">
          {{ t('linear.link') }}
        </Button>
      </form>
    </AccordionContent>
  </AccordionItem>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { X } from 'lucide-vue-next'
import {
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@shared-ui/components/ui/accordion'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useConversationStore } from '@/stores/conversation'
import { useUserStore } from '@/stores/user'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()
const userStore = useUserStore()
const conversationStore = useConversationStore()
const uuid = computed(() => conversationStore.current?.uuid)
const canEdit = computed(() => userStore.can('conversations:update_tags'))

const enabled = ref(false)
const issues = ref([])
const ref_ = ref('')
const saving = ref(false)

const toastError = (err) =>
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(err).message
  })

watch(
  uuid,
  async (id) => {
    issues.value = []
    ref_.value = ''
    if (!id) return
    try {
      const { data } = await api.getLinearIssues(id)
      // The agent may have opened another conversation while this one loaded.
      if (uuid.value !== id) return
      enabled.value = data.data.enabled
      issues.value = data.data.issues || []
    } catch {
      if (uuid.value === id) enabled.value = false
    }
  },
  { immediate: true }
)

async function link() {
  const id = uuid.value
  saving.value = true
  try {
    const { data } = await api.linkLinearIssue(id, ref_.value.trim())
    if (uuid.value !== id) return
    issues.value = [...issues.value.filter((i) => i.id !== data.data.id), data.data]
    ref_.value = ''
  } catch (err) {
    toastError(err)
  } finally {
    saving.value = false
  }
}

async function unlink(issue) {
  const id = uuid.value
  try {
    await api.unlinkLinearIssue(id, issue.id)
    if (uuid.value !== id) return
    issues.value = issues.value.filter((i) => i.id !== issue.id)
  } catch (err) {
    toastError(err)
  }
}
</script>
