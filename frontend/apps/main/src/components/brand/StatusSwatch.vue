<template>
  <span
    class="code-swatch"
    :class="code.textClass"
    :data-hollow="code.hollow ? 'true' : undefined"
    role="img"
    :aria-label="status"
  />
</template>

<script setup>
import { computed } from 'vue'
import { statusCode } from '@main/utils/conversationCode'

const props = defineProps({
  status: { type: String, default: '' }
})

const code = computed(() => statusCode(props.status))
</script>
