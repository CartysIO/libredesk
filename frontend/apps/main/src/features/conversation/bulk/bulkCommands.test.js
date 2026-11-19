// @vitest-environment node
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const t = vi.fn((key) => key)

vi.mock('@main/i18n', () => ({ getI18n: () => ({ global: { t } }) }))
vi.mock('@main/api', () => ({
  default: {
    deleteConversation: vi.fn(),
    markConversationSpam: vi.fn(),
    updateConversationStatus: vi.fn()
  }
}))
vi.mock('@/stores/conversation', () => ({
  useConversationStore: () => ({
    captureListSnapshots: vi.fn(() => []),
    removeConversations: vi.fn(),
    clearSelection: vi.fn(),
    fetchSidebarCounts: vi.fn(),
    refreshInboxSurface: vi.fn(),
    applyBulkStatusLocal: vi.fn(() => []),
    restoreListSnapshots: vi.fn()
  })
}))
vi.mock('@/stores/conversationBoard', () => ({
  useConversationBoardStore: () => ({
    captureBoardSnapshots: vi.fn(() => []),
    removeConversations: vi.fn(),
    applyBulkStatusBoard: vi.fn(),
    restoreBoardSnapshots: vi.fn()
  })
}))

import {
  createBulkDeleteCommand,
  createBulkSpamCommand,
  createBulkStatusCommand
} from './bulkCommands'

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

  it('createBulkSpamCommand builds a spam pending toast', () => {
    const command = createBulkSpamCommand(['x'])
    expect(t).toHaveBeenCalledWith('conversation.bulkActions.spamPending', { count: 1 })
    expect(command.toastDescription).toBe('conversation.bulkActions.spamPending')
  })

  it('createBulkStatusCommand builds a status pending toast', () => {
    const command = createBulkStatusCommand(['y'], 'Pending')
    expect(t).toHaveBeenCalledWith('conversation.bulkActions.statusPending', {
      count: 1,
      status: 'Pending'
    })
    expect(command.toastDescription).toBe('conversation.bulkActions.statusPending')
  })
})