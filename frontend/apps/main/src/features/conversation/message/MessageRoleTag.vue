<template>
  <!-- Role tag next to a sender's name: "Customer" for the other party, "AI" for an assistant reply. -->
  <span v-if="role === 'customer'" class="desk-role-tag desk-role-tag--customer">
    <!-- The default covers a server whose catalog predates this key. -->
    <span class="desk-role-tag__text">{{ t('globals.terms.customer', 'Customer') }}</span>
  </span>
  <span v-else-if="role === 'ai'" class="desk-role-tag desk-role-tag--ai">
    <Sparkles class="desk-role-tag__icon" aria-hidden="true" />
    <span class="desk-role-tag__text">{{ t('admin.ai.title') }}</span>
  </span>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { Sparkles } from 'lucide-vue-next'

defineProps({
  role: { type: String, default: 'agent' }
})

const { t } = useI18n()
</script>
