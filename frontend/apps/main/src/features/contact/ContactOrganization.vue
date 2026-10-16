<template>
  <div class="space-y-2" data-cy="contact-organization">
    <h3 class="text-base font-semibold">{{ t('globals.terms.organization') }}</h3>
    <div class="flex flex-wrap items-center gap-3">
      <div v-if="canWrite" class="w-72">
        <SelectOrganizationCombobox
          :model-value="selectedValue"
          include-none
          :placeholder="t('organization.select')"
          @select="onSelect"
        />
      </div>
      <router-link
        v-if="contact.organization_id"
        :to="{ name: 'organization-detail', params: { id: contact.organization_id } }"
        class="flex items-center gap-1.5 text-sm hover:underline"
      >
        <Building2 size="14" class="text-muted-foreground" />
        {{ contact.organization_name }}
        <span v-if="contact.organization_external_id" class="text-xs text-muted-foreground">
          ({{ contact.organization_external_id }})
        </span>
      </router-link>
      <span v-else-if="!canWrite" class="text-sm text-muted-foreground">
        {{ t('globals.terms.none') }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Building2 } from 'lucide-vue-next'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import SelectOrganizationCombobox from '@/components/combobox/SelectOrganizationCombobox.vue'
import { useUserStore } from '@/stores/user'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

const props = defineProps({
  contact: { type: Object, required: true }
})
const emit = defineEmits(['updated'])

const { t } = useI18n()
const emitter = useEmitter()
const userStore = useUserStore()

const canWrite = computed(() => userStore.can('contacts:write'))
const selectedValue = computed(() =>
  props.contact.organization_id ? String(props.contact.organization_id) : 'none'
)

const onSelect = async (item) => {
  if (item.value === selectedValue.value) return
  const organizationID = item.value === 'none' ? null : Number(item.value)
  try {
    const { data } = await api.setContactOrganization(props.contact.id, organizationID)
    emit('updated', data.data)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  }
}
</script>
