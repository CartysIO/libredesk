// @vitest-environment node
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const t = vi.fn((key) => key)

vi.mock('@main/i18n', () => ({ getI18n: () => ({ global: { t } }) }))
vi.mock('@main/api', () => ({ default: { deleteConversation: vi.fn() } }))
vi.mock('@/stores/conversation', () => ({
  useConversationStore: () => ({
    captureListSnapshots: vi.fn(() => []),
    removeConversations: vi.fn(),
    clearSelection: vi.fn(),
    fetchSidebarCounts: vi.fn(),
    refreshInboxSurface: vi.fn()
  })
}))
vi.mock('@/stores/conversationBoard', () => ({
  useConversationBoardStore: () => ({
    captureBoardSnapshots: vi.fn(() => []),
    removeConversations: vi.fn()
  })
}))

import { createBulkDeleteCommand } from './bulkCommands'

beforeEach(() => {
  setActivePinia(createPinia())
  t.mockClear()
})

describe('bulkCommands', () => {
  it('createBulkDeleteCommand reads t from i18n global (composition API)', () => {
    const command = createBulkDeleteCommand(['a', 'b'])
    expect(t).toHaveBeenCalledWith('conversation.bulkActions.deletedPending', { count: 2 })
    expect(command.toastDescription).toBe('conversation.bulkActions.deletedPending')
    expect(typeof command.apply).toBe('function')
  })
})