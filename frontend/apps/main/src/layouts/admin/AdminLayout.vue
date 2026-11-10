<template>
  <div class="overflow-y-auto overscroll-contain min-h-0 flex-1">
    <div class="desk-admin admin-content flex min-h-full flex-col px-3 py-4">
      <div
        class="flex flex-col items-center justify-center flex-grow"
        v-if="$route.name === 'admin'"
      >
        <div class="desk-admin-empty">{{ $t('admin.empty') }}</div>
      </div>
      <div class="flex-grow flex flex-col">
        <router-view />
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-content {
  container-type: inline-size;
}
</style>
