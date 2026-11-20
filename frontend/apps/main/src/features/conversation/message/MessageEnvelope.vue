<template>
  <div class="desk-envelope text-desk-xs">
    <p v-if="message.meta.subject" class="desk-envelope__subject">
      {{ message.meta.subject }}
    </p>
    <div class="desk-envelope__routes">
      <p v-if="message.meta.from?.length">
        <span class="desk-envelope__label">From:</span> {{ message.meta.from.join(', ') }}
      </p>
      <p v-if="message.meta.to?.length">
        <span class="desk-envelope__label">To:</span> {{ message.meta.to.join(', ') }}
      </p>
      <p v-if="message.meta.cc?.length">
        <span class="desk-envelope__label">Cc:</span> {{ message.meta.cc.join(', ') }}
      </p>
      <p v-if="message.meta.bcc?.length">
        <span class="desk-envelope__label">Bcc:</span> {{ message.meta.bcc.join(', ') }}
      </p>
    </div>
  </div>
</template>

<script setup>
defineProps({
  message: {
    type: Object,
    required: true
  }
})
</script>
