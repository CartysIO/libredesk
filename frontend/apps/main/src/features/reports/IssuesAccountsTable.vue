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
          <td
            v-for="col in columns"
            :key="col.key"
            class="py-2 pr-4"
            :class="{ 'tabular-nums': col.numeric !== false }"
          >
            <router-link
              v-if="linkColumn && col.key === linkColumn.columnKey"
              :to="linkColumn.to(row)"
              class="font-medium text-primary underline-offset-4 can-hover:hover:underline tabular-nums"
            >
              {{ linkColumn.prefix ?? '' }}{{ row[col.key] }}
            </router-link>
            <span v-else :class="{ 'tabular-nums': col.numeric !== false && !col.plainText }">{{ formatCell(row, col) }}</span>
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
  sortDir: { type: String, default: 'desc' },
  /** When set, renders a router-link for this column (e.g. conversation reference). */
  linkColumn: { type: Object, default: null },
  /** Sort keys that default to ascending when newly selected. */
  sortAscKeys: { type: Array, default: () => ['account_name'] },
  /** Prefer this field for row keys when present (e.g. uuid). */
  rowIdKey: { type: String, default: '' }
})

const emit = defineEmits(['update:sort'])

const defaultDirFor = (key) => (props.sortAscKeys.includes(key) ? 'asc' : 'desc')

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
  if (props.rowIdKey && row[props.rowIdKey] != null && row[props.rowIdKey] !== '') {
    return String(row[props.rowIdKey])
  }
  const orgId = row.organization_id
  if (orgId != null && orgId !== '') return `org-${orgId}`
  const assigneeId = row.assigned_user_id
  if (assigneeId != null && assigneeId !== '') return `assignee-${assigneeId}`
  const label = row.account_name ?? row.assignee_name ?? ''
  return `row-${label}-${index}`
}

const formatCell = (row, col) => {
  const raw = row[col.key]
  if (col.format) return col.format(raw, row)
  if (raw == null || raw === '') return col.empty ?? '—'
  return raw
}
</script>