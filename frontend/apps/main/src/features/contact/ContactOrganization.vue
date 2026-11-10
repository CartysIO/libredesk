<template>
  <div class="space-y-2" data-cy="contact-organization">
    <h3 class="text-base font-semibold">{{ t('globals.terms.organization', 2) }}</h3>
    <p class="text-xs text-muted-foreground">{{ t('organization.defaultHelp') }}</p>
    <div
      v-for="org in organizations"
      :key="org.id"
      class="flex flex-wrap items-center gap-2 text-sm"
      data-cy="contact-organization-row"
    >
      <router-link
        :to="{ name: 'organization-detail', params: { id: org.id } }"
        class="flex items-center gap-1.5 hover:underline"
      >
        <Building2 size="14" class="text-muted-foreground" />
        {{ org.name }}
        <span v-if="org.external_id" class="text-xs text-muted-foreground">
          ({{ org.external_id }})
        </span>
      </router-link>
      <span
        v-if="org.is_default"
        class="rounded border px-1.5 py-0.5 text-xs text-muted-foreground"
        data-cy="contact-organization-default"
      >
        {{ t('organization.default') }}
      </span>
      <template v-if="canWrite">
        <Button
          v-if="!org.is_default"
          variant="ghost"
          size="sm"
          class="h-7 px-2 text-xs"
          :disabled="busy"
          @click="makeDefault(org.id)"
        >
          {{ t('organization.makeDefault') }}
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="h-7 w-7"
          :title="t('organization.remove')"
          :aria-label="t('organization.remove')"
          :disabled="busy"
          @click="remove(org.id)"
        >
          <X size="14" />
        </Button>
      </template>
    </div>
    <p v-if="loaded && organizations.length === 0" class="text-sm text-muted-foreground">
      {{ t('globals.terms.none') }}
    </p>
    <div v-if="canWrite" class="w-72">
      <SelectOrganizationCombobox
        :model-value="undefined"
        :placeholder="t('organization.add')"
        @select="add"
      />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Building2, X } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import SelectOrganizationCombobox from '@/components/combobox/SelectOrganizationCombobox.vue'
import { useUserStore } from '@/stores/user'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

const props = defineProps({
  contact: { type: Object, required: true }
})
// updated: the contact's organizations, the default one first.
const emit = defineEmits(['updated'])

const { t } = useI18n()
const emitter = useEmitter()
const userStore = useUserStore()

const organizations = ref([])
const loaded = ref(false)
const busy = ref(false)
const canWrite = computed(() => userStore.can('contacts:write'))

const showError = (err) =>
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(err).message
  })

async function fetchOrganizations() {
  try {
    const { data } = await api.getContactOrganizations(props.contact.id)
    organizations.value = data.data
    loaded.value = true
  } catch (err) {
    showError(err)
  }
}

async function run(request) {
  busy.value = true
  try {
    const { data } = await request()
    organizations.value = data.data
    emit('updated', data.data)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (err) {
    showError(err)
  } finally {
    busy.value = false
  }
}

const add = (item) => {
  const id = Number(item.value)
  if (!id || organizations.value.some((o) => o.id === id)) return
  return run(() => api.addContactOrganization(props.contact.id, id))
}
const remove = (id) => run(() => api.removeContactOrganization(props.contact.id, id))
const makeDefault = (id) => run(() => api.setContactDefaultOrganization(props.contact.id, id))

watch(() => props.contact.id, fetchOrganizations, { immediate: true })
</script>
