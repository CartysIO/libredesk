import { describe, it, expect } from 'vitest'
import { messageRole } from './messageRole'

describe('messageRole', () => {
  it('reads a contact message as customer', () => {
    expect(messageRole({ type: 'incoming', sender_type: 'contact', author: { type: 'contact' } })).toBe('customer')
    expect(messageRole({ type: 'incoming', author: { type: 'visitor' } })).toBe('customer')
    expect(messageRole({ type: 'incoming' })).toBe('customer')
    expect(messageRole({ type: 'outgoing', sender_type: 'contact' })).toBe('customer')
  })

  it('reads a human agent reply as agent', () => {
    expect(messageRole({ type: 'outgoing', sender_type: 'agent', author: { type: 'agent' } })).toBe('agent')
    expect(messageRole({ type: 'outgoing' })).toBe('agent')
    expect(messageRole(undefined)).toBe('agent')
  })

  it('reads an AI assistant reply as ai', () => {
    expect(messageRole({ type: 'outgoing', sender_type: 'agent', author: { type: 'ai_assistant' } })).toBe('ai')
  })

  it('a private note wins over the author type', () => {
    expect(messageRole({ type: 'outgoing', private: true, author: { type: 'agent' } })).toBe('note')
    expect(messageRole({ type: 'outgoing', private: true, author: { type: 'ai_assistant' } })).toBe('note')
  })
})
