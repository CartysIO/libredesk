import { describe, expect, it } from 'vitest'
import {
  clearSignedOut,
  clerkFrontendApi,
  isMarkedSignedOut,
  keepLoginError,
  markSignedOut,
  takeLoginError
} from './clerk'

const memoryStorage = () => {
  const data = new Map()
  return {
    getItem: (k) => (data.has(k) ? data.get(k) : null),
    setItem: (k, v) => data.set(k, String(v)),
    removeItem: (k) => data.delete(k)
  }
}

describe('clerkFrontendApi', () => {
  it('decodes the host from test and live keys', () => {
    expect(clerkFrontendApi('pk_test_' + btoa('sacred-chimp-95.clerk.accounts.dev$'))).toBe(
      'sacred-chimp-95.clerk.accounts.dev'
    )
    expect(clerkFrontendApi('pk_live_' + btoa('clerk.clyr.io$'))).toBe('clerk.clyr.io')
  })

  it('rejects anything that is not a publishable key', () => {
    expect(clerkFrontendApi('')).toBe('')
    expect(clerkFrontendApi(undefined)).toBe('')
    expect(clerkFrontendApi('sk_test_abc')).toBe('')
    expect(clerkFrontendApi('pk_test_!!!')).toBe('')
    expect(clerkFrontendApi('pk_test_' + btoa('evil.com/x$'))).toBe('')
  })
})

describe('signed-out flag', () => {
  it('stays set until cleared', () => {
    const storage = memoryStorage()
    expect(isMarkedSignedOut(storage)).toBe(false)
    markSignedOut(storage)
    expect(isMarkedSignedOut(storage)).toBe(true)
    expect(isMarkedSignedOut(storage)).toBe(true)
    clearSignedOut(storage)
    expect(isMarkedSignedOut(storage)).toBe(false)
  })
})

describe('login error', () => {
  it('survives one reload', () => {
    const storage = memoryStorage()
    expect(takeLoginError(storage)).toBe('')
    keepLoginError(storage, 'No agent account')
    expect(takeLoginError(storage)).toBe('No agent account')
    expect(takeLoginError(storage)).toBe('')
  })
})
