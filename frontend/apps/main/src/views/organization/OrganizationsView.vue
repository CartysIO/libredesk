<template>
  <ContactList>
    <OrganizationsList v-if="userStore.can('contacts:read')" />
  </ContactList>
</template>
<script setup>
import OrganizationsList from '@/features/organization/OrganizationsList.vue'
import ContactList from '@/layouts/contact/ContactList.vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
</script>
