// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { getChatContext } from './chatContext.js'

const hostOrigin = 'https://app.example.com'
let parent

// Plays widget.js: answers a REQUEST_PAGE_CONTEXT from the iframe like the host page does.
const answerWith = (reply, { origin = hostOrigin, source = parent } = {}) => {
  parent.postMessage.mockImplementation((message, target) => {
    expect(target).toBe(hostOrigin)
    expect(message.type).toBe('REQUEST_PAGE_CONTEXT')
    window.dispatchEvent(
      new MessageEvent('message', {
        origin,
        source,
        data: { type: 'PAGE_CONTEXT', requestId: message.requestId, ...reply }
      })
    )
  })
}

beforeEach(() => {
  parent = { postMessage: vi.fn() }
  vi.spyOn(window, 'parent', 'get').mockReturnValue(parent)
  window.history.replaceState(
    {},
    '',
    `/widget?inbox_id=1&parent_origin=${encodeURIComponent(hostOrigin)}`
  )
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.useRealTimers()
})

describe('getChatContext', () => {
  test('returns the page URL and app from the host page and the browser timezone', async () => {
    answerWith({ url: 'https://app.example.com/billing#plan', app: 'web' })
    const context = await getChatContext()
    expect(context.page_url).toBe('https://app.example.com/billing#plan')
    expect(context.app).toBe('web')
    expect(context.timezone).toBe(Intl.DateTimeFormat().resolvedOptions().timeZone)
  })

  test('ignores a reply from another origin or window and then times out', async () => {
    vi.useFakeTimers()
    answerWith({ url: 'https://evil.example/' }, { origin: 'https://evil.example' })
    const pending = getChatContext()
    await vi.advanceTimersByTimeAsync(1100)
    const context = await pending
    expect(context.page_url).toBeUndefined()
    expect(context.timezone).toBeTruthy()
  })

  test('has no page data when the host page does not answer', async () => {
    vi.useFakeTimers()
    const pending = getChatContext()
    await vi.advanceTimersByTimeAsync(1100)
    const context = await pending
    expect(context.page_url).toBeUndefined()
    expect(context.app).toBeUndefined()
  })

  test('does not ask when the widget has no parent origin', async () => {
    window.history.replaceState({}, '', '/widget?inbox_id=1')
    const context = await getChatContext()
    expect(parent.postMessage).not.toHaveBeenCalled()
    expect(context.page_url).toBeUndefined()
  })
})
