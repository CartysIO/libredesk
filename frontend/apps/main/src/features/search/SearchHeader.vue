<template>
  <div class="desk-search-header flex min-h-[3.75rem] shrink-0 items-center gap-4 border-b-2 border-foreground/10 px-4">
    <SidebarTrigger class="desk-page-trigger cursor-pointer" />
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
