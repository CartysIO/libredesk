<template>
  <ContactDetail>
    <div v-if="canRead" class="flex flex-col mx-auto items-start">
      <div class="mb-6">
        <CustomBreadcrumb :links="breadcrumbLinks" />
      </div>

      <div v-if="organization" class="flex flex-col w-full mt-6 space-y-10">
        <div class="flex flex-col space-y-2">
          <div class="flex gap-2 items-center">
            <Building2 size="20" class="text-muted-foreground" />
            <h2 class="text-xl font-semibold text-foreground" data-cy="organization-name">
              {{ organization.name }}
            </h2>
            <DropdownMenu v-if="canWrite || canDelete">
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" class="h-7 w-7">
                  <MoreVerticalIcon class="h-4 w-4" />
                  <span class="sr-only">{{ t('globals.terms.openMenu') }}</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" class="w-[200px]">
                <DropdownMenuItem v-if="canWrite" class="cursor-pointer" @click="showEdit = true">
                  <PencilIcon class="mr-2" size="15" />
                  {{ t('globals.messages.edit') }}
                </DropdownMenuItem>
                <template v-if="canDelete">
                  <DropdownMenuSeparator v-if="canWrite" />
                  <DropdownMenuItem
                    class="text-destructive cursor-pointer"
                    @click="showDeleteConfirmation = true"
                  >
                    <Trash2Icon class="mr-2" size="15" />
                    {{ t('globals.messages.delete') }}
                  </DropdownMenuItem>
                </template>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <div
            v-if="organization.external_id"
            class="flex items-center gap-1.5 text-xs text-muted-foreground"
          >
            <IdCardIcon size="14" class="flex-shrink-0" />
            {{ organization.external_id }}
          </div>
          <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
            <CalendarIcon size="14" class="flex-shrink-0" />
            {{ $t('globals.terms.createdOn') }}
            {{ format(new Date(organization.created_at), 'PPP') }}
          </div>
        </div>

        <!-- Contacts -->
        <section class="w-full space-y-3" data-cy="organization-contacts">
          <h3 class="text-base font-semibold">
            {{ t('globals.terms.contact', 2) }}
            <span class="text-muted-foreground font-normal"
              >({{ organization.contact_count }})</span
            >
          </h3>
          <router-link
            v-for="contact in contacts"
            :key="contact.id"
            :to="{ name: 'contact-detail', params: { id: contact.id } }"
            class="flex items-center gap-3 p-3 rounded-md border hover:bg-accent/50"
          >
            <Avatar class="h-8 w-8 border">
              <AvatarImage :src="contact.avatar_url || ''" />
              <AvatarFallback class="text-xs">
                {{ initials(contact.first_name, contact.last_name) }}
              </AvatarFallback>
            </Avatar>
            <div class="min-w-0">
              <p class="text-sm font-medium truncate">
                {{ contact.first_name }} {{ contact.last_name }}
              </p>
              <p class="text-xs text-muted-foreground truncate">{{ contact.email }}</p>
            </div>
            <span
              v-if="contact.is_default"
              class="ml-auto shrink-0 rounded border px-1.5 py-0.5 text-xs text-muted-foreground"
              :title="t('organization.defaultHelp')"
            >
              {{ t('organization.default') }}
            </span>
          </router-link>
          <p v-if="contacts.length === 0" class="text-sm text-muted-foreground">
            {{ t('organization.noContacts') }}
          </p>
          <Button
            v-if="hasMoreContacts"
            variant="outline"
            size="sm"
            :isLoading="loadingContacts"
            @click="fetchContacts(contactsPage + 1)"
          >
            {{ t('globals.terms.loadMore') }}
          </Button>
        </section>

        <!-- Conversations, read through the organization_id filter -->
        <section
          v-if="userStore.can('conversations:read_all')"
          class="w-full space-y-3"
          data-cy="organization-conversations"
        >
          <h3 class="text-base font-semibold">
            {{ t('globals.terms.conversation', 2) }}
            <span class="text-muted-foreground font-normal">({{ conversationsTotal }})</span>
          </h3>
          <router-link
            v-for="conv in conversations"
            :key="conv.uuid"
            :to="{ name: 'inbox-conversation', params: { type: 'all', uuid: conv.uuid } }"
            class="flex items-start justify-between gap-3 p-3 rounded-md border hover:bg-accent/50"
          >
            <div class="min-w-0">
              <p class="text-sm font-medium truncate">
                #{{ conv.reference_number }} {{ conv.subject || conv.last_message }}
              </p>
              <p class="text-xs text-muted-foreground truncate">
                {{ conv.contact?.first_name }} {{ conv.contact?.last_name }}
              </p>
            </div>
            <div class="flex flex-col items-end gap-1 shrink-0">
              <Badge variant="secondary" class="text-xs">{{ conv.status }}</Badge>
              <span v-if="conv.last_message_at" class="text-xs text-muted-foreground">
                {{ getRelativeTime(new Date(conv.last_message_at)) }}
              </span>
            </div>
          </router-link>
          <p v-if="conversations.length === 0" class="text-sm text-muted-foreground">
            {{ t('organization.noConversations') }}
          </p>
          <Button
            v-if="conversations.length < conversationsTotal"
            variant="outline"
            size="sm"
            :isLoading="loadingConversations"
            @click="fetchConversations(conversationsPage + 1)"
          >
            {{ t('globals.terms.loadMore') }}
          </Button>
        </section>
      </div>

      <Spinner v-if="loading" />

      <OrganizationFormDialog
        v-model:open="showEdit"
        :organization="organization"
        @saved="onSaved"
      />

      <AlertDialog
        :open="showDeleteConfirmation"
        @update:open="(v) => (showDeleteConfirmation = v)"
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{{ t('organization.delete') }}</AlertDialogTitle>
            <AlertDialogDescription>{{ t('organization.deleteConfirm') }}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
            <AlertDialogAction variant="destructive" @click="confirmDelete">
              {{ t('globals.messages.delete') }}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  </ContactDetail>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { format } from 'date-fns'
import {
  Building2,
  CalendarIcon,
  IdCardIcon,
  MoreVerticalIcon,
  PencilIcon,
  Trash2Icon
} from 'lucide-vue-next'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import { Badge } from '@shared-ui/components/ui/badge'
import { Button } from '@shared-ui/components/ui/button'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Spinner } from '@shared-ui/components/ui/spinner'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator
} from '@shared-ui/components/ui/dropdown-menu'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import ContactDetail from '@/layouts/contact/ContactDetail.vue'
import OrganizationFormDialog from '@/features/organization/OrganizationFormDialog.vue'
import { organizationConversationFilters } from '@/features/organization/filters.js'
import { useUserStore } from '@/stores/user'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

const CONTACTS_PAGE_SIZE = 30
const CONVERSATIONS_PAGE_SIZE = 20

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const emitter = useEmitter()
const userStore = useUserStore()

const organization = ref(null)
const loading = ref(false)
const contacts = ref([])
const contactsPage = ref(1)
const hasMoreContacts = ref(false)
const loadingContacts = ref(false)
const conversations = ref([])
const conversationsPage = ref(1)
const conversationsTotal = ref(0)
const loadingConversations = ref(false)
const showEdit = ref(false)
const showDeleteConfirmation = ref(false)

const canRead = computed(() => userStore.can('contacts:read'))
const canWrite = computed(() => userStore.can('contacts:write'))
const canDelete = computed(() => userStore.can('contacts:delete'))

const breadcrumbLinks = computed(() => [
  { path: 'organizations', label: t('globals.terms.organization', 2) },
  { path: '', label: organization.value?.name || '' }
])

const initials = (first, last) => `${first?.[0] || ''}${last?.[0] || ''}`.toUpperCase()

const showError = (err) =>
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(err).message
  })

async function fetchOrganization() {
  loading.value = true
  try {
    const { data } = await api.getOrganization(route.params.id)
    organization.value = data.data
  } catch (err) {
    showError(err)
  } finally {
    loading.value = false
  }
}

async function fetchContacts(page = 1) {
  loadingContacts.value = true
  try {
    const { data } = await api.getOrganizationContacts(route.params.id, {
      page,
      page_size: CONTACTS_PAGE_SIZE
    })
    const rows = data.data || []
    contacts.value = page === 1 ? rows : [...contacts.value, ...rows]
    contactsPage.value = page
    hasMoreContacts.value = rows.length === CONTACTS_PAGE_SIZE
  } catch (err) {
    showError(err)
  } finally {
    loadingContacts.value = false
  }
}

async function fetchConversations(page = 1) {
  if (!userStore.can('conversations:read_all')) return
  loadingConversations.value = true
  try {
    const { data } = await api.getAllConversations({
      page,
      page_size: CONVERSATIONS_PAGE_SIZE,
      order_by: 'conversations.last_message_at',
      order: 'desc',
      filters: JSON.stringify(organizationConversationFilters(route.params.id))
    })
    const rows = data.data.results || []
    conversations.value = page === 1 ? rows : [...conversations.value, ...rows]
    conversationsPage.value = page
    conversationsTotal.value = data.data.total
  } catch (err) {
    showError(err)
  } finally {
    loadingConversations.value = false
  }
}

const onSaved = (org) => {
  organization.value = { ...organization.value, ...org }
}

async function confirmDelete() {
  showDeleteConfirmation.value = false
  try {
    await api.deleteOrganization(organization.value.id)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.deletedSuccessfully')
    })
    router.push({ name: 'organizations' })
  } catch (err) {
    showError(err)
  }
}

watch(
  [() => route.params.id, canRead],
  ([id, allowed]) => {
    if (!id || !allowed) return
    fetchOrganization()
    fetchContacts()
    fetchConversations()
  },
  { immediate: true }
)
</script>
