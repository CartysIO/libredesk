import { describe, expect, it, vi } from 'vitest'
import { createLogout } from './useLogout'

describe('createLogout', () => {
  it('clears the push association before logging out', async () => {
    const pushNotifications = { disable: vi.fn().mockResolvedValue() }
    const browser = { location: { href: '' } }

    await createLogout(pushNotifications, browser)()

    expect(pushNotifications.disable).toHaveBeenCalled()
    expect(browser.location.href).toBe('/logout')
  })

  it('logs out when clearing the push association fails', async () => {
    const pushNotifications = { disable: vi.fn().mockRejectedValue(new Error('offline')) }
    const browser = { location: { href: '' } }

    await createLogout(pushNotifications, browser)()

    expect(browser.location.href).toBe('/logout')
  })

  it('asks the next login page to end the Clerk session', async () => {
    const pushNotifications = { disable: vi.fn().mockResolvedValue() }
    const stored = {}
    const browser = {
      location: { href: '' },
      sessionStorage: { setItem: (k, v) => (stored[k] = v) }
    }

    await createLogout(pushNotifications, browser)()

    expect(stored.libredesk_signed_out).toBe('1')
  })
})
