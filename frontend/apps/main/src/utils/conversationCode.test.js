import { describe, expect, it } from 'vitest'
import { statusCode } from './conversationCode'

describe('statusCode', () => {
  it('maps each board status to its column colour', () => {
    expect(statusCode('Open').textClass).toBe('text-code-new')
    expect(statusCode('On you').textClass).toBe('text-code-you')
    expect(statusCode('On customer').textClass).toBe('text-code-customer')
    expect(statusCode('Snoozed').textClass).toBe('text-code-hold')
    expect(statusCode('Resolved').textClass).toBe('text-code-done')
  })

  it('marks only Resolved as hollow', () => {
    expect(statusCode('Resolved').hollow).toBe(true)
    expect(statusCode('On you').hollow).toBeFalsy()
  })

  it('gives a custom status the neutral code', () => {
    expect(statusCode('Waiting on vendor')).toEqual({ key: 'other', textClass: 'text-code-hold', hollow: false })
    expect(statusCode(undefined).textClass).toBe('text-code-hold')
  })
})
