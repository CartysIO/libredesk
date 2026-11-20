<template>
  <div class="desk-activity" role="status">
    <span class="desk-activity__text">{{ message.content }}</span>
    <Tooltip>
      <TooltipTrigger as-child>
        <time class="desk-activity__time" :datetime="message.updated_at">
          {{ format(message.updated_at, 'h:mm a') }}
        </time>
      </TooltipTrigger>
      <TooltipContent>
        <p>
          {{ format(message.updated_at, "MMMM dd, yyyy 'at' HH:mm") }}
        </p>
      </TooltipContent>
    </Tooltip>
  </div>
</template>

<script setup>
import { format } from 'date-fns'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'

defineProps({
  message: Object
})
</script>
