<template>
  <div class="inline-flex items-center rounded-md border p-0.5 shrink-0" role="group">
    <Button
      v-for="option in options"
      :key="option.value"
      type="button"
      variant="ghost"
      size="sm"
      class="h-7 px-2 gap-1"
      :class="{ 'bg-accent text-accent-foreground': (option.value === 'board') === isBoard }"
      :aria-pressed="(option.value === 'board') === isBoard"
      @click="setBoardMode(option.value === 'board')"
    >
      <component :is="option.icon" class="w-4 h-4" />
      <span class="max-md:hidden">{{ option.label }}</span>
    </Button>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { List, Columns3 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useBoardMode } from '@main/composables/useBoardMode'

const { t } = useI18n()
const { isBoard, setBoardMode } = useBoardMode()

const options = computed(() => [
  { value: 'list', label: t('kanban.list'), icon: List },
  { value: 'board', label: t('kanban.board'), icon: Columns3 }
])
</script>
