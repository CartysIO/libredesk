// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { api, emit, subscribeListReplace } = vi.hoisted(() => ({
  api: {
    getAssignedConversations: vi.fn(),
    getAllConversations: vi.fn(),
    getViewConversations: vi.fn(),
    updateConversationStatus: vi.fn()
  },
  emit: vi.fn(),
  subscribeListReplace: vi.fn()
}))
vi.mock('@main/api', () => ({ default: api }))
vi.mock('@main/websocket', () => ({ subscribeListReplace }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit }) }))
vi.mock('@shared-ui/utils/http.js', () => ({ handleHTTPError: (e) => ({ message: e.message }) }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => ({ userID: 7 }) }))
vi.mock('@main/stores/conversation', () => ({
  useConversationStore: () => ({
    getListSortParams: () => ({ order_by: 'conversations.last_message_at', order: 'desc' }),
    refreshSidebarCounts: vi.fn(),
    conversations: { data: [] }
  })
}))

const { useConversationBoardStore } = await import('./conversationBoard')

const page = (results, { page = 1, total_pages = 1, total = results.length } = {}) => ({
  data: { data: { results, page, total_pages, total } }
})

// Answers each status filter with that status' rows.
const answerByStatus = (rowsByStatus, extra = {}) => (params) => {
  const status = JSON.parse(params.filters)[0].value
  return Promise.resolve(page(rowsByStatus[status] || [], extra))
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('board loading', () => {
  it('loads every column with its own status filter', async () => {
    api.getAssignedConversations.mockImplementation(
      answerByStatus({ Open: [{ uuid: 'a', status: 'Open' }], 'On you': [{ uuid: 'b', status: 'On you' }] })
    )
    const store = useConversationBoardStore()
    await store.activate({ listType: 'assigned' })
    expect(api.getAssignedConversations).toHaveBeenCalledTimes(5)
    const params = api.getAssignedConversations.mock.calls[0][0]
    expect(params).toMatchObject({ page: 1, page_size: 25, order: 'desc' })
    expect(store.columns.Open.items.map((c) => c.uuid)).toEqual(['a'])
    expect(store.columns['On you'].total).toBe(1)
    expect(store.columns.Resolved.items).toEqual([])
  })

  it('loads the next page of one column only', async () => {
    api.getAssignedConversations.mockImplementation((params) => {
      const status = JSON.parse(params.filters)[0].value
      if (status !== 'Open') return Promise.resolve(page([]))
      return Promise.resolve(
        page([{ uuid: `o${params.page}`, status: 'Open' }], { page: params.page, total_pages: 2, total: 2 })
      )
    })
    const store = useConversationBoardStore()
    await store.activate({ listType: 'assigned' })
    expect(store.columns.Open.hasMore).toBe(true)
    api.getAssignedConversations.mockClear()
    await store.fetchColumn('Open')
    expect(api.getAssignedConversations).toHaveBeenCalledTimes(1)
    expect(api.getAssignedConversations.mock.calls[0][0].page).toBe(2)
    expect(store.columns.Open.items.map((c) => c.uuid)).toEqual(['o1', 'o2'])
    expect(store.columns.Open.hasMore).toBe(false)
  })

  it('splits view pages into columns on the client', async () => {
    api.getViewConversations.mockResolvedValue(
      page([
        { uuid: 'a', status: 'Open' },
        { uuid: 'b', status: 'Snoozed' },
        { uuid: 'c', status: 'Closed' }
      ])
    )
    const store = useConversationBoardStore()
    await store.activate({ listType: 'view', viewID: 4 })
    expect(api.getViewConversations).toHaveBeenCalledWith(4, expect.not.objectContaining({ filters: expect.anything() }))
    expect(store.columns.Open.items).toHaveLength(1)
    expect(store.columns.Snoozed.items).toHaveLength(1)
    expect(store.columns.Resolved.items).toHaveLength(0)
  })

  it('starts over when the list context changes', async () => {
    api.getAssignedConversations.mockImplementation(answerByStatus({ Open: [{ uuid: 'a', status: 'Open' }] }))
    api.getAllConversations.mockImplementation(answerByStatus({}))
    const store = useConversationBoardStore()
    await store.activate({ listType: 'assigned' })
    await store.activate({ listType: 'all' })
    expect(store.columns.Open.items).toEqual([])
  })
})

describe('drag and drop moves', () => {
  const loaded = async () => {
    api.getAssignedConversations.mockImplementation(
      answerByStatus({ Open: [{ uuid: 'a', status: 'Open', assigned_user_id: 7 }] })
    )
    const store = useConversationBoardStore()
    await store.activate({ listType: 'assigned' })
    return store
  }

  it('moves optimistically and calls the status API', async () => {
    const store = await loaded()
    api.updateConversationStatus.mockResolvedValue({})
    const pending = store.moveCard('a', 'On customer')
    expect(store.columns.Open.items).toHaveLength(0)
    expect(store.columns['On customer'].items[0].uuid).toBe('a')
    expect(await pending).toBe(true)
    expect(api.updateConversationStatus).toHaveBeenCalledWith('a', { status: 'On customer' })
  })

  it('sends the snooze duration when dropped on Snoozed', async () => {
    const store = await loaded()
    api.updateConversationStatus.mockResolvedValue({})
    await store.moveCard('a', 'Snoozed', '2h')
    expect(api.updateConversationStatus).toHaveBeenCalledWith('a', { status: 'Snoozed', snoozed_until: '2h' })
  })

  it('reverts and shows a toast when the API fails', async () => {
    const store = await loaded()
    api.updateConversationStatus.mockRejectedValue(new Error('nope'))
    expect(await store.moveCard('a', 'Resolved')).toBe(false)
    expect(store.columns.Open.items.map((c) => c.uuid)).toEqual(['a'])
    expect(store.columns.Resolved.items).toEqual([])
    expect(store.columns.Open.total).toBe(1)
    expect(emit).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ variant: 'destructive', description: 'nope' }))
  })
})

describe('live updates', () => {
  const loaded = async () => {
    api.getAssignedConversations.mockImplementation(
      answerByStatus({ Open: [{ uuid: 'a', status: 'Open', assigned_user_id: 7, unread_message_count: 0, last_message_at: '2026-01-01T00:00:00Z' }] })
    )
    const store = useConversationBoardStore()
    await store.activate({ listType: 'assigned' })
    return store
  }

  it('moves a card when its status changes', async () => {
    const store = await loaded()
    store.mergeConversationUpdate({ uuid: 'a', status: 'On you' })
    expect(store.columns.Open.items).toHaveLength(0)
    expect(store.columns['On you'].items[0].uuid).toBe('a')
    expect(store.columns['On you'].total).toBe(1)
  })

  it('inserts a pushed conversation that belongs to the list', async () => {
    const store = await loaded()
    store.handleConvPush({ uuid: 'n', status: 'Open', assigned_user_id: 7 })
    store.handleConvPush({ uuid: 'x', status: 'Open', assigned_user_id: 9 })
    expect(store.columns.Open.items.map((c) => c.uuid)).toEqual(['n', 'a'])
  })

  it('drops a card that leaves the list scope', async () => {
    const store = await loaded()
    store.mergeConversationUpdate({ uuid: 'a', assigned_user_id: 9 })
    expect(store.columns.Open.items).toHaveLength(0)
  })

  it('counts unread messages and clears them on read', async () => {
    const store = await loaded()
    store.incrementUnread('a')
    expect(store.columns.Open.items[0].unread_message_count).toBe(1)
    store.applyConversationRead({ conversation_uuid: 'a', is_unread: false, last_seen_at: '2999-01-01T00:00:00Z' })
    expect(store.columns.Open.items[0].unread_message_count).toBe(0)
  })

  it('ignores updates while the board is hidden', async () => {
    const store = await loaded()
    store.deactivate()
    store.mergeConversationUpdate({ uuid: 'a', status: 'Resolved' })
    expect(store.columns.Open.items).toHaveLength(1)
  })
})
