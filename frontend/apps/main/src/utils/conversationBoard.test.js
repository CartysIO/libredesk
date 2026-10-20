// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  CONVERSATION_BOARD_COLUMNS,
  CONVERSATION_BOARD_STATUS_LIST,
  CONVERSATION_LIST_TYPE
} from '@main/constants/conversation'
import {
  applyBoardMove,
  boardColumnForStatus,
  boardStatusFilter,
  conversationMatchesScope,
  createBoardColumns,
  findBoardCard,
  groupByBoardStatus,
  isBoardStatus,
  readBoardMode,
  undoBoardMove,
  writeBoardMode
} from './conversationBoard'

describe('board column mapping', () => {
  it('lists the columns in Pylon order with the agreed status names', () => {
    expect(CONVERSATION_BOARD_COLUMNS.map((c) => [c.key, c.status])).toEqual([
      ['new', 'Open'],
      ['onYou', 'On you'],
      ['onCustomer', 'On customer'],
      ['onHold', 'Snoozed'],
      ['resolved', 'Resolved']
    ])
    expect(CONVERSATION_BOARD_STATUS_LIST).toEqual(['Open', 'On you', 'On customer', 'Snoozed', 'Resolved'])
  })

  it('finds a column by status', () => {
    expect(boardColumnForStatus('Snoozed').key).toBe('onHold')
    expect(boardColumnForStatus('Escalated')).toBeNull()
    expect(isBoardStatus('On you')).toBe(true)
    expect(isBoardStatus('Escalated')).toBe(false)
  })

  it('builds the status name filter the list API expects', () => {
    expect(boardStatusFilter('On you')).toEqual([
      { model: 'conversation_statuses', field: 'name', operator: 'equals', value: 'On you' }
    ])
  })
})

const board = () => {
  const columns = createBoardColumns()
  columns.Open.items = [{ uuid: 'a', status: 'Open' }, { uuid: 'b', status: 'Open' }]
  columns.Open.total = 2
  columns.Resolved.items = [{ uuid: 'c', status: 'Resolved' }]
  columns.Resolved.total = 1
  return columns
}

describe('moving cards', () => {
  it('moves a card to the top of the target column and updates totals', () => {
    const columns = board()
    const move = applyBoardMove(columns, 'b', 'Resolved')
    expect(columns.Open.items.map((c) => c.uuid)).toEqual(['a'])
    expect(columns.Resolved.items.map((c) => c.uuid)).toEqual(['b', 'c'])
    expect([columns.Open.total, columns.Resolved.total]).toEqual([1, 2])
    expect(findBoardCard(columns, 'b').card.status).toBe('Resolved')
    expect(move.fromStatus).toBe('Open')
  })

  it('does nothing for the same column or an unknown card', () => {
    const columns = board()
    expect(applyBoardMove(columns, 'a', 'Open')).toBeNull()
    expect(applyBoardMove(columns, 'zzz', 'Resolved')).toBeNull()
  })

  it('undo puts the card back at its old position', () => {
    const columns = board()
    const move = applyBoardMove(columns, 'a', 'Snoozed')
    undoBoardMove(columns, move)
    expect(columns.Open.items.map((c) => c.uuid)).toEqual(['a', 'b'])
    expect(columns.Snoozed.items).toEqual([])
    expect([columns.Open.total, columns.Snoozed.total]).toEqual([2, 0])
    expect(findBoardCard(columns, 'a').card.status).toBe('Open')
  })

  it('undo does nothing when a live update moved the card meanwhile', () => {
    const columns = board()
    const move = applyBoardMove(columns, 'a', 'Snoozed')
    // A websocket update moved the card on to Resolved before the request failed.
    applyBoardMove(columns, 'a', 'Resolved')
    expect(undoBoardMove(columns, move)).toBe(false)
    expect(findBoardCard(columns, 'a').status).toBe('Resolved')
    expect(columns.Open.items.map((c) => c.uuid)).toEqual(['b'])
    expect(columns.Snoozed.items).toEqual([])
    expect([columns.Open.total, columns.Snoozed.total, columns.Resolved.total]).toEqual([1, 0, 2])
  })

  it('undo does nothing when the card was removed meanwhile, and never duplicates it', () => {
    const columns = board()
    const move = applyBoardMove(columns, 'a', 'Snoozed')
    columns.Snoozed.items.splice(0, 1)
    columns.Snoozed.total = 0
    expect(undoBoardMove(columns, move)).toBe(false)
    expect(findBoardCard(columns, 'a')).toBeNull()
    // The card moved back to Open by a live update: undo must not add a second copy.
    const again = board()
    const m2 = applyBoardMove(again, 'a', 'Snoozed')
    applyBoardMove(again, 'a', 'Open')
    expect(undoBoardMove(again, m2)).toBe(false)
    expect(again.Open.items.filter((c) => c.uuid === 'a')).toHaveLength(1)
  })

  it('drops a card moved to a status that has no column', () => {
    const columns = board()
    applyBoardMove(columns, 'a', 'Escalated')
    expect(findBoardCard(columns, 'a')).toBeNull()
    expect(columns.Open.total).toBe(1)
  })
})

describe('scope and grouping', () => {
  const ctx = { userID: 7, teamID: 3 }
  it('decides assignment scopes and leaves views to the server', () => {
    const T = CONVERSATION_LIST_TYPE
    expect(conversationMatchesScope(T.ASSIGNED, { assigned_user_id: 7 }, ctx)).toBe(true)
    expect(conversationMatchesScope(T.ASSIGNED, { assigned_user_id: 8 }, ctx)).toBe(false)
    expect(conversationMatchesScope(T.UNASSIGNED, {}, ctx)).toBe(true)
    expect(conversationMatchesScope(T.TEAM_UNASSIGNED, { assigned_team_id: 3 }, ctx)).toBe(true)
    expect(conversationMatchesScope(T.TEAM_UNASSIGNED, { assigned_team_id: 3, assigned_user_id: 1 }, ctx)).toBe(false)
    expect(conversationMatchesScope(T.ALL, {}, ctx)).toBe(true)
    expect(conversationMatchesScope(T.VIEW, {}, ctx)).toBeNull()
  })

  it('groups view rows by status and drops other statuses', () => {
    const groups = groupByBoardStatus([
      { uuid: '1', status: 'Open' },
      { uuid: '2', status: 'Escalated' },
      { uuid: '3', status: 'On you' }
    ])
    expect(groups.Open.map((c) => c.uuid)).toEqual(['1'])
    expect(groups['On you'].map((c) => c.uuid)).toEqual(['3'])
    expect(Object.keys(groups)).toEqual(CONVERSATION_BOARD_STATUS_LIST)
  })
})

describe('remembered board mode', () => {
  beforeEach(() => localStorage.clear())

  it('defaults to list and remembers the choice', () => {
    expect(readBoardMode()).toBe(false)
    writeBoardMode(true)
    expect(readBoardMode()).toBe(true)
    writeBoardMode(false)
    expect(readBoardMode()).toBe(false)
  })

  it('survives blocked storage', () => {
    const spy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    const setSpy = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    expect(readBoardMode()).toBe(false)
    expect(() => writeBoardMode(true)).not.toThrow()
    spy.mockRestore()
    setSpy.mockRestore()
  })
})

describe('board drawer', () => {
  const makeColumns = () => {
    const columns = createBoardColumns()
    columns[CONVERSATION_BOARD_STATUS_LIST[0]].items = [{ uuid: 'a' }, { uuid: 'b' }, { uuid: 'c' }]
    columns[CONVERSATION_BOARD_STATUS_LIST[1]].items = [{ uuid: 'd' }]
    return columns
  }

  it('finds the previous and next card in the same column', async () => {
    const { adjacentBoardCard } = await import('./conversationBoard')
    const columns = makeColumns()
    expect(adjacentBoardCard(columns, 'b', -1).uuid).toBe('a')
    expect(adjacentBoardCard(columns, 'b', 1).uuid).toBe('c')
  })

  it('returns null at the column ends and for unknown cards', async () => {
    const { adjacentBoardCard } = await import('./conversationBoard')
    const columns = makeColumns()
    expect(adjacentBoardCard(columns, 'a', -1)).toBeNull()
    expect(adjacentBoardCard(columns, 'c', 1)).toBeNull()
    expect(adjacentBoardCard(columns, 'd', 1)).toBeNull()
    expect(adjacentBoardCard(columns, 'zzz', 1)).toBeNull()
  })

  it('shows the board on conversation routes in board mode', async () => {
    const { shouldShowBoard, isConversationRoute } = await import('./conversationBoard')
    expect(shouldShowBoard({ isBoard: true, routeName: 'inbox-conversation' })).toBe(true)
    expect(shouldShowBoard({ isBoard: true, routeName: 'inbox' })).toBe(true)
    expect(shouldShowBoard({ isBoard: true, routeName: 'search' })).toBe(false)
    expect(shouldShowBoard({ isBoard: false, routeName: 'inbox-conversation' })).toBe(false)
    expect(isConversationRoute('view-inbox-conversation')).toBe(true)
    expect(isConversationRoute('view-inbox')).toBe(false)
  })

  it('clamps the drawer width', async () => {
    const { clampDrawerWidth } = await import('./conversationBoard')
    expect(clampDrawerWidth(2000, 1200)).toBe(1200)
    expect(clampDrawerWidth(100, 1200)).toBe(480)
    expect(clampDrawerWidth(900, 1200)).toBe(900)
    expect(clampDrawerWidth(900, 400)).toBe(400)
  })

  it('detects typing targets', async () => {
    const { isTypingTarget } = await import('./conversationBoard')
    expect(isTypingTarget({ tagName: 'INPUT' })).toBe(true)
    expect(isTypingTarget({ tagName: 'DIV', isContentEditable: true })).toBe(true)
    expect(isTypingTarget({ tagName: 'BUTTON' })).toBe(false)
  })
})
