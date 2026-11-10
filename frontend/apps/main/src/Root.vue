<template>
  <TooltipProvider :delay-duration="150">
    <Toaster
      class="desk-toaster pointer-events-auto"
      position="top-center"
      close-button
      :theme="mode === 'auto' ? 'system' : mode"
      :toast-options="toastOptions"
    />
    <RouterView />
  </TooltipProvider>
</template>

<script setup>
import { useColorMode } from '@vueuse/core'
import { RouterView } from 'vue-router'
import { Toaster } from '@shared-ui/components/ui/sonner'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'

// Clyr desk: light by default, not the device theme. An "auto" saved by an earlier build becomes light too;
// the dark-mode switch in the user menu still stores an explicit choice.
const mode = useColorMode({ initialValue: 'light' })
if (mode.store.value === 'auto') mode.value = 'light'

const toastOptions = {
  classes: {
    toast:
      'desk-toast group flex items-start gap-3 rounded-xl border border-border/80 bg-popover text-popover-foreground shadow-lg',
    title: 'font-display text-sm font-semibold tracking-tight',
    description: 'text-sm text-muted-foreground leading-snug',
    actionButton:
      'rounded-full border border-border/70 bg-primary text-primary-foreground text-xs font-semibold',
    cancelButton: 'rounded-full border border-border/70 bg-card text-foreground text-xs font-medium',
    closeButton: 'rounded-full border border-border/60 bg-card focus-visible:ring-2 focus-visible:ring-ring'
  }
}
</script>
