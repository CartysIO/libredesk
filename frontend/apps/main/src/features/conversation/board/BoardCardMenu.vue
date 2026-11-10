<template>
  <ContextMenu>
    <ContextMenuTrigger as-child>
      <slot />
    </ContextMenuTrigger>
    <ContextMenuContent class="w-56">
      <ContextMenuItem @select="emit('action', 'open')">
        <PanelRight class="w-4 h-4 mr-2" />
        {{ t('globals.terms.open') }}
      </ContextMenuItem>
      <ContextMenuItem @select="emit('action', 'openFull')">
        <Maximize2 class="w-4 h-4 mr-2" />
        {{ t('kanban.openFullPage') }}
      </ContextMenuItem>
      <ContextMenuItem @select="emit('action', 'copyLink')">
        <Link2 class="w-4 h-4 mr-2" />
        {{ t('globals.messages.copyLink') }}
      </ContextMenuItem>

      <template v-if="canUpdateStatus || canUpdatePriority">
        <ContextMenuSeparator />
        <ContextMenuSub v-if="canUpdateStatus">
          <ContextMenuSubTrigger>
            <CircleDot class="w-4 h-4 mr-2" />
            {{ t('actions.setStatus') }}
          </ContextMenuSubTrigger>
          <ContextMenuSubContent class="max-h-80 overflow-y-auto">
            <template v-for="status in conversationStore.statusOptions" :key="status.value">
              <!-- Snoozing needs a time. -->
              <ContextMenuSub v-if="status.label === SNOOZED && conversation.status !== SNOOZED">
                <ContextMenuSubTrigger>
                  <Check class="w-4 h-4 mr-2 invisible" />
                  {{ statusLabel(status.label) }}
                </ContextMenuSubTrigger>
                <ContextMenuSubContent>
                  <ContextMenuItem
                    v-for="preset in SNOOZE_PRESETS"
                    :key="preset.minutes"
                    @select="
                      actions.setStatus(conversation, SNOOZED, formatSnoozeDuration(preset.minutes))
                    "
                  >
                    {{ preset.count }} {{ t(preset.unitKey, preset.count) }}
                  </ContextMenuItem>
                  <ContextMenuSeparator />
                  <ContextMenuItem @select="emit('action', 'snooze')">
                    <CalendarClock class="w-4 h-4 mr-2" />
                    {{ t('globals.messages.pickDateAndTime') }}
                  </ContextMenuItem>
                </ContextMenuSubContent>
              </ContextMenuSub>
              <ContextMenuItem
                v-else
                :disabled="status.label === conversation.status"
                @select="actions.setStatus(conversation, status.label)"
              >
                <Check
                  class="w-4 h-4 mr-2"
                  :class="{ invisible: status.label !== conversation.status }"
                />
                {{ statusLabel(status.label) }}
              </ContextMenuItem>
            </template>
          </ContextMenuSubContent>
        </ContextMenuSub>
        <ContextMenuSub v-if="canUpdatePriority">
          <ContextMenuSubTrigger>
            <Flag class="w-4 h-4 mr-2" />
            {{ t('actions.setPriority') }}
          </ContextMenuSubTrigger>
          <ContextMenuSubContent>
            <ContextMenuItem
              v-for="priority in conversationStore.priorityOptions"
              :key="priority.value"
              :disabled="priority.label === conversation.priority"
              @select="actions.setPriority(conversation, priority)"
            >
              <Check
                class="w-4 h-4 mr-2"
                :class="{ invisible: priority.label !== conversation.priority }"
              />
              {{ priority.label }}
            </ContextMenuItem>
          </ContextMenuSubContent>
        </ContextMenuSub>
      </template>

      <template v-if="canAssignAgent || canAssignTeam">
        <ContextMenuSeparator />
        <template v-if="canAssignAgent">
          <ContextMenuItem
            v-if="conversation.assigned_user_id !== userStore.userID"
            @select="actions.assignUser(conversation, userStore.userID)"
          >
            <UserCheck class="w-4 h-4 mr-2" />
            {{ t('conversation.assignSelfAction') }}
          </ContextMenuItem>
          <ContextMenuSub>
            <ContextMenuSubTrigger>
              <UserPlus class="w-4 h-4 mr-2" />
              {{ t('actions.assignAgent') }}
            </ContextMenuSubTrigger>
            <ContextMenuSubContent class="max-h-80 overflow-y-auto">
              <ContextMenuItem
                v-for="agent in usersStore.options"
                :key="agent.value"
                :disabled="agent.value === String(conversation.assigned_user_id)"
                @select="actions.assignUser(conversation, agent.value)"
              >
                <Check
                  class="w-4 h-4 mr-2"
                  :class="{ invisible: agent.value !== String(conversation.assigned_user_id) }"
                />
                {{ agent.label }}
              </ContextMenuItem>
            </ContextMenuSubContent>
          </ContextMenuSub>
        </template>
        <ContextMenuSub v-if="canAssignTeam">
          <ContextMenuSubTrigger>
            <Users class="w-4 h-4 mr-2" />
            {{ t('actions.assignTeam') }}
          </ContextMenuSubTrigger>
          <ContextMenuSubContent class="max-h-80 overflow-y-auto">
            <ContextMenuItem
              v-for="team in teamStore.options"
              :key="team.value"
              :disabled="team.value === String(conversation.assigned_team_id)"
              @select="actions.assignTeam(conversation, team.value)"
            >
              <Check
                class="w-4 h-4 mr-2"
                :class="{ invisible: team.value !== String(conversation.assigned_team_id) }"
              />
              {{ team.emoji ? `${team.emoji} ${team.label}` : team.label }}
            </ContextMenuItem>
            <template v-if="conversation.assigned_team_id">
              <ContextMenuSeparator />
              <ContextMenuItem @select="actions.assignTeam(conversation, null)">
                <UserX class="w-4 h-4 mr-2" />
                {{ t('command.unassignTeam') }}
              </ContextMenuItem>
            </template>
          </ContextMenuSubContent>
        </ContextMenuSub>
        <ContextMenuItem
          v-if="canAssignAgent && conversation.assigned_user_id"
          @select="actions.assignUser(conversation, null)"
        >
          <UserX class="w-4 h-4 mr-2" />
          {{ t('command.unassignAgent') }}
        </ContextMenuItem>
      </template>

      <template v-if="canDelete">
        <ContextMenuSeparator />
        <ContextMenuItem v-if="canMarkSpam" @select="emit('action', 'spam')">
          <ShieldBan class="w-4 h-4 mr-2" />
          {{ t('conversation.markAsSpam') }}
        </ContextMenuItem>
        <ContextMenuItem class="text-destructive" @select="emit('action', 'delete')">
          <Trash2 class="w-4 h-4 mr-2" />
          {{ t('conversation.deleteConversation') }}
        </ContextMenuItem>
      </template>
    </ContextMenuContent>
  </ContextMenu>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CalendarClock,
  Check,
  CircleDot,
  Flag,
  Link2,
  Maximize2,
  PanelRight,
  ShieldBan,
  Trash2,
  UserCheck,
  UserPlus,
  UserX,
  Users
} from 'lucide-vue-next'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuSub,
  ContextMenuSubContent,
  ContextMenuSubTrigger,
  ContextMenuTrigger
} from '@shared-ui/components/ui/context-menu'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useUsersStore } from '@main/stores/users'
import { useTeamStore } from '@main/stores/team'
import { useBulkActionPermissions } from '@main/composables/useBulkActionPermissions'
import { useConversationActions } from '@main/composables/useConversationActions'
import { permissions as perms } from '@main/constants/permissions'
import { CONVERSATION_DEFAULT_STATUSES } from '@main/constants/conversation'
import {
  SNOOZE_PRESETS,
  formatSnoozeDuration
} from '@main/features/command/providers/useConversationCommands'
import { boardColumnForStatus } from '@main/utils/conversationBoard'

// Right-click menu for one board card. Actions that need the board (navigation, dialogs) are emitted.
defineProps({
  conversation: { type: Object, required: true }
})
const emit = defineEmits(['action'])

const SNOOZED = CONVERSATION_DEFAULT_STATUSES.SNOOZED

const { t } = useI18n()
const conversationStore = useConversationStore()
const userStore = useUserStore()
const usersStore = useUsersStore()
const teamStore = useTeamStore()
const actions = useConversationActions()
const { canAssignAgent, canAssignTeam, canUpdateStatus, canDelete, canMarkSpam } =
  useBulkActionPermissions()
const canUpdatePriority = computed(() => userStore.can(perms.CONVERSATIONS_UPDATE_PRIORITY))

// Statuses with a column show the column's name, as the board does.
const statusLabel = (status) => {
  const column = boardColumnForStatus(status)
  return column ? t(column.labelKey) : status
}
</script>
