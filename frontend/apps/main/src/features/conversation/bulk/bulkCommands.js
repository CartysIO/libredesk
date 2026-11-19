import api from '@/api'
import { useConversationStore } from '@/stores/conversation'
import { useConversationBoardStore } from '@/stores/conversationBoard'
import { getI18n } from '@main/i18n'
import { createBulkRemovalCommand } from './bulkRemovalCommand'

/**
 * @param {string[]} uuids
 * @returns {BulkCommand}
 */
export function createBulkDeleteCommand (uuids) {
  const t = getI18n().global.t
  return createBulkRemovalCommand(
    uuids,
    t('conversation.bulkActions.deletedPending', { count: uuids.length }),
    async () => {
      const results = await Promise.allSettled(uuids.map((uuid) => api.deleteConversation(uuid)))
      return results.some((r) => r.status === 'rejected')
    }
  )
}

/**
 * @param {string[]} uuids
 * @returns {BulkCommand}
 */
export function createBulkSpamCommand (uuids) {
  const conversationStore = useConversationStore()
  const boardStore = useConversationBoardStore()
  const t = getI18n().global.t
  const deletedUUIDs = new Set()

  return createBulkRemovalCommand(
    uuids,
    t('conversation.bulkActions.spamPending', { count: uuids.length }),
    async () => {
      const results = await Promise.allSettled(
        uuids.map(async (uuid) => {
          try {
            const response = await api.markConversationSpam(uuid)
            for (const id of response.data.data.deleted_uuids || []) deletedUUIDs.add(id)
          } catch (error) {
            if (error.response?.status !== 404) throw error
          }
        })
      )
      const failed = results.some((r) => r.status === 'rejected')
      if (!failed && deletedUUIDs.size) {
        boardStore.removeConversations([...deletedUUIDs])
        conversationStore.removeConversations([...deletedUUIDs])
      }
      return failed
    }
  )
}

/**
 * @param {string[]} uuids
 * @param {string} status
 * @returns {BulkCommand}
 */
export function createBulkStatusCommand (uuids, status) {
  const conversationStore = useConversationStore()
  const boardStore = useConversationBoardStore()
  const t = getI18n().global.t
  let listSnapshots = []
  let boardSnapshots = []

  return {
    toastDescription: t('conversation.bulkActions.statusPending', { count: uuids.length, status }),
    apply () {
      boardSnapshots = boardStore.captureBoardSnapshots(uuids)
      boardStore.applyBulkStatusBoard(uuids, status)
      listSnapshots = conversationStore.applyBulkStatusLocal(uuids, status)
      conversationStore.clearSelection()
    },
    revert () {
      conversationStore.restoreListSnapshots(listSnapshots)
      boardStore.restoreBoardSnapshots(boardSnapshots)
      conversationStore.fetchSidebarCounts({ force: true })
    },
    async commit () {
      const results = await Promise.allSettled(
        uuids.map((uuid) => api.updateConversationStatus(uuid, { status }))
      )
      const failed = results.some((r) => r.status === 'rejected')
      if (failed) {
        conversationStore.restoreListSnapshots(listSnapshots)
        boardStore.restoreBoardSnapshots(boardSnapshots)
      }
      conversationStore.fetchSidebarCounts({ force: true })
      conversationStore.refreshInboxSurface()
    }
  }
}

/** @typedef {import('./bulkCommandRunner.js').BulkCommand} BulkCommand */