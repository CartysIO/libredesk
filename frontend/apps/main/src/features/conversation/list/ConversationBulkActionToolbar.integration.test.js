// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, createApp, defineComponent, h, nextTick, ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'

const { api, toast } = vi.hoisted(() => ({
  api: { deleteConversation: vi.fn().mockResolvedValue({}) },
  toast: vi.fn()
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
    context: { active: false },
    loadedCardUUIDs: [],
    captureBoardSnapshots: () => [],
    removeConversations: vi.fn(),
    refreshAll: vi.fn()
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
import { bulkCommandPending } from '@/features/conversation/bulk/bulkCommandRunner'
import ConversationBulkActionToolbar from './ConversationBulkActionToolbar.vue'

const stub = { render: () => null }

const settle = async () => {
  for (let i = 0; i < 12; i++) await nextTick()
}

let app
let root

afterEach(async () => {
  app?.unmount()
  root?.remove()
  bulkCommandPending.value = false
  vi.clearAllMocks()
})

let pinia

beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
  api.deleteConversation.mockResolvedValue({})
})

const confirmButton = () => {
  const buttons = document.querySelectorAll('[role="alertdialog"] button')
  return buttons[buttons.length - 1]
}

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
      const show = computed(
        () => conversationStore.selectedCount > 0 || bulkCommandPending.value
      )
      return () =>
        h('div', show.value ? h(ConversationBulkActionToolbar) : null)
    }
  })

  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp(Shell)
  app.use(pinia)
  app.mount(root)
  await settle()
  return store
}

describe('ConversationBulkActionToolbar integration', () => {
  it('deletes selected rows when confirm runs on pointerdown while parent v-if can unmount', async () => {
    const store = await mountWithParentVIf()
    const deleteBtn = root.querySelector('[aria-label="conversation.bulkActions.delete"]')
    deleteBtn.click()
    await settle()

    const btn = confirmButton()
    btn.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, cancelable: true }))
    await settle()

    expect(store.conversations.data.map((c) => c.uuid)).toEqual([])
    expect(toast).toHaveBeenCalled()
    expect(bulkCommandPending.value).toBe(true)
  })
})