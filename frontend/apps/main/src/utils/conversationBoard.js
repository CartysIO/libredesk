import { CONVERSATION_BOARD_COLUMNS, CONVERSATION_LIST_TYPE } from '@main/constants/conversation'

export const BOARD_PAGE_SIZE = 25

export const boardColumnForStatus = (status) =>
  CONVERSATION_BOARD_COLUMNS.find((c) => c.status === status) || null

export const isBoardStatus = (status) => boardColumnForStatus(status) !== null

// Same filter the list sends for its status dropdown.
export const boardStatusFilter = (status) => [
  { model: 'conversation_statuses', field: 'name', operator: 'equals', value: status }
]

export const emptyBoardColumn = () => ({
  items: [],
  page: 0,
  hasMore: false,
  total: 0,
  loading: false,
  fetching: false,
  error: ''
})

export const createBoardColumns = () =>
  Object.fromEntries(CONVERSATION_BOARD_COLUMNS.map((c) => [c.status, emptyBoardColumn()]))

export const findBoardCard = (columns, uuid) => {
  for (const status of Object.keys(columns)) {
    const index = columns[status].items.findIndex((c) => c.uuid === uuid)
    if (index !== -1) return { status, index, card: columns[status].items[index] }
  }
  return null
}

// Moves a card between columns. Returns what undoBoardMove needs, or null when nothing moved.
export const applyBoardMove = (columns, uuid, toStatus) => {
  const found = findBoardCard(columns, uuid)
  if (!found || found.status === toStatus) return null
  const from = columns[found.status]
  from.items.splice(found.index, 1)
  from.total = Math.max(0, from.total - 1)
  found.card.status = toStatus
  const to = columns[toStatus]
  if (to) {
    to.items.unshift(found.card)
    to.total += 1
  }
  return { uuid, card: found.card, fromStatus: found.status, fromIndex: found.index, toStatus }
}

// Reverts applyBoardMove. Does nothing when the card is no longer in the column it was moved to
// (a live update moved or removed it meanwhile), so a rollback cannot duplicate it or restore a stale column.
export const undoBoardMove = (columns, move) => {
  if (!move) return false
  const found = findBoardCard(columns, move.uuid)
  if (!found || found.status !== move.toStatus) return false
  const to = columns[move.toStatus]
  to.items.splice(found.index, 1)
  to.total = Math.max(0, to.total - 1)
  move.card.status = move.fromStatus
  const from = columns[move.fromStatus]
  from.items.splice(Math.min(move.fromIndex, from.items.length), 0, move.card)
  from.total += 1
  return true
}

// true / false when the assignment scope decides, null when only the server knows (views, mentions).
export const conversationMatchesScope = (listType, conv, { userID, teamID }) => {
  switch (listType) {
    case CONVERSATION_LIST_TYPE.ASSIGNED:
      return conv.assigned_user_id === userID
    case CONVERSATION_LIST_TYPE.UNASSIGNED:
      return !conv.assigned_user_id && !conv.assigned_team_id
    case CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED:
      return Number(conv.assigned_team_id) === Number(teamID) && !conv.assigned_user_id
    case CONVERSATION_LIST_TYPE.ALL:
      return true
    default:
      return null
  }
}

// Splits view results into columns by status. Rows with a status that has no column are dropped.
export const groupByBoardStatus = (rows) => {
  const groups = Object.fromEntries(CONVERSATION_BOARD_COLUMNS.map((c) => [c.status, []]))
  for (const row of rows) {
    if (groups[row.status]) groups[row.status].push(row)
  }
  return groups
}

const STORAGE_KEY = 'conversationBoardMode'

export const readBoardMode = () => {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'board'
  } catch {
    return false
  }
}

export const writeBoardMode = (isBoard) => {
  try {
    localStorage.setItem(STORAGE_KEY, isBoard ? 'board' : 'list')
  } catch {
    // Storage may be blocked. The choice then lasts for this page only.
  }
}
