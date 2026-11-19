// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, createApp, defineComponent, h, ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import {
  confirmBulkToolbarAction,
  settleBulkToolbar
} from './conversationBulkActionToolbar.test-helpers'

const { api, toast, board } = vi.hoisted(() => ({
  api: { deleteConversation: vi.fn().mockResolvedValue({}) },
  toast: vi.fn(),
  board: {
    active: false,
    loadedCardUUIDs: [],
    removeConversations: vi.fn(),
    captureBoardSnapshots: vi.fn(() => []),
    refreshAll: vi.fn()
  }
}))

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@main/i18n', () => ({
  getI18n: () => ({ global: { t: (key, ...args) => (typeof args[0] === 'object' ? key : key) } })
}))
vi.mock('vue-sonner', () => ({ toast }))
vi.mock('@main/api', () => ({ default: api }))
vi.mock('vue-router', () => ({ useRouter: () => ({ currentRoute: ref({ params: {} }) }) }))
vi.mock('@/stores/user', () => ({ useUserStore: () => ({ can: () => true }) }))
vi.mock('@/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
vi.mock('@/stores/conversationBoard', () => ({
  useConversationBoardStore: () => ({
    context: { active: board.active },
    loadedCardUUIDs: board.loadedCardUUIDs,
    captureBoardSnapshots: board.captureBoardSnapshots,
    removeConversations: board.removeConversations,
    refreshAll: board.refreshAll
  })
}))
vi.mock('@main/components/combobox/SelectAgentCombobox.vue', () => ({
  default: { render: () => null }
}))
vi.mock('@main/components/combobox/SelectTeamCombobox.vue', () => ({
  default: { render: () => null }
}))
vi.mock('@main/components/combobox/SelectTagCombobox.vue', () => ({
  default: { render: () => null }
}))

import { useConversationStore } from '@/stores/conversation'
import {
  isBulkUndoPending,
  resetBulkCommandRunner,
  useBulkCommandPending
} from '@/features/conversation/bulk/bulkCommandRunner'
import ConversationBulkActionToolbar from './ConversationBulkActionToolbar.vue'

let app
let root
let pinia
const bulkCommandPending = useBulkCommandPending()

afterEach(async () => {
  app?.unmount()
  root?.remove()
  resetBulkCommandRunner()
  vi.clearAllMocks()
  board.active = false
  board.loadedCardUUIDs = []
})

beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
  api.deleteConversation.mockResolvedValue({})
})

const mountWithParentVIf = async () => {
  const store = useConversationStore()
  store.conversations.data = [
    { uuid: 'a', status: 'Open' },
    { uuid: 'b', status: 'Open' }
  ]
  store.conversations.total = 2
  store.selectedUUIDs = new Set(['a', 'b'])

  const Shell = defineComponent({
    name: 'BulkToolbarShell',
    setup () {
      const conversationStore = useConversationStore()
      const show = computed(() => conversationStore.selectedCount > 0)
      return () =>
        h('div', show.value ? h(ConversationBulkActionToolbar) : null)
    }
  })

  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp(Shell)
  app.use(pinia)
  app.mount(root)
  await settleBulkToolbar(12)
  return store
}

const openDeleteConfirm = async () => {
  const deleteBtn = root.querySelector('[aria-label="conversation.bulkActions.delete"]')
  deleteBtn.click()
  await settleBulkToolbar(12)
}

describe('ConversationBulkActionToolbar integration', () => {
  it('deletes selected list rows when confirm runs on pointerdown while parent v-if can unmount', async () => {
    const store = await mountWithParentVIf()
    await openDeleteConfirm()
    await confirmBulkToolbarAction()

    expect(store.conversations.data.map((c) => c.uuid)).toEqual([])
    expect(toast).toHaveBeenCalled()
    expect(isBulkUndoPending()).toBe(true)
    expect(bulkCommandPending.value).toBe(true)
  })

  it('removes board cards optimistically when board mode is active', async () => {
    board.active = true
    board.loadedCardUUIDs = ['a', 'b']
    await mountWithParentVIf()
    await openDeleteConfirm()
    await confirmBulkToolbarAction()

    expect(board.removeConversations).toHaveBeenCalledWith(['a', 'b'])
    expect(toast).toHaveBeenCalled()
  })
})