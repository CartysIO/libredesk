<template>
  <div class="flex min-h-full flex-col">
    <div class="desk-directory-toolbar mb-4 flex w-full items-center gap-4">
      <Input
        v-model="searchTerm"
        type="text"
        class="max-w-sm"
        :placeholder="$t('organization.searchPlaceholder')"
        @input="fetchOrganizationsDebounced"
      />

      <Button
        v-if="userStore.can('contacts:write')"
        size="sm"
        class="ml-auto flex h-8 items-center"
        @click="showCreateDialog = true"
      >
        {{ $t('organization.new') }}
      </Button>
    </div>

    <div class="desk-directory-stack w-full">
      <div v-if="loading" class="w-full">
        <div
          v-for="i in perPage"
          :key="i"
          class="flex animate-pulse items-center gap-4 border-b border-border/70 px-3 py-3"
        >
          <Skeleton class="h-10 w-10 rounded-md" />
          <div class="space-y-2">
            <Skeleton class="h-3 w-40" />
            <Skeleton class="h-3 w-28" />
          </div>
        </div>
      </div>

      <template v-else>
        <router-link
          v-for="org in organizations"
          :key="org.id"
          :to="{ name: 'organization-detail', params: { id: org.id } }"
          class="desk-directory-row px-3 py-3"
          data-cy="organization-row"
        >
          <div class="flex items-center gap-4">
            <div class="flex h-10 w-10 items-center justify-center rounded-md border">
              <Building2 size="18" class="text-muted-foreground" />
            </div>
            <div class="min-w-0 flex-1 space-y-1">
              <h4 class="truncate text-sm font-semibold">{{ org.name }}</h4>
              <div
                v-if="org.external_id"
                class="flex items-center gap-1 text-xs text-muted-foreground"
              >
                <IdCardIcon size="12" class="shrink-0" />
                <span class="truncate">{{ org.external_id }}</span>
              </div>
            </div>
            <Badge variant="secondary" class="shrink-0 text-xs tabular-nums">
              {{ $t('organization.contactCount', { count: org.contact_count }, org.contact_count) }}
            </Badge>
          </div>
        </router-link>
        <div
          v-if="organizations.length === 0"
          class="desk-empty-list flex h-32 w-full items-center justify-center"
        >
          <p class="text-sm text-muted-foreground">{{ $t('organization.noneFound') }}</p>
        </div>
      </template>
    </div>

    <PaginationBar v-model:page="page" v-model:per-page="perPage" :total-pages="totalPages" />
  </div>

  <OrganizationFormDialog v-model:open="showCreateDialog" @saved="onCreated" />
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useDebounceFn } from '@vueuse/core'
import { Building2, IdCardIcon } from 'lucide-vue-next'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { Badge } from '@shared-ui/components/ui/badge'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
import { useUserStore } from '@main/stores/user'
import OrganizationFormDialog from './OrganizationFormDialog.vue'
import api from '@main/api'

const organizations = ref([])
const loading = ref(false)
const page = ref(1)
const perPage = ref(15)
const totalPages = ref(0)
const searchTerm = ref('')
const showCreateDialog = ref(false)
const emitter = useEmitter()
const router = useRouter()
const userStore = useUserStore()
let fetchRequestId = 0

const fetchOrganizationsDebounced = useDebounceFn(() => {
  if (page.value === 1) {
    fetchOrganizations()
  } else {
    page.value = 1
  }
}, 300)

const fetchOrganizations = async () => {
  const requestId = ++fetchRequestId
  loading.value = true
  try {
    const response = await api.getOrganizations({
      page: page.value,
      page_size: perPage.value,
      q: searchTerm.value.trim()
    })
    if (requestId !== fetchRequestId) return
    organizations.value = response.data.data.results || []
    totalPages.value = response.data.data.total_pages
  } catch (error) {
    if (requestId !== fetchRequestId) return
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    if (requestId === fetchRequestId) loading.value = false
  }
}

const onCreated = (org) => {
  router.push({ name: 'organization-detail', params: { id: org.id } })
}

watch([page, perPage], fetchOrganizations)

onMounted(fetchOrganizations)
</script>
