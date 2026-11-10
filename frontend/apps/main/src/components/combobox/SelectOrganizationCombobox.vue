<template>
  <SelectComboBox
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
    :search="organizationStore.searchOrganizations"
    type="organization"
  >
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>
  </SelectComboBox>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useOrganizationStore } from '@/stores/organization'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: undefined
  },
  placeholder: {
    type: String,
    default: ''
  },
  includeNone: {
    type: Boolean,
    default: false
  }
})

const { t } = useI18n()
const organizationStore = useOrganizationStore()

const placeholderText = computed(() => props.placeholder || t('organization.select'))

const items = computed(() => {
  const prepended = props.includeNone ? [{ value: 'none', label: t('globals.terms.none') }] : []
  return [...prepended, ...organizationStore.options]
})

onMounted(organizationStore.fetchOrganizations)

watch(
  () => props.modelValue,
  (value) => organizationStore.ensureOrganizationIDs([value]),
  { immediate: true }
)
</script>
