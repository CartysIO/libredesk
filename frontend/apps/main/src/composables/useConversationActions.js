import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useConversationStore } from '@main/stores/conversation'
import { useConversationBoardStore } from '@main/stores/conversationBoard'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import api from '@main/api'

// Actions on any conversation by uuid, not only the open one (the conversation store's actions act on the open one).
export function useConversationActions() {
  const conversationStore = useConversationStore()
  const boardStore = useConversationBoardStore()
  const emitter = useEmitter()
  const { t } = useI18n()

  const toastError = (error) =>
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })

  // Applies the change through the live-update paths the websocket uses, so its echo changes nothing
  // and a card that leaves the board's scope drops out. The conversation store merge also updates
  // the open drawer: the assignment echo reaches it only when the list holds the conversation.
  const update = async (uuid, request, fields) => {
    try {
      await request()
      boardStore.mergeConversationUpdate({ uuid, ...fields })
      conversationStore.mergeConversationUpdate({ uuid, ...fields })
      conversationStore.refreshSidebarCounts?.()
      return true
    } catch (error) {
      toastError(error)
      return false
    }
  }

  // Board cards only: moves the card optimistically and reverts when the server refuses.
  const setStatus = (conversation, status, snoozedUntil = '') =>
    boardStore.moveCard(conversation.uuid, status, snoozedUntil)

  const assignUser = (conversation, userID) => {
    const { uuid } = conversation
    if (!userID) {
      return update(uuid, () => api.removeAssignee(uuid, 'user'), { assigned_user_id: null })
    }
    const id = Number(userID)
    return update(uuid, () => api.updateAssignee(uuid, 'user', { assignee_id: id }), {
      assigned_user_id: id
    })
  }

  const assignTeam = (conversation, teamID) => {
    const { uuid } = conversation
    if (!teamID) {
      return update(uuid, () => api.removeAssignee(uuid, 'team'), { assigned_team_id: null })
    }
    const id = Number(teamID)
    // The server clears the agent when the team changes.
    const fields =
      Number(conversation.assigned_team_id) === id
        ? { assigned_team_id: id }
        : { assigned_team_id: id, assigned_user_id: null }
    return update(uuid, () => api.updateAssignee(uuid, 'team', { assignee_id: id }), fields)
  }

  const setPriority = (conversation, priority) =>
    update(
      conversation.uuid,
      () => api.updateConversationPriority(conversation.uuid, { priority: priority.label }),
      { priority: priority.label, priority_id: priority.value }
    )

  // action is 'delete' or 'spam'. Returns the deleted uuids, or null when the server refused.
  // Spam also deletes the contact's other unresolved conversations.
  const removeConversation = async (uuid, action) => {
    try {
      let deleted = [uuid]
      if (action === 'spam') {
        const response = await api.markConversationSpam(uuid)
        deleted = response.data.data.deleted_uuids
      } else {
        await api.deleteConversation(uuid)
      }
      conversationStore.removeConversations(deleted)
      boardStore.removeConversations(deleted)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t(action === 'spam' ? 'conversation.markedAsSpam' : 'conversation.deleted')
      })
      return deleted
    } catch (error) {
      toastError(error)
      return null
    }
  }

  return { setStatus, assignUser, assignTeam, setPriority, removeConversation }
}
