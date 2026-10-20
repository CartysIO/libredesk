import { describe, expect, it } from 'vitest'
import { initials } from './initials'

describe('initials', () => {
  it('takes the first letter of the first and last name', () => {
    expect(initials('Dana', 'Levi')).toBe('DL')
    expect(initials('tom', 'alvarez')).toBe('TA')
  })

  it('handles one name or none', () => {
    expect(initials('Google', '')).toBe('G')
    expect(initials('', '', 'U')).toBe('U')
    expect(initials(undefined, null)).toBe('?')
  })
})
