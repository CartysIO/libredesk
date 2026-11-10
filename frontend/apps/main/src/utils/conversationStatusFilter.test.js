import { describe, expect, it } from 'vitest'
import {
  CONVERSATION_ACTIVE_STATUS,
  CONVERSATION_DEFAULT_STATUSES_LIST
} from '@main/constants/conversation'
import { isValidListStatus, listStatusFilter, matchesListStatus } from './conversationStatusFilter'

describe('listStatusFilter', () => {
  it('maps Active to an in rule on Open, On you and On customer', () => {
    const rule = listStatusFilter(CONVERSATION_ACTIVE_STATUS)
    expect(rule).toMatchObject({ model: 'conversation_statuses', field: 'name', operator: 'in' })
    expect(JSON.parse(rule.value)).toEqual(['Open', 'On you', 'On customer'])
  })

  it('keeps an exact match for a single status', () => {
    expect(listStatusFilter('Open')).toEqual({
      model: 'conversation_statuses', field: 'name', operator: 'equals', value: 'Open'
    })
  })

  it('accepts Active even though it is not a status row', () => {
    expect(isValidListStatus(CONVERSATION_ACTIVE_STATUS, [{ name: 'Open' }])).toBe(true)
    expect(isValidListStatus('Open', [{ name: 'Open' }])).toBe(true)
    expect(isValidListStatus('Gone', [{ name: 'Open' }])).toBe(false)
  })

  it('keeps a custom status named Active on the exact-name path', () => {
    expect(listStatusFilter('Active')).toEqual({
      model: 'conversation_statuses',
      field: 'name',
      operator: 'equals',
      value: 'Active'
    })
    expect(matchesListStatus('Active', 'Active')).toBe(true)
    expect(matchesListStatus('On you', 'Active')).toBe(false)
  })

  it('matches conversations client-side', () => {
    expect(matchesListStatus('On you', CONVERSATION_ACTIVE_STATUS)).toBe(true)
    expect(matchesListStatus('Resolved', CONVERSATION_ACTIVE_STATUS)).toBe(false)
    expect(matchesListStatus('Open', 'Open')).toBe(true)
    expect(matchesListStatus('On you', 'Open')).toBe(false)
    expect(matchesListStatus('Resolved', '')).toBe(true)
  })

  it('treats the board statuses as default statuses', () => {
    expect(CONVERSATION_DEFAULT_STATUSES_LIST).toEqual(
      expect.arrayContaining(['Open', 'Snoozed', 'Resolved', 'On you', 'On customer'])
    )
    expect(CONVERSATION_DEFAULT_STATUSES_LIST).not.toContain('Closed')
  })
})
