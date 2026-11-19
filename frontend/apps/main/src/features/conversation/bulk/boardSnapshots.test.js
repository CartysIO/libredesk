// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { createBoardColumns } from '@main/utils/conversationBoard'
import { CONVERSATION_BOARD_STATUSES, CONVERSATION_DEFAULT_STATUSES } from '@main/constants/conversation'
import { applyBulkStatusOnBoard, captureBoardSnapshots, restoreBoardSnapshots } from './boardSnapshots'

describe('boardSnapshots', () => {
  it('restores a card to its column and index', () => {
    const columns = createBoardColumns()
    columns[CONVERSATION_DEFAULT_STATUSES.OPEN].items = [
      { uuid: 'a', status: CONVERSATION_DEFAULT_STATUSES.OPEN },
      { uuid: 'b', status: CONVERSATION_DEFAULT_STATUSES.OPEN }
    ]
    columns[CONVERSATION_DEFAULT_STATUSES.OPEN].total = 2

    const snapshots = captureBoardSnapshots(columns, ['b'])
    columns[CONVERSATION_DEFAULT_STATUSES.OPEN].items.splice(1, 1)
    columns[CONVERSATION_DEFAULT_STATUSES.OPEN].total = 1

    restoreBoardSnapshots(columns, snapshots)
    expect(columns[CONVERSATION_DEFAULT_STATUSES.OPEN].items.map((c) => c.uuid)).toEqual(['a', 'b'])
    expect(columns[CONVERSATION_DEFAULT_STATUSES.OPEN].total).toBe(2)
  })

  it('moves cards across board columns on bulk status', () => {
    const columns = createBoardColumns()
    columns[CONVERSATION_DEFAULT_STATUSES.OPEN].items = [
      { uuid: 'a', status: CONVERSATION_DEFAULT_STATUSES.OPEN }
    ]
    columns[CONVERSATION_DEFAULT_STATUSES.OPEN].total = 1

    applyBulkStatusOnBoard(columns, ['a'], CONVERSATION_BOARD_STATUSES.ON_CUSTOMER)
    expect(columns[CONVERSATION_DEFAULT_STATUSES.OPEN].items).toHaveLength(0)
    expect(columns[CONVERSATION_BOARD_STATUSES.ON_CUSTOMER].items.map((c) => c.uuid)).toEqual(['a'])
  })
})