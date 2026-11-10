<template>
  <div v-if="!isHidden" class="shrink-0 border-b border-border/70">
    <div class="flex h-14 shrink-0 items-center gap-3 px-3">
      <SidebarTrigger class="cursor-pointer" />
      <h1 class="desk-title min-w-0 truncate text-xl font-semibold">
        {{ title }}
      </h1>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const { t } = useI18n()
const title = computed(() => {
  const key = route.meta?.titleKey
  if (!key) return ''
  return t(key, route.meta?.titleCount || 1)
})
const isHidden = computed(() => route.meta.hidePageHeader === true)
</script>
