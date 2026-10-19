<template>
  <!-- The organization this conversation is filed under, picked from the contact's organizations (Clyr fork). -->
  <div
    v-if="conversation && (conversation.organization_id || options.length > 0)"
    class="flex gap-2 items-center"
    data-cy="sidebar-organization"
  >
    <Building2 size="16" class="text-muted-foreground flex-shrink-0" />
    <Select
      v-if="canChange"
      :model-value="selected"
      :disabled="saving"
      @update:model-value="onChange"
    >
      <SelectTrigger class="h-7 min-w-0 flex-1 text-sm" data-cy="sidebar-organization-select">
        <SelectValue :placeholder="t('organization.select')" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem v-for="org in options" :key="org.id" :value="String(org.id)">
          {{ org.name }}
          <span v-if="org.is_default" class="text-xs text-muted-foreground">
            · {{ t('organization.default') }}
          </span>
          <span v-else-if="org.former" class="text-xs text-muted-foreground">
            · {{ t('organization.notMember') }}
          </span>
        </SelectItem>
        <SelectItem value="none">{{ t('globals.terms.none') }}</SelectItem>
      </SelectContent>
    </Select>
    <template v-else-if="conversation.organization_id">
      <router-link
        v-if="userStore.can('contacts:read')"
        :to="{ name: 'organization-detail', params: { id: conversation.organization_id } }"
        class="sidebar-value min-w-0 truncate hover:underline"
      >
        {{ conversation.organization_name }}
      </router-link>
      <span v-else class="sidebar-value min-w-0 truncate">
        {{ conversation.organization_name }}
      </span>
    </template>
    <span v-else class="sidebar-label">{{ t('globals.terms.none') }}</span>
    <router-link
      v-if="canChange && conversation.organization_id && userStore.can('contacts:read')"
      :to="{ name: 'organization-detail', params: { id: conversation.organization_id } }"
      class="shrink-0 text-muted-foreground hover:text-foreground"
      :title="conversation.organization_name"
    >
      <ExternalLink size="14" />
    </router-link>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Building2, ExternalLink } from 'lucide-vue-next'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
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
const conversation = computed(() => conversationStore.current)
const saving = ref(false)

// The contact's organizations. The conversation's own organization is kept in the list after the contact
// leaves it, so the picker still shows where the conversation is filed.
const options = computed(() => {
  const c = conversation.value
  const list = c?.contact?.organizations || []
  if (c?.organization_id && !list.some((o) => o.id === c.organization_id)) {
    return [...list, { id: c.organization_id, name: c.organization_name, former: true }]
  }
  return list
})
const selected = computed(() =>
  conversation.value?.organization_id ? String(conversation.value.organization_id) : 'none'
)
// The picker shows when the agent may change it and the contact has organizations to pick from.
const canChange = computed(
  () => userStore.can('conversations:update_organization') && options.value.length > 0
)

async function onChange(value) {
  if (value === selected.value) return
  const uuid = conversation.value.uuid
  saving.value = true
  try {
    const { data } = await api.setConversationOrganization(
      uuid,
      value === 'none' ? null : Number(value)
    )
    conversationStore.mergeConversationUpdate({ uuid, ...data.data })
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    saving.value = false
  }
}
</script>
