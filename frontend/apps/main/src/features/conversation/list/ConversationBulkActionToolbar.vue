<template>
  <div
    role="toolbar"
    :aria-label="t('conversation.bulkActions.toolbar')"
    class="desk-bulk-toolbar desk-bulk-toolbar-floating flex flex-wrap items-center gap-1.5 px-3 py-2.5 shadow-lg"
  >
    <Checkbox
      :checked="conversationStore.allSelected"
      @update:checked="toggleSelectAll"
      :aria-label="t('conversation.bulkActions.selectAll')"
      class="ml-1 mr-1"
    />
    <div class="flex min-w-0 flex-col gap-0.5 pr-1 mr-0.5 border-r border-border/60">
      <span
        class="text-sm font-semibold whitespace-nowrap tabular-nums"
        aria-live="polite"
      >
        {{ t('conversation.bulkActions.selected', conversationStore.selectedCount, { count: conversationStore.selectedCount }) }}
      </span>
      <span class="text-[11px] text-muted-foreground whitespace-nowrap tabular-nums">
        {{
          isBoardActive
            ? t('conversation.bulkActions.selectedOfBoard', {
                selected: conversationStore.selectedCount,
                total: boardLoadedCount
              })
            : t('conversation.bulkActions.selectedOfTotal', {
                selected: conversationStore.selectedCount,
                total: listLoadedCount
              })
        }}
      </span>
    </div>

    <!-- Assign Agent -->
    <SelectAgentCombobox
      v-if="canAssignAgent"
      include-none
      align="start"
      @select="(item) => onAssigneeSelect('user', item)"
    >
      <template #trigger>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="t('actions.assignAgent')"
          :aria-label="t('actions.assignAgent')"
        >
          <UserPlus class="w-4 h-4" />
        </Button>
      </template>
    </SelectAgentCombobox>

    <!-- Assign Team -->
    <SelectTeamCombobox
      v-if="canAssignTeam"
      include-none
      align="start"
      @select="(item) => onAssigneeSelect('team', item)"
    >
      <template #trigger>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="t('actions.assignTeam')"
          :aria-label="t('actions.assignTeam')"
        >
          <Users class="w-4 h-4" />
        </Button>
      </template>
    </SelectTeamCombobox>

    <!-- Add Tag -->
    <SelectTagCombobox
      v-if="canUpdateTags"
      align="start"
      @select="onTagSelect"
    >
      <template #trigger>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="t('actions.addTags')"
          :aria-label="t('actions.addTags')"
        >
          <Tag class="w-4 h-4" />
        </Button>
      </template>
    </SelectTagCombobox>

    <!-- Set Status -->
    <DropdownMenu v-if="canUpdateStatus">
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="t('actions.setStatus')"
          :aria-label="t('actions.setStatus')"
        >
          <CircleDot class="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        <DropdownMenuItem
          v-for="status in conversationStore.statusOptionsNoSnooze"
          :key="status.value"
          @click="bulkUpdateStatus(status.label)"
        >
          {{ status.label }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>

    <!-- Mark as spam -->
    <Button
      v-if="canMarkSpam"
      variant="ghost"
      size="icon"
      :disabled="bulkLoading"
      :title="t('conversation.bulkActions.markAsSpam')"
      :aria-label="t('conversation.bulkActions.markAsSpam')"
      @click="openConfirm('spam')"
    >
      <ShieldBan class="w-4 h-4" />
    </Button>

    <!-- Delete -->
    <Button
      v-if="canDelete"
      variant="ghost"
      size="icon"
      :disabled="bulkLoading"
      :title="t('conversation.bulkActions.delete')"
      :aria-label="t('conversation.bulkActions.delete')"
      @click="openConfirm('delete')"
    >
      <Trash2 class="w-4 h-4" />
    </Button>

    <Loader2 v-if="bulkLoading" class="w-4 h-4 animate-spin text-muted-foreground ml-2" />

    <Button
      variant="ghost"
      size="icon"
      class="ml-auto"
      :aria-label="t('conversation.bulkActions.clearSelection')"
      @click="conversationStore.clearSelection()"
    >
      <X class="w-4 h-4" />
    </Button>

    <AlertDialog :open="confirmOpen" @update:open="(v) => (confirmOpen = v)">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {{
              t(
                confirmAction === 'spam'
                  ? 'conversation.bulkActions.markAsSpam'
                  : 'conversation.bulkActions.delete'
              )
            }}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {{
              t(
                confirmAction === 'spam'
                  ? 'conversation.bulkActions.markAsSpamConfirm'
                  : 'conversation.bulkActions.deleteConfirm',
                { count: conversationStore.selectedCount }
              )
            }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            @pointerdown.prevent.stop="runConfirmedAction"
            @keydown.enter.prevent.stop="runConfirmedAction"
            @keydown.space.prevent.stop="runConfirmedAction"
          >
            {{
              t(
                confirmAction === 'spam'
                  ? 'conversation.bulkActions.markAsSpam'
                  : 'globals.messages.delete'
              )
            }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { UserPlus, Users, Tag, CircleDot, Loader2, X, ShieldBan, Trash2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import SelectTagCombobox from '@main/components/combobox/SelectTagCombobox.vue'
import { useConversationStore } from '@/stores/conversation'
import { useConversationBoardStore } from '@/stores/conversationBoard'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import { useBulkActions } from '@/composables/useBulkActions'

const conversationStore = useConversationStore()
const boardStore = useConversationBoardStore()
const isBoardActive = computed(() => boardStore.context.active)
const boardLoadedCount = computed(() => boardStore.loadedCardUUIDs.length)
const listLoadedCount = computed(() => conversationStore.conversationsList.length)
const { t } = useI18n()
const { bulkLoading, bulkAssign, bulkAddTag, bulkUpdateStatus, bulkDelete, bulkMarkSpam } =
  useBulkActions()

const { canAssignAgent, canAssignTeam, canUpdateStatus, canUpdateTags, canDelete, canMarkSpam } =
  useBulkActionPermissions()

// The action the open confirmation runs: 'delete' or 'spam'. Closing the dialog keeps it,
// because the dialog closes before its action button's click handler runs.
const confirmAction = ref('delete')
const confirmOpen = ref(false)

const openConfirm = (action) => {
  confirmAction.value = action
  confirmOpen.value = true
}

// Radix closes the dialog on pointerdown, which can unmount the action before a click
// fires — and clearing selection can tear down this toolbar mid-sequence. Run on pointerdown.
let confirmHandling = false
const runConfirmedAction = () => {
  if (confirmHandling) return
  confirmHandling = true
  try {
    const action = confirmAction.value
    if (action === 'spam') bulkMarkSpam()
    else bulkDelete()
    confirmOpen.value = false
  } finally {
    queueMicrotask(() => {
      confirmHandling = false
    })
  }
}

const toggleSelectAll = () => {
  if (conversationStore.allSelected) {
    conversationStore.clearSelection()
  } else {
    conversationStore.selectAll()
  }
}

const onAssigneeSelect = (assigneeType, item) => bulkAssign(assigneeType, item.value)

const onTagSelect = (item) => bulkAddTag(item.value)
</script>
