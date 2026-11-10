<template>
  <div class="desk-auth-shell min-h-dvh overflow-x-hidden overflow-y-auto overscroll-contain">
    <div class="flex min-h-full flex-col">
      <main class="flex flex-1 items-start justify-center px-4 pb-10 pt-[12vh] sm:px-6">
        <div class="auth-content w-full max-w-md overflow-visible">
          <slot />
        </div>
      </main>
      <footer class="flex justify-center px-4 pb-6">
        <PoweredBy />
      </footer>
    </div>
  </div>
</template>

<script setup>
import PoweredBy from '@shared-ui/components/PoweredBy.vue'
</script>

<style scoped>
.auth-content > :deep(.desk-auth-card) {
  border-radius: var(--radius);
  overflow: visible;
}
</style>
