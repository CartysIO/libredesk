import { useConversationStore } from '@/stores/conversation'
import { useConversationBoardStore } from '@/stores/conversationBoard'

/**
 * Shared optimistic remove + dual snapshots for delete/spam bulk commands.
 * @param {string[]} uuids
 * @param {string} toastDescription
 * @param {() => Promise<boolean>} commitImpl Resolves true when any request failed (triggers revert).
 */
export function createBulkRemovalCommand (uuids, toastDescription, commitImpl) {
  const conversationStore = useConversationStore()
  const boardStore = useConversationBoardStore()
  let listSnapshots = []
  let boardSnapshots = []

  const applyRemove = () => {
    listSnapshots = conversationStore.captureListSnapshots(uuids)
    boardSnapshots = boardStore.captureBoardSnapshots(uuids)
    conversationStore.removeConversations(uuids)
    boardStore.removeConversations(uuids)
    conversationStore.clearSelection()
  }

  const revertRemove = () => {
    conversationStore.restoreListSnapshots(listSnapshots)
    boardStore.restoreBoardSnapshots(boardSnapshots)
    conversationStore.fetchSidebarCounts({ force: true })
  }

  return {
    toastDescription,
    apply: applyRemove,
    revert: revertRemove,
    async commit () {
      const failed = await commitImpl()
      if (failed) revertRemove()
      conversationStore.fetchSidebarCounts({ force: true })
      conversationStore.refreshInboxSurface()
    }
  }
}