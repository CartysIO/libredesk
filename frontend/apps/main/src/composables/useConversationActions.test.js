// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { api, emit, conversationStore } = vi.hoisted(() => ({
  api: {
    getAssignedConversations: vi.fn(),
    updateConversationStatus: vi.fn(),
    updateAssignee: vi.fn(),
    removeAssignee: vi.fn(),
    updateConversationPriority: vi.fn(),
    deleteConversation: vi.fn(),
    markConversationSpam: vi.fn()
  },
  emit: vi.fn(),
  conversationStore: {
    getListSortParams: () => ({}),
    refreshSidebarCounts: vi.fn(),
    removeConversations: vi.fn(),
    conversations: { data: [] }
  }
}))
vi.mock('@main/api', () => ({ default: api }))
vi.mock('@main/websocket', () => ({ subscribeListReplace: vi.fn() }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit }) }))
vi.mock('@shared-ui/utils/http.js', () => ({ handleHTTPError: (e) => ({ message: e.message }) }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => ({ userID: 7 }) }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => conversationStore }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))

const { useConversationBoardStore } = await import('@main/stores/conversationBoard')
const { useConversationActions } = await import('./useConversationActions')

const card = (uuid, status, extra = {}) => ({ uuid, status, assigned_user_id: 7, ...extra })

// The "Assigned to me" board with the given cards.
const assignedBoard = async (cards) => {
  api.getAssignedConversations.mockImplementation((params) => {
    const status = JSON.parse(params.filters)[0].value
    const results = cards.filter((c) => c.status === status)
    return Promise.resolve({ data: { data: { results, total_pages: 1, total: results.length } } })
  })
  const store = useConversationBoardStore()
  await store.activate({ listType: 'assigned' })
  return store
}
const uuids = (store, status) => store.columns[status].items.map((c) => c.uuid)

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  for (const fn of Object.values(api)) fn.mockResolvedValue({})
})

describe('status', () => {
  it('moves the card once, and the websocket echo does not move it again', async () => {
    const store = await assignedBoard([card('a', 'Open'), card('b', 'Open')])
    const { setStatus } = useConversationActions()
    expect(await setStatus(store.columns.Open.items[0], 'Resolved')).toBe(true)
    expect(api.updateConversationStatus).toHaveBeenCalledWith('a', { status: 'Resolved' })
    store.mergeConversationUpdate({ uuid: 'a', status: 'Resolved' })
    expect(uuids(store, 'Open')).toEqual(['b'])
    expect(uuids(store, 'Resolved')).toEqual(['a'])
    expect(store.columns.Resolved.total).toBe(1)
  })

  it('sends the snooze time with Snoozed', async () => {
    const store = await assignedBoard([card('a', 'Open')])
    await useConversationActions().setStatus(store.columns.Open.items[0], 'Snoozed', '3h')
    expect(api.updateConversationStatus).toHaveBeenCalledWith('a', {
      status: 'Snoozed',
      snoozed_until: '3h'
    })
    expect(uuids(store, 'Snoozed')).toEqual(['a'])
  })
})

describe('assignment', () => {
  it('assigns an agent and updates the card', async () => {
    const store = await assignedBoard([card('a', 'Open')])
    const conv = store.columns.Open.items[0]
    expect(await useConversationActions().assignUser(conv, '7')).toBe(true)
    expect(api.updateAssignee).toHaveBeenCalledWith('a', 'user', { assignee_id: 7 })
    expect(conv.assigned_user_id).toBe(7)
    expect(conversationStore.refreshSidebarCounts).toHaveBeenCalled()
  })

  it('unassigning drops the card from the "Assigned to me" board', async () => {
    const store = await assignedBoard([card('a', 'Open')])
    await useConversationActions().assignUser(store.columns.Open.items[0], null)
    expect(api.removeAssignee).toHaveBeenCalledWith('a', 'user')
    expect(uuids(store, 'Open')).toEqual([])
  })

  it('a new team clears the agent, the same team keeps it', async () => {
    const store = await assignedBoard([card('a', 'Open', { assigned_team_id: 2 })])
    const conv = store.columns.Open.items[0]
    const { assignTeam } = useConversationActions()
    await assignTeam(conv, '2')
    expect(conv.assigned_user_id).toBe(7)
    await assignTeam(conv, '3')
    expect(api.updateAssignee).toHaveBeenLastCalledWith('a', 'team', { assignee_id: 3 })
    expect(conv.assigned_team_id).toBe(3)
    expect(uuids(store, 'Open')).toEqual([])
  })

  it('a refused request shows a toast and leaves the card alone', async () => {
    const store = await assignedBoard([card('a', 'Open')])
    api.removeAssignee.mockRejectedValue(new Error('forbidden'))
    expect(await useConversationActions().assignUser(store.columns.Open.items[0], null)).toBe(false)
    expect(uuids(store, 'Open')).toEqual(['a'])
    expect(emit).toHaveBeenCalledWith(
      'show-toast',
      expect.objectContaining({ variant: 'destructive', description: 'forbidden' })
    )
  })
})

it('sets the priority by name', async () => {
  const store = await assignedBoard([card('a', 'Open')])
  const conv = store.columns.Open.items[0]
  await useConversationActions().setPriority(conv, { label: 'High', value: 3 })
  expect(api.updateConversationPriority).toHaveBeenCalledWith('a', { priority: 'High' })
  expect([conv.priority, conv.priority_id]).toEqual(['High', 3])
})

describe('removal', () => {
  it('spam drops every conversation the server deleted', async () => {
    const store = await assignedBoard([card('a', 'Open'), card('b', 'Resolved')])
    api.markConversationSpam.mockResolvedValue({ data: { data: { deleted_uuids: ['a', 'b'] } } })
    expect(await useConversationActions().removeConversation('a', 'spam')).toEqual(['a', 'b'])
    expect(conversationStore.removeConversations).toHaveBeenCalledWith(['a', 'b'])
    expect([uuids(store, 'Open'), uuids(store, 'Resolved')]).toEqual([[], []])
  })

  it('delete returns null and keeps the card when the server refuses', async () => {
    const store = await assignedBoard([card('a', 'Open')])
    api.deleteConversation.mockRejectedValue(new Error('nope'))
    expect(await useConversationActions().removeConversation('a', 'delete')).toBeNull()
    expect(uuids(store, 'Open')).toEqual(['a'])
  })
})
