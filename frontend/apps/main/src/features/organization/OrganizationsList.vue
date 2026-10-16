<template>
  <div class="min-h-screen flex flex-col">
    <div class="flex flex-wrap gap-4 pb-4">
      <div class="flex items-center gap-4 mb-4 w-full">
        <Input
          type="text"
          class="max-w-sm"
          v-model="searchTerm"
          :placeholder="$t('organization.searchPlaceholder')"
          @input="fetchOrganizationsDebounced"
        />

        <Button
          v-if="userStore.can('contacts:write')"
          size="sm"
          class="flex items-center h-8 ml-auto"
          @click="showCreateDialog = true"
        >
          {{ $t('organization.new') }}
        </Button>
      </div>

      <div v-if="loading" class="flex flex-col gap-4 w-full">
        <Card v-for="i in perPage" :key="i" class="p-4 flex-shrink-0">
          <div class="flex items-center gap-4">
            <Skeleton class="h-10 w-10 rounded-md" />
            <div class="space-y-2">
              <Skeleton class="h-3 w-[160px]" />
              <Skeleton class="h-3 w-[140px]" />
            </div>
          </div>
        </Card>
      </div>

      <template v-else>
        <Card
          v-for="org in organizations"
          :key="org.id"
          class="p-4 w-full hover:bg-accent/50 cursor-pointer"
          data-cy="organization-row"
          @click="$router.push({ name: 'organization-detail', params: { id: org.id } })"
        >
          <div class="flex items-center gap-4">
            <div class="h-10 w-10 rounded-md border flex items-center justify-center">
              <Building2 size="18" class="text-muted-foreground" />
            </div>
            <div class="space-y-1 overflow-hidden flex-1">
              <h4 class="text-sm font-semibold truncate">{{ org.name }}</h4>
              <div
                v-if="org.external_id"
                class="flex items-center gap-1 text-xs text-muted-foreground"
              >
                <IdCardIcon size="12" class="flex-shrink-0" />
                <span class="truncate">{{ org.external_id }}</span>
              </div>
            </div>
            <Badge variant="secondary" class="text-xs">
              {{ $t('organization.contactCount', { count: org.contact_count }, org.contact_count) }}
            </Badge>
          </div>
        </Card>
        <div v-if="organizations.length === 0" class="flex items-center justify-center w-full h-32">
          <p class="text-lg text-muted-foreground">{{ $t('organization.noneFound') }}</p>
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
import { Card } from '@shared-ui/components/ui/card'
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
