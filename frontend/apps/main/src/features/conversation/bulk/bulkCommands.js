import api from '@/api'
import { useConversationStore } from '@/stores/conversation'
import { useConversationBoardStore } from '@/stores/conversationBoard'
import { getI18n } from '@main/i18n'

/**
 * @param {string[]} uuids
 * @returns {BulkCommand}
 */
export function createBulkDeleteCommand (uuids) {
  const conversationStore = useConversationStore()
  const boardStore = useConversationBoardStore()
  const { t } = getI18n()
  /** @type {ReturnType<typeof conversationStore.captureListSnapshots>} */
  let listSnapshots = []
  let boardSnapshots = []

  return {
    toastDescription: t('conversation.bulkActions.deletedPending', { count: uuids.length }),
    apply () {
      listSnapshots = conversationStore.captureListSnapshots(uuids)
      boardSnapshots = boardStore.captureBoardSnapshots(uuids)
      conversationStore.removeConversations(uuids)
      boardStore.removeConversations(uuids)
      conversationStore.clearSelection()
    },
    revert () {
      conversationStore.restoreListSnapshots(listSnapshots)
      boardStore.restoreBoardSnapshots(boardSnapshots)
      conversationStore.fetchSidebarCounts({ force: true })
    },
    async commit () {
      const results = await Promise.allSettled(uuids.map((uuid) => api.deleteConversation(uuid)))
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

/**
 * @param {string[]} uuids
 * @returns {BulkCommand}
 */
export function createBulkSpamCommand (uuids) {
  const conversationStore = useConversationStore()
  const boardStore = useConversationBoardStore()
  const { t } = getI18n()
  let listSnapshots = []
  let boardSnapshots = []
  const deletedUUIDs = new Set()

  return {
    toastDescription: t('conversation.bulkActions.spamPending', { count: uuids.length }),
    apply () {
      listSnapshots = conversationStore.captureListSnapshots(uuids)
      boardSnapshots = boardStore.captureBoardSnapshots(uuids)
      conversationStore.removeConversations(uuids)
      boardStore.removeConversations(uuids)
      conversationStore.clearSelection()
    },
    revert () {
      conversationStore.restoreListSnapshots(listSnapshots)
      boardStore.restoreBoardSnapshots(boardSnapshots)
      conversationStore.fetchSidebarCounts({ force: true })
    },
    async commit () {
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
      if (failed) {
        conversationStore.restoreListSnapshots(listSnapshots)
        boardStore.restoreBoardSnapshots(boardSnapshots)
      } else if (deletedUUIDs.size) {
        boardStore.removeConversations([...deletedUUIDs])
        conversationStore.removeConversations([...deletedUUIDs])
      }
      conversationStore.fetchSidebarCounts({ force: true })
      conversationStore.refreshInboxSurface()
    }
  }
}

/**
 * @param {string[]} uuids
 * @param {string} status
 * @returns {BulkCommand}
 */
export function createBulkStatusCommand (uuids, status) {
  const conversationStore = useConversationStore()
  const boardStore = useConversationBoardStore()
  const { t } = getI18n()
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
    }
  }
}

/** @typedef {import('./bulkCommandRunner.js').BulkCommand} BulkCommand */