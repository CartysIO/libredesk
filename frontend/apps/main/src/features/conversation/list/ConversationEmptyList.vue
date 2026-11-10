<template>
  <div class="desk-empty-list flex h-auto min-h-48 flex-col items-center justify-center text-center select-none">
    <div
      class="mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-muted text-muted-foreground"
      aria-hidden="true"
    >
      <component :is="icon" :stroke-width="1.5" :size="24" />
    </div>
    <p class="font-display text-sm font-semibold tracking-tight text-foreground">
      {{ title }}
    </p>
    <p class="mt-2 max-w-[16rem] text-sm leading-relaxed text-muted-foreground">
      {{ message }}
    </p>
  </div>
</template>

<script setup>
defineProps({
  icon: {
    type: Function,
    required: true
  },
  title: {
    type: String,
    required: true
  },
  message: {
    type: String,
    required: true
  }
})
</script>
