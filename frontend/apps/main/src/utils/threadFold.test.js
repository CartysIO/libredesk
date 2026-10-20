import { describe, expect, it } from 'vitest'
import { defaultOpenUUIDs, foldThreadRows, messageSnippet, FOLD_RUN_MIN } from './threadFold'

const msg = (uuid, type = 'incoming', extra = {}) => ({ uuid, type, private: false, status: 'sent', ...extra })
const rowsOf = (messages) => messages.map((message) => ({ message }))

describe('defaultOpenUUIDs', () => {
  it('opens the newest message and the customer messages after the last agent reply', () => {
    const thread = [msg('a'), msg('b', 'outgoing'), msg('c'), msg('d')]
    expect([...defaultOpenUUIDs(thread)].sort()).toEqual(['c', 'd'])
  })

  it('opens only the newest message when the agent replied last', () => {
    const thread = [msg('a'), msg('b', 'outgoing')]
    expect([...defaultOpenUUIDs(thread)]).toEqual(['b'])
  })

  it('does not count a private note or a failed send as a reply', () => {
    const thread = [msg('a'), msg('n', 'outgoing', { private: true }), msg('f', 'outgoing', { status: 'failed' }), msg('b', 'outgoing'), msg('c')]
    const open = defaultOpenUUIDs(thread)
    expect(open.has('c')).toBe(true)
    expect(open.has('f')).toBe(true) // failed sends stay visible
    expect(open.has('n')).toBe(false)
    expect(open.has('a')).toBe(false)
  })

  it('ignores activity lines and opens a linked message', () => {
    const thread = [msg('a'), msg('b', 'outgoing'), msg('x', 'activity')]
    const open = defaultOpenUUIDs(thread, 'a')
    expect(open.has('b')).toBe(true)
    expect(open.has('a')).toBe(true)
    expect(open.has('x')).toBe(false)
  })

  it('handles an empty thread', () => {
    expect(defaultOpenUUIDs([]).size).toBe(0)
  })
})

describe('foldThreadRows', () => {
  it('folds closed messages and keeps open ones', () => {
    const rows = rowsOf([msg('a'), msg('b'), msg('c')])
    const out = foldThreadRows(rows, (m) => m.uuid === 'c')
    expect(out.map((r) => r.kind)).toEqual(['fold', 'fold', 'message'])
  })

  it(`shrinks a run of ${FOLD_RUN_MIN} or more folded rows to first, summary, last`, () => {
    const rows = rowsOf([msg('a'), msg('b'), msg('c'), msg('d'), msg('e'), msg('z')])
    const out = foldThreadRows(rows, (m) => m.uuid === 'z')
    expect(out.map((r) => r.kind)).toEqual(['fold', 'summary', 'fold', 'message'])
    expect(out[1]).toEqual({ kind: 'summary', key: 'a', count: 3 })
    expect(out[2].row.message.uuid).toBe('e')
  })

  it('shows every folded row once the run is expanded', () => {
    const rows = rowsOf([msg('a'), msg('b'), msg('c'), msg('d'), msg('z')])
    const out = foldThreadRows(rows, (m) => m.uuid === 'z', new Set(['a']))
    expect(out.map((r) => r.kind)).toEqual(['fold', 'fold', 'fold', 'fold', 'message'])
  })

  it('lets an activity line break a run', () => {
    const rows = rowsOf([msg('a'), msg('b'), msg('x', 'activity'), msg('c'), msg('d'), msg('z')])
    const out = foldThreadRows(rows, (m) => m.uuid === 'z')
    expect(out.map((r) => r.kind)).toEqual(['fold', 'fold', 'message', 'fold', 'fold', 'message'])
  })
})

describe('messageSnippet', () => {
  it('flattens HTML and whitespace and trims to length', () => {
    expect(messageSnippet({ content: '<p>Hi  there,</p>\n<p>thanks</p>' })).toBe('Hi there, thanks')
    expect(messageSnippet({ text_content: 'x'.repeat(10) }, 4)).toBe('xxxx…')
  })
})
