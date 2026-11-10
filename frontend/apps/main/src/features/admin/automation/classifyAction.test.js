import { describe, test, expect } from 'vitest'
import { classifyActionValue } from './classifyAction'

describe('classifyActionValue', () => {
  test('key only', () => {
    expect(classifyActionValue('question_type', '  ', {})).toEqual(['question_type'])
  })

  test('key and instructions', () => {
    expect(classifyActionValue('question_type', ' Prefer Bug ', undefined)).toEqual(['question_type', 'Prefer Bug'])
  })

  test('descriptions without instructions keep an empty instructions slot', () => {
    expect(classifyActionValue('question_type', '', { Bug: ' broken ', 'User Error': ' ' })).toEqual([
      'question_type',
      '',
      '{"Bug":"broken"}'
    ])
  })
})
