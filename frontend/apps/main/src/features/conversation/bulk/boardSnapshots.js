import { boardColumnForStatus, findBoardCard, applyBoardMove } from '@main/utils/conversationBoard'

const cloneCard = (card) => JSON.parse(JSON.stringify(card))

export function captureBoardSnapshots (columns, uuids) {
  return uuids
    .map((uuid) => {
      const found = findBoardCard(columns, uuid)
      if (!found) return null
      return {
        status: found.status,
        index: found.index,
        card: cloneCard(found.card)
      }
    })
    .filter(Boolean)
}

export function restoreBoardSnapshots (columns, snapshots) {
  if (!snapshots?.length) return
  for (const snap of snapshots) {
    const found = findBoardCard(columns, snap.card.uuid)
    if (found) {
      const col = columns[found.status]
      col.items.splice(found.index, 1)
      col.total = Math.max(0, col.total - 1)
    }
  }
  for (const snap of [...snapshots].sort((a, b) => a.index - b.index)) {
    const col = columns[snap.status]
    if (!col) continue
    const at = Math.min(snap.index, col.items.length)
    col.items.splice(at, 0, cloneCard(snap.card))
    col.total += 1
  }
}

export function applyBulkStatusOnBoard (columns, uuids, status) {
  const hasColumn = boardColumnForStatus(status) !== null
  for (const uuid of uuids) {
    const found = findBoardCard(columns, uuid)
    if (!found) continue
    if (hasColumn) {
      applyBoardMove(columns, uuid, status)
    } else {
      found.card.status = status
      const col = columns[found.status]
      col.items.splice(found.index, 1)
      col.total = Math.max(0, col.total - 1)
    }
  }
}