<template>
  <div class="flex min-h-full flex-col">
    <div class="flex w-full flex-wrap items-center gap-3 pb-4">
      <SearchInput
        v-model="searchTerm"
        class="min-w-0 max-w-sm flex-1"
        :placeholder="$t('contact.searchByEmail')"
        :clear-label="$t('globals.messages.clearSearch')"
        @update:model-value="fetchContactsDebounced"
      />

      <Popover>
        <PopoverTrigger as-child>
          <Button
            type="button"
            variant="outline"
            size="icon"
            class="size-9"
            :aria-label="$t('globals.messages.sortBy')"
          >
            <ArrowDownWideNarrow size="18" class="cursor-pointer text-muted-foreground" />
          </Button>
        </PopoverTrigger>
        <PopoverContent class="flex w-[200px] flex-col gap-4 p-4">
          <Select v-model="orderByField" @update:model-value="fetchContacts">
            <SelectTrigger class="h-8 w-full">
              <SelectValue :placeholder="orderByField" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="'users.created_at'">{{ $t('globals.terms.createdAt') }}</SelectItem>
              <SelectItem :value="'users.email'">{{ $t('globals.terms.email') }}</SelectItem>
            </SelectContent>
          </Select>

          <Select v-model="orderByDirection" @update:model-value="fetchContacts">
            <SelectTrigger class="h-8 w-full">
              <SelectValue :placeholder="orderByDirection" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="'asc'">{{ $t('globals.terms.ascending') }}</SelectItem>
              <SelectItem :value="'desc'">{{ $t('globals.terms.descending') }}</SelectItem>
            </SelectContent>
          </Select>
        </PopoverContent>
      </Popover>

      <Button
        :variant="blockedOnly ? 'default' : 'outline'"
        size="sm"
        class="flex h-8 items-center gap-1"
        :aria-pressed="blockedOnly"
        @click="toggleBlockedOnly"
      >
        <ShieldOffIcon size="14" />
        {{ $t('contact.blockedOnly') }}
      </Button>

      <Button
        v-if="userStore.can('contacts:write')"
        type="button"
        size="sm"
        class="ml-auto flex h-8 items-center"
        @click="showCreateDialog = true"
      >
        {{ $t('contact.new') }}
      </Button>
    </div>

    <div class="w-full border-t border-border/70">
      <div v-if="loading" class="w-full">
        <div
          v-for="i in perPage"
          :key="i"
          class="flex animate-pulse items-center gap-4 border-b border-border/70 px-3 py-3"
        >
          <Skeleton class="h-10 w-10 rounded-full" />
          <div class="space-y-2">
            <Skeleton class="h-3 w-40" />
            <Skeleton class="h-3 w-32" />
          </div>
        </div>
      </div>

      <template v-else>
        <router-link
          v-for="contact in contacts"
          :key="contact.id"
          :to="{ name: 'contact-detail', params: { id: contact.id } }"
          class="desk-directory-row px-3 py-3"
        >
          <div class="flex items-center gap-4">
            <Avatar class="h-10 w-10 border">
              <AvatarImage :src="contact.avatar_url || ''" />
              <AvatarFallback class="text-sm font-medium">
                {{ getInitials(contact.first_name, contact.last_name) }}
              </AvatarFallback>
            </Avatar>

            <div class="min-w-0 flex-1 space-y-1">
              <div class="flex items-center gap-2">
                <h4 class="ui-subheading truncate">
                  {{ contact.first_name }} {{ contact.last_name }}
                </h4>
                <Badge v-if="contact.type" variant="secondary" class="px-1.5 py-0 text-xs">
                  {{
                    contact.type === 'visitor'
                      ? $t('contact.type.visitor')
                      : $t('contact.type.contact')
                  }}
                </Badge>
                <Badge v-if="!contact.enabled" variant="destructive" class="px-1.5 py-0 text-xs">
                  {{ $t('globals.terms.blocked') }}
                </Badge>
              </div>
              <p class="truncate text-xs text-muted-foreground">
                {{ contact.email }}
              </p>
              <div
                v-if="contact.external_user_id"
                class="flex items-center gap-1 text-xs text-muted-foreground"
              >
                <IdCardIcon size="12" class="shrink-0" />
                <span class="truncate">{{ contact.external_user_id }}</span>
              </div>
            </div>
          </div>
        </router-link>
        <div
          v-if="contacts.length === 0"
          class="flex h-32 w-full items-center justify-center border-b border-border/70"
        >
          <p class="text-sm text-muted-foreground">{{ $t('contact.noContactsFound') }}</p>
        </div>
      </template>
    </div>

    <PaginationBar v-model:page="page" v-model:per-page="perPage" :total-pages="totalPages" />
  </div>

  <CreateContactDialog v-model:open="showCreateDialog" @created="onContactCreated" />
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import SearchInput from '@shared-ui/components/SearchInput.vue'
import { Button } from '@shared-ui/components/ui/button'
import { ArrowDownWideNarrow, IdCardIcon, ShieldOffIcon } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { useDebounceFn } from '@vueuse/core'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
import CreateContactDialog from './CreateContactDialog.vue'
import { useUserStore } from '@main/stores/user'
import api from '@main/api'

const contacts = ref([])
const loading = ref(false)
const page = ref(1)
const perPage = ref(15)
const totalPages = ref(0)
const searchTerm = ref('')
const orderByField = ref('users.created_at')
const orderByDirection = ref('desc')
const total = ref(0)
const showCreateDialog = ref(false)
const blockedOnly = ref(false)
const emitter = useEmitter()
const router = useRouter()
const userStore = useUserStore()
let fetchRequestId = 0

const fetchContactsDebounced = useDebounceFn(() => {
  if (page.value === 1) {
    fetchContacts()
  } else {
    page.value = 1
  }
}, 300)

const fetchContacts = async () => {
  const requestId = ++fetchRequestId
  loading.value = true
  const filters = []
  if (searchTerm.value && searchTerm.value.length >= 3) {
    filters.push({
      model: 'users',
      field: 'email',
      operator: 'ilike',
      value: searchTerm.value
    })
  }
  if (blockedOnly.value) {
    filters.push({ model: 'users', field: 'enabled', operator: 'equals', value: 'false' })
  }
  const filterJSON = filters.length ? JSON.stringify(filters) : ''
  try {
    const response = await api.getContacts({
      page: page.value,
      page_size: perPage.value,
      filters: filterJSON,
      order: orderByDirection.value,
      order_by: orderByField.value
    })
    if (requestId !== fetchRequestId) return
    contacts.value = response.data.data.results
    totalPages.value = response.data.data.total_pages
    total.value = response.data.data.total
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

const toggleBlockedOnly = () => {
  blockedOnly.value = !blockedOnly.value
  fetchContactsDebounced()
}

const getInitials = (firstName, lastName) => {
  return `${firstName?.[0] || ''}${lastName?.[0] || ''}`.toUpperCase()
}

const onContactCreated = (contact) => {
  router.push({ name: 'contact-detail', params: { id: contact.id } })
}

watch([page, perPage], fetchContacts)

onMounted(() => {
  fetchContacts()
})
</script>
