<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>
          {{ organization ? t('organization.edit') : t('organization.new') }}
        </DialogTitle>
      </DialogHeader>
      <form class="space-y-4" @submit.prevent="onSubmit">
        <div class="space-y-2">
          <Label for="organization-name">{{ t('globals.terms.name') }}</Label>
          <Input id="organization-name" v-model="name" type="text" required />
        </div>
        <div class="space-y-2">
          <Label for="organization-external-id">{{ t('organization.externalId') }}</Label>
          <Input id="organization-external-id" v-model="externalID" type="text" />
          <p class="text-xs text-muted-foreground">{{ t('organization.externalIdHelp') }}</p>
        </div>
        <div class="space-y-2">
          <Label for="organization-domains">{{ t('organization.domains') }}</Label>
          <TagsInput
            id="organization-domains"
            v-model="domains"
            :addOnBlur="true"
            :addOnPaste="true"
            :addOnTab="true"
            delimiter=","
          >
            <TagsInputItem v-for="item in domains" :key="item" :value="item">
              <TagsInputItemText />
              <TagsInputItemDelete />
            </TagsInputItem>
            <TagsInputInput placeholder="acme.com" />
          </TagsInput>
          <p class="text-xs text-muted-foreground">{{ t('organization.domainsHelp') }}</p>
        </div>
        <DialogFooter>
          <Button type="submit" :disabled="loading || !name.trim()" :isLoading="loading">
            {{ organization ? t('globals.messages.save') : t('globals.messages.create') }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText
} from '@shared-ui/components/ui/tags-input'
import { Button } from '@shared-ui/components/ui/button'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import api from '@main/api'

const props = defineProps({
  open: { type: Boolean, default: false },
  // The organization to edit. Empty creates a new one.
  organization: { type: Object, default: null }
})

const emit = defineEmits(['update:open', 'saved'])

const { t } = useI18n()
const emitter = useEmitter()
const loading = ref(false)
const name = ref('')
const externalID = ref('')
const domains = ref([])

watch(
  () => props.open,
  (isOpen) => {
    if (!isOpen) return
    name.value = props.organization?.name || ''
    externalID.value = props.organization?.external_id || ''
    domains.value = [...(props.organization?.domains || [])]
  },
  { immediate: true }
)

const onSubmit = async () => {
  loading.value = true
  const body = {
    name: name.value.trim(),
    external_id: externalID.value.trim(),
    domains: domains.value
  }
  try {
    const { data } = props.organization
      ? await api.updateOrganization(props.organization.id, body)
      : await api.createOrganization(body)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    emit('update:open', false)
    emit('saved', data.data)
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    loading.value = false
  }
}
</script>
