<template>
  <header v-if="!isHidden" class="desk-page-header shrink-0">
    <div class="desk-page-header__bar flex min-h-[3.75rem] items-stretch">
      <p class="desk-page-header__rail" aria-hidden="true" />
      <div class="flex min-w-0 flex-1 items-center gap-3 px-4 py-3">
        <SidebarTrigger class="desk-page-trigger cursor-pointer" />
        <div class="min-w-0 flex-1 border-l border-border/80 pl-4">
          <p v-if="sectionHint" class="desk-page-header__meta">
            {{ sectionHint }}
          </p>
          <h1 class="desk-title min-w-0 truncate text-xl font-semibold tracking-tight">
            {{ title }}
          </h1>
        </div>
      </div>
    </div>
  </header>
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
