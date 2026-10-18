// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick, ref } from 'vue'

const { bulkDelete, bulkMarkSpam, stub } = vi.hoisted(() => ({
  bulkDelete: vi.fn(),
  bulkMarkSpam: vi.fn(),
  stub: { render: () => null }
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@/stores/conversation', () => ({
  useConversationStore: () => ({
    allSelected: false,
    selectedCount: 2,
    statusOptionsNoSnooze: [],
    clearSelection: vi.fn(),
    selectAll: vi.fn()
  })
}))
vi.mock('@/composables/useBulkActions', () => ({
  useBulkActions: () => ({
    bulkLoading: ref(false),
    bulkAssign: vi.fn(),
    bulkAddTag: vi.fn(),
    bulkUpdateStatus: vi.fn(),
    bulkDelete,
    bulkMarkSpam
  })
}))
vi.mock('@/composables/useBulkActionPermissions', () => ({
  useBulkActionPermissions: () => ({
    canAssignAgent: ref(false),
    canAssignTeam: ref(false),
    canUpdateStatus: ref(false),
    canUpdateTags: ref(false),
    canDelete: ref(true),
    canMarkSpam: ref(true)
  })
}))
vi.mock('@main/components/combobox/SelectAgentCombobox.vue', () => ({ default: stub }))
vi.mock('@main/components/combobox/SelectTeamCombobox.vue', () => ({ default: stub }))
vi.mock('@main/components/combobox/SelectTagCombobox.vue', () => ({ default: stub }))
import ConversationBulkActionToolbar from './ConversationBulkActionToolbar.vue'

let app
let root
const settle = async () => {
  for (let i = 0; i < 10; i++) await nextTick()
}
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.clearAllMocks()
})

const mount = async () => {
  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp(ConversationBulkActionToolbar)
  app.mount(root)
  await settle()
}

const click = async (el) => {
  el.click()
  await settle()
}

// The confirm button is the dialog's last button; the first is Cancel.
const confirmButton = () => {
  const buttons = document.querySelectorAll('[role="alertdialog"] button')
  return buttons[buttons.length - 1]
}

describe('ConversationBulkActionToolbar', () => {
  it('runs mark as spam, not delete, when the spam confirmation is accepted', async () => {
    await mount()
    await click(root.querySelector('[aria-label="conversation.bulkActions.markAsSpam"]'))
    expect(document.querySelector('[role="alertdialog"]').textContent).toContain(
      'conversation.bulkActions.markAsSpamConfirm'
    )
    await click(confirmButton())
    expect(bulkMarkSpam).toHaveBeenCalledTimes(1)
    expect(bulkDelete).not.toHaveBeenCalled()
  })

  it('runs delete when the delete confirmation is accepted', async () => {
    await mount()
    await click(root.querySelector('[aria-label="conversation.bulkActions.delete"]'))
    await click(confirmButton())
    expect(bulkDelete).toHaveBeenCalledTimes(1)
    expect(bulkMarkSpam).not.toHaveBeenCalled()
  })

  it('runs nothing when the confirmation is cancelled', async () => {
    await mount()
    await click(root.querySelector('[aria-label="conversation.bulkActions.delete"]'))
    await click(document.querySelector('[role="alertdialog"] button'))
    expect(bulkDelete).not.toHaveBeenCalled()
    expect(bulkMarkSpam).not.toHaveBeenCalled()
  })
})
