// Collects the context that a chat starts with: the URL of the host page, the app label that the host page
// set in window.LibredeskSettings, and the browser timezone. The widget runs in an iframe, so the host
// page (widget.js) must give the URL. The widget asks for it when the chat starts, so a single-page app
// that changed route after load gives its current page.

const REPLY_TIMEOUT_MS = 1000

const browserTimezone = () => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || ''
  } catch {
    return ''
  }
}

const askHostPage = () =>
  new Promise((resolve) => {
    if (!window.parent || window.parent === window) return resolve({})
    const parentOrigin = new URLSearchParams(window.location.search).get('parent_origin')
    if (!parentOrigin || parentOrigin === 'null') return resolve({})

    const requestId = Math.random().toString(36).slice(2)
    let timer
    const onMessage = (event) => {
      const data = event.data
      if (event.source !== window.parent || event.origin !== parentOrigin) return
      if (!data || data.type !== 'PAGE_CONTEXT' || data.requestId !== requestId) return
      clearTimeout(timer)
      window.removeEventListener('message', onMessage)
      resolve({ page_url: data.url, app: data.app })
    }
    window.addEventListener('message', onMessage)
    timer = setTimeout(() => {
      window.removeEventListener('message', onMessage)
      resolve({})
    }, REPLY_TIMEOUT_MS)
    window.parent.postMessage({ type: 'REQUEST_PAGE_CONTEXT', requestId }, parentOrigin)
  })

// Returns { page_url, app, timezone }. A field is left out when it is not known. It never rejects.
export const getChatContext = async () => {
  const context = {}
  try {
    const page = await askHostPage()
    if (typeof page.page_url === 'string' && page.page_url) context.page_url = page.page_url
    if (typeof page.app === 'string' && page.app) context.app = page.app
  } catch {
    // The chat starts without the page data.
  }
  const timezone = browserTimezone()
  if (timezone) context.timezone = timezone
  return context
}
