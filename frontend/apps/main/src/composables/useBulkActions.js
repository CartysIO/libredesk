import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConversationStore } from '@/stores/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { TAG_ACTION } from '@/constants/conversation'
import api from '@/api'
import { runDeferredBulkCommand } from '@/features/conversation/bulk/bulkCommandRunner'
import {
  createBulkDeleteCommand,
  createBulkSpamCommand,
  createBulkStatusCommand
} from '@/features/conversation/bulk/bulkCommands'

const bulkLoading = ref(false)

export function useBulkActions () {
  const conversationStore = useConversationStore()
  const emitter = useEmitter()
  const { t } = useI18n()

  const selectedUUIDs = () => [...conversationStore.selectedUUIDs]

  const runBulkAction = async (actionFn) => {
    const uuids = selectedUUIDs()
    bulkLoading.value = true
    const results = await Promise.allSettled(uuids.map((uuid) => actionFn(uuid)))
    bulkLoading.value = false

    const hasFailures = results.some((r) => r.status === 'rejected')

    conversationStore.clearSelection()
    conversationStore.refreshInboxSurface()
    conversationStore.fetchSidebarCounts({ force: true })

    if (hasFailures) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        title: t('globals.terms.error', 1),
        description: t('conversation.bulkActions.failedToast')
      })
    } else {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('conversation.bulkActions.successToast')
      })
    }
  }

  const runDeferred = (buildCommand) => {
    const uuids = selectedUUIDs()
    if (!uuids.length) return
    const started = runDeferredBulkCommand(buildCommand(uuids), {
      undoLabel: t('globals.messages.undo'),
      onBlocked: () => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'warning',
          description: t('conversation.bulkActions.pendingUndo')
        })
      }
    })
    if (!started) return
  }

  const bulkAssign = (assigneeType, assigneeValue) => {
    if (assigneeValue === 'none') {
      return runBulkAction((uuid) => api.removeAssignee(uuid, assigneeType))
    }
    const assigneeId = parseInt(assigneeValue, 10)
    return runBulkAction((uuid) =>
      api.updateAssignee(uuid, assigneeType, { assignee_id: assigneeId })
    )
  }

  const bulkAddTag = (tag) =>
    runBulkAction((uuid) => conversationStore.updateConversationTags(uuid, TAG_ACTION.ADD, [tag]))

  const bulkUpdateStatus = (status) => runDeferred((uuids) => createBulkStatusCommand(uuids, status))

  const bulkDelete = () => runDeferred((uuids) => createBulkDeleteCommand(uuids))

  const bulkMarkSpam = () => runDeferred((uuids) => createBulkSpamCommand(uuids))

  return {
    bulkLoading,
    runBulkAction,
    bulkAssign,
    bulkAddTag,
    bulkUpdateStatus,
    bulkDelete,
    bulkMarkSpam
  }
}