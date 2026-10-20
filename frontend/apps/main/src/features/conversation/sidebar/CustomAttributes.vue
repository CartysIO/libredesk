<template>
  <div class="space-y-3">
    <div
      v-for="attribute in attributes"
      :key="attribute.id"
      class="grid min-w-0 grid-cols-[minmax(0,2fr)_minmax(0,3fr)] items-start gap-x-4"
    >
      <div class="sidebar-label min-w-0 py-0.5 [overflow-wrap:anywhere]">
        <Tooltip :disabled="!attribute.description">
          <TooltipTrigger as-child>
            <span>{{ attribute.name }}</span>
          </TooltipTrigger>
          <TooltipContent v-if="attribute.description">{{ attribute.description }}</TooltipContent>
        </Tooltip>
      </div>

      <Checkbox
        v-if="attribute.data_type === 'checkbox'"
        type="button"
        class="mt-1 justify-self-end"
        :disabled="loading"
        :aria-label="attribute.name"
        :checked="!!customAttributes?.[attribute.key]"
        @update:checked="(value) => saveAttribute(attribute.key, value)"
      />

      <Popover
        v-else
        :open="editingAttributeKey === attribute.key"
        @update:open="(open) => (open ? startEditing(attribute) : cancelEditing())"
      >
        <PopoverTrigger as-child>
          <Button
            type="button"
            variant="link"
            :disabled="loading"
            class="sidebar-value h-auto min-h-6 min-w-6 max-w-full justify-end justify-self-end rounded-sm whitespace-normal px-0 py-0.5 text-right font-normal decoration-muted-foreground/40 decoration-dotted underline-offset-4 can-hover:hover:decoration-foreground"
            :class="{ 'text-muted-foreground': !hasValue(customAttributes?.[attribute.key]) }"
            :aria-label="`${t('globals.messages.edit')}: ${attribute.name}`"
          >
            <span class="line-clamp-2 min-w-0 [overflow-wrap:anywhere]">{{
              displayValue(customAttributes?.[attribute.key])
            }}</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" class="w-72" :aria-label="attribute.name">
          <form class="space-y-3" novalidate @submit.stop.prevent="saveAttribute(attribute.key)">
            <div class="space-y-1.5">
              <Label :for="`${editorId}-${attribute.key}`">{{ attribute.name }}</Label>
              <p v-if="attribute.description" class="text-xs text-muted-foreground">
                {{ attribute.description }}
              </p>
            </div>

            <Select v-if="attribute.data_type === 'list'" v-model="editingValue">
              <SelectTrigger
                :id="`${editorId}-${attribute.key}`"
                :aria-invalid="!!errorMessage"
                :aria-describedby="errorMessage ? `${editorId}-${attribute.key}-error` : undefined"
              >
                <SelectValue :placeholder="t('placeholders.selectValue')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="option in attribute.values" :key="option" :value="option">
                  {{ option }}
                </SelectItem>
              </SelectContent>
            </Select>
            <SelectTag
              v-else-if="attribute.data_type === 'multi_select'"
              :id="`${editorId}-${attribute.key}`"
              v-model="editingValue"
              :items="attribute.values.map((v) => ({ label: v, value: v }))"
              :placeholder="t('placeholders.selectValue')"
            />
            <Input
              v-else
              :id="`${editorId}-${attribute.key}`"
              v-model="editingValue"
              :type="INPUT_TYPES[attribute.data_type] || 'text'"
              :class="{ 'hide-number-spinners': attribute.data_type === 'number' }"
              :aria-invalid="!!errorMessage"
              :aria-describedby="errorMessage ? `${editorId}-${attribute.key}-error` : undefined"
            />

            <p
              v-if="errorMessage"
              :id="`${editorId}-${attribute.key}-error`"
              class="text-destructive text-xs"
              role="alert"
            >
              {{ errorMessage }}
            </p>
            <div class="flex items-center justify-between gap-2">
              <Button
                v-if="hasValue(customAttributes?.[attribute.key])"
                type="button"
                variant="ghost"
                size="sm"
                class="text-muted-foreground"
                @click="deleteAttribute(attribute)"
              >
                {{ t('globals.terms.remove') }}
              </Button>
              <div class="ml-auto flex items-center gap-2">
                <Button type="button" variant="outline" size="sm" @click="cancelEditing">
                  {{ t('globals.messages.cancel') }}
                </Button>
                <Button type="submit" size="sm">{{ t('globals.messages.save') }}</Button>
              </div>
            </div>
          </form>
        </PopoverContent>
      </Popover>
    </div>
  </div>
</template>

<script setup>
import { ref, useId } from 'vue'
import * as z from 'zod'
import { Input } from '@shared-ui/components/ui/input'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SelectTag
} from '@shared-ui/components/ui/select'
import { Button } from '@shared-ui/components/ui/button'
import { Label } from '@shared-ui/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  attributes: {
    type: Array,
    required: true
  },
  customAttributes: {
    type: Object,
    required: true
  },
  loading: {
    type: Boolean,
    default: false
  }
})
const emit = defineEmits(['update:setattributes'])
const { t } = useI18n()
const editorId = useId()
const INPUT_TYPES = { text: 'text', number: 'number', date: 'date', link: 'url' }
const errorMessage = ref('')
const editingAttributeKey = ref(null)
const editingValue = ref(null)

const startEditing = (attribute) => {
  errorMessage.value = ''
  editingAttributeKey.value = attribute.key
  const currentValue = props.customAttributes?.[attribute.key]
  if (attribute.data_type === 'checkbox') editingValue.value = !!currentValue
  else if (attribute.data_type === 'multi_select')
    editingValue.value = Array.isArray(currentValue) ? [...currentValue] : []
  else editingValue.value = currentValue ?? null
}

// Multi-select values are arrays; an empty array shows as unset.
const hasValue = (value) => (Array.isArray(value) ? value.length > 0 : value != null)
const displayValue = (value) => {
  if (Array.isArray(value)) return value.length ? value.join(', ') : t('globals.messages.set')
  return value ?? t('globals.messages.set')
}

const cancelEditing = () => {
  editingAttributeKey.value = null
  editingValue.value = null
  errorMessage.value = ''
}

const getValidationSchema = (attribute) => {
  switch (attribute.data_type) {
    case 'text': {
      let schema = z.string().min(1, t('globals.messages.required'))
      // If regex is provided and valid, add it to the schema validation along with the hint
      if (attribute.regex) {
        try {
          const regex = new RegExp(attribute.regex)
          schema = schema.regex(regex, {
            message: attribute.regex_hint
          })
        } catch (err) {
          console.error('Error creating regex:', err)
        }
      }
      return schema.nullable()
    }
    case 'number':
      return z.preprocess(
        (val) => Number(val),
        z
          .number({
            invalid_type_error: t('validation.invalidValue')
          })
          .nullable()
      )
    case 'checkbox':
      return z.boolean().nullable()
    case 'date':
      return z
        .string()
        .refine((val) => !isNaN(Date.parse(val)), t('validation.invalidValue'))
        .nullable()
    case 'link':
      return z.string().url(t('validation.invalidUrl')).nullable()
    case 'list':
      return z
        .string()
        .refine((val) => attribute.values.includes(val), {
          message: t('validation.invalidValue')
        })
        .nullable()
    case 'multi_select':
      return z
        .array(z.string())
        .refine((vals) => vals.every((val) => attribute.values.includes(val)), {
          message: t('validation.invalidValue')
        })
    default:
      return z.any()
  }
}

const saveAttribute = (key, value = editingValue.value) => {
  const attribute = props.attributes.find((attr) => attr.key === key)
  if (!attribute) return

  try {
    const schema = getValidationSchema(attribute)
    schema.parse(value)
  } catch (validationError) {
    if (validationError instanceof z.ZodError) {
      errorMessage.value = validationError.errors[0].message
      return
    }
    errorMessage.value = validationError
    return
  }

  const updatedAttributes = { ...(props.customAttributes || {}) }
  updatedAttributes[attribute.key] = value
  emit('update:setattributes', updatedAttributes)
  cancelEditing()
}

const deleteAttribute = (attribute) => {
  const updatedAttributes = { ...(props.customAttributes || {}) }
  delete updatedAttributes[attribute.key]
  emit('update:setattributes', updatedAttributes)
  if (editingAttributeKey.value === attribute.key) cancelEditing()
}
</script>
