<template>
  <div class="desk-search-header flex h-16 shrink-0 items-center gap-4 border-b border-border/60 px-4">
    <SidebarTrigger
      class="cursor-pointer rounded-lg border border-border/70 bg-background p-1 shadow-sm"
    />
    <h1 class="ui-heading min-w-0 truncate text-lg font-semibold tracking-tight">
      {{ t('globals.terms.search') }}
    </h1>
  </div>
</template>

<script setup>
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>
