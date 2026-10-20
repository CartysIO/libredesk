<template>
  <TooltipProvider :delay-duration="150">
    <Toaster
      class="pointer-events-auto"
      position="top-center"
      close-button
      :theme="mode === 'auto' ? 'system' : mode"
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
</script>
