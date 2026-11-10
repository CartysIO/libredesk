<template>
  <div class="space-y-2">
    <div v-if="ids.length" class="flex flex-wrap gap-1.5">
      <span
        v-for="id in ids"
        :key="id"
        class="inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-xs"
      >
        {{ labelFor(id) }}
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground"
          @click="remove(id)"
        >
          <X size="12" />
          <span class="sr-only">{{ t('globals.terms.remove') }}</span>
        </button>
      </span>
    </div>
    <SelectOrganizationCombobox :model-value="''" @select="add" />
  </div>
</template>

<script setup>
// SelectOrganizationsCombobox picks several organizations. Its value is a comma-separated list of IDs,
// the format automation rules store for "is any of" conditions (Clyr fork).
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { X } from 'lucide-vue-next'
import { useOrganizationStore } from '@/stores/organization'
import SelectOrganizationCombobox from '@/components/combobox/SelectOrganizationCombobox.vue'

const props = defineProps({
  modelValue: { type: String, default: '' }
})
const emit = defineEmits(['update:modelValue'])

const { t } = useI18n()
const organizationStore = useOrganizationStore()

const ids = computed(() =>
  (props.modelValue || '')
    .split(',')
    .map((id) => id.trim())
    .filter(Boolean)
)

const labelFor = (id) => organizationStore.options.find((o) => o.value === id)?.label || `#${id}`

const add = (item) => {
  const id = String(item?.value ?? '')
  if (!id || ids.value.includes(id)) return
  emit('update:modelValue', [...ids.value, id].join(','))
}

const remove = (id) => emit('update:modelValue', ids.value.filter((v) => v !== id).join(','))

watch(ids, (value) => organizationStore.ensureOrganizationIDs(value), { immediate: true })
</script>
