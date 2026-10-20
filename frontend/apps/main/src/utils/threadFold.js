// Gmail-style folding for email threads (Clyr desk). Old messages fold to one-line rows so the agent
// reads the newest reply first; a long run of folded rows shrinks further to first, "N earlier", last.

export const FOLD_RUN_MIN = 4

const isThreadMessage = (message) => message && message.type !== 'activity'

const isPublicAgentReply = (message) =>
  message.type === 'outgoing' && !message.private && message.status !== 'failed'

// Messages open by default: the newest message, everything after the last public agent reply (the
// customer is waiting on us), sends that are pending or failed, and a message a link points to.
export const defaultOpenUUIDs = (messages, focusUUID = '') => {
  const open = new Set()
  const thread = (messages || []).filter(isThreadMessage)
  if (!thread.length) return open

  open.add(thread[thread.length - 1].uuid)

  let lastReply = -1
  thread.forEach((message, index) => {
    if (isPublicAgentReply(message)) lastReply = index
  })
  thread.slice(lastReply + 1).forEach((message) => open.add(message.uuid))

  thread.forEach((message) => {
    if (message.type === 'outgoing' && ['pending', 'failed'].includes(message.status)) open.add(message.uuid)
  })
  if (focusUUID) open.add(focusUUID)
  return open
}

// Turns message rows into display rows: { kind: 'message' | 'fold' | 'summary' }.
// `isOpen(message)` decides open vs folded; `openRuns` holds the keys of runs the agent expanded.
export const foldThreadRows = (rows, isOpen, openRuns = new Set()) => {
  const out = []
  let run = []

  const flush = () => {
    if (!run.length) return
    const key = run[0].message.uuid
    if (run.length >= FOLD_RUN_MIN && !openRuns.has(key)) {
      out.push({ kind: 'fold', row: run[0] })
      out.push({ kind: 'summary', key, count: run.length - 2 })
      out.push({ kind: 'fold', row: run[run.length - 1] })
    } else {
      run.forEach((row) => out.push({ kind: 'fold', row }))
    }
    run = []
  }

  rows.forEach((row) => {
    if (isThreadMessage(row.message) && !isOpen(row.message)) {
      run.push(row)
      return
    }
    flush()
    out.push({ kind: 'message', row })
  })
  flush()
  return out
}

// One line of plain text for a folded row.
export const messageSnippet = (message, max = 160) => {
  const text = (message?.text_content || message?.content || '')
    .replace(/<[^>]*>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  return text.length > max ? `${text.slice(0, max).trimEnd()}…` : text
}
