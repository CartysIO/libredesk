import { cloneSnapshotRow } from './snapshotClone'

export function captureListSnapshots (data, uuids) {
  const list = data || []
  return uuids
    .map((uuid) => {
      const index = list.findIndex((c) => c.uuid === uuid)
      if (index === -1) return null
      return { index, conversation: cloneSnapshotRow(list[index]) }
    })
    .filter(Boolean)
}

export function restoreListSnapshots (data, totalRef, snapshots) {
  if (!snapshots?.length) return { data, added: 0 }
  const rows = [...(data || [])]
  let added = 0
  for (const { index, conversation } of [...snapshots].sort((a, b) => a.index - b.index)) {
    const cur = rows.findIndex((c) => c.uuid === conversation.uuid)
    if (cur !== -1) rows.splice(cur, 1)
    const at = Math.min(index, rows.length)
    rows.splice(at, 0, conversation)
    if (cur === -1) added += 1
  }
  return { data: rows, added, total: totalRef + added }
}