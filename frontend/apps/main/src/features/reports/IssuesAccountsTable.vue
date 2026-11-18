<template>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <thead>
        <tr class="border-b border-border/70 text-left text-muted-foreground">
          <th
            v-for="col in columns"
            :key="col.key"
            scope="col"
            class="py-2 pr-4 font-medium"
            :aria-sort="ariaSort(col.key)"
          >
            <Button
              type="button"
              variant="ghost"
              size="sm"
              class="group h-auto justify-start gap-1.5 px-0 py-1 text-left font-medium text-muted-foreground can-hover:hover:bg-transparent can-hover:hover:text-foreground"
              :class="{ 'text-foreground': sortBy === col.key }"
              @click="toggleSort(col.key)"
            >
              <span>{{ col.label }}</span>
              <ChevronUp
                v-if="sortBy === col.key && sortDir === 'asc'"
                class="h-3.5 w-3.5 shrink-0"
                aria-hidden="true"
              />
              <ChevronDown
                v-else-if="sortBy === col.key && sortDir === 'desc'"
                class="h-3.5 w-3.5 shrink-0"
                aria-hidden="true"
              />
              <ArrowUpDown
                v-else
                class="h-3.5 w-3.5 shrink-0 opacity-50 can-hover:opacity-0 can-hover:group-hover:opacity-50 group-focus-visible:opacity-50"
                aria-hidden="true"
              />
            </Button>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, i) in rows" :key="rowKey(row, i)" class="border-b border-border/70 last:border-0">
          <td v-for="col in columns" :key="col.key" class="py-2 pr-4 tabular-nums">
            {{ formatCell(row, col) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ArrowUpDown, ChevronDown, ChevronUp } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'

const props = defineProps({
  rows: { type: Array, required: true },
  columns: { type: Array, required: true },
  sortBy: { type: String, default: 'total' },
  sortDir: { type: String, default: 'desc' }
})

const emit = defineEmits(['update:sort'])

const defaultDirFor = (key) => (key === 'account_name' ? 'asc' : 'desc')

const toggleSort = (key) => {
  if (props.sortBy === key) {
    emit('update:sort', { by: key, dir: props.sortDir === 'desc' ? 'asc' : 'desc' })
    return
  }
  emit('update:sort', { by: key, dir: defaultDirFor(key) })
}

const ariaSort = (key) => {
  if (props.sortBy !== key) return 'none'
  return props.sortDir === 'asc' ? 'ascending' : 'descending'
}

const rowKey = (row, index) => {
  const id = row.organization_id
  if (id != null && id !== '') return `org-${id}`
  return `no-org-${row.account_name ?? ''}-${index}`
}

const formatCell = (row, col) => {
  const raw = row[col.key]
  if (col.format) return col.format(raw)
  return raw ?? ''
}
</script>