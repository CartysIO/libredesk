<template>
  <div v-if="!isHidden" class="desk-page-header shrink-0 border-b border-border/60">
    <div class="flex h-16 shrink-0 items-center gap-4 px-5">
      <SidebarTrigger class="cursor-pointer rounded-lg border border-border/70 bg-card p-1 shadow-sm" />
      <div class="min-w-0 flex-1">
        <p
          v-if="sectionHint"
          class="text-[0.6875rem] font-semibold uppercase tracking-widest text-muted-foreground"
        >
          {{ sectionHint }}
        </p>
        <h1 class="desk-title font-display min-w-0 truncate text-lg font-semibold tracking-tight">
          {{ title }}
        </h1>
      </div>
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

const sectionHint = computed(() => {
  const path = route.path
  if (path.startsWith('/admin')) return t('globals.terms.admin')
  if (path.startsWith('/inboxes')) return t('globals.terms.inbox', 2)
  if (path.startsWith('/contacts')) return t('globals.terms.contact', 2)
  if (path.startsWith('/organizations')) return t('globals.terms.organization', 2)
  if (path.startsWith('/reports')) return t('globals.terms.report', 2)
  if (path.startsWith('/account')) return t('globals.terms.account')
  return ''
})
</script>
