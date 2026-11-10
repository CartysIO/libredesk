<template>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <thead>
        <tr class="border-b border-border/70 text-left text-muted-foreground">
          <th v-for="col in columns" :key="col.key" class="py-2 pr-4 font-medium">
            {{ col.label }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, i) in rows" :key="i" class="border-b border-border/70 last:border-0">
          <td v-for="col in columns" :key="col.key" class="py-2 pr-4 tabular-nums">
            {{ formatCell(row, col) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
defineProps({
  rows: { type: Array, required: true },
  columns: { type: Array, required: true }
})

const formatCell = (row, col) => {
  const raw = row[col.key]
  if (col.format) return col.format(raw)
  return raw ?? ''
}
</script>
