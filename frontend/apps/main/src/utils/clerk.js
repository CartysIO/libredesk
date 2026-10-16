// Loads Clerk's browser SDK and its UI bundle (the prebuilt SignIn component)
// from the Clerk instance's Frontend API, the same way Clerk's own SDKs do.

const CLERK_JS_MAJOR = '6'
const CLERK_UI_MAJOR = '1'
const SIGNED_OUT_KEY = 'libredesk_signed_out'
const LOGIN_ERROR_KEY = 'libredesk_login_error'

let clerkPromise = null

// clerkFrontendApi decodes the Frontend API host from a publishable key:
// "pk_test_" or "pk_live_" followed by base64("<host>$").
export const clerkFrontendApi = (publishableKey) => {
  const match = /^pk_(test|live)_(.+)$/.exec(publishableKey || '')
  if (!match) return ''
  try {
    const host = atob(match[2]).replace(/\$$/, '')
    return /^[a-z0-9.-]+$/i.test(host) ? host : ''
  } catch {
    return ''
  }
}

const loadScript = (browser, src, attrs = {}) =>
  new Promise((resolve, reject) => {
    const script = browser.document.createElement('script')
    script.src = src
    script.async = true
    script.crossOrigin = 'anonymous'
    for (const [name, value] of Object.entries(attrs)) script.setAttribute(name, value)
    script.onload = resolve
    script.onerror = () => reject(new Error('Could not load Clerk'))
    browser.document.head.appendChild(script)
  })

export const loadClerk = (publishableKey, browser = window) => {
  if (clerkPromise) return clerkPromise
  const host = clerkFrontendApi(publishableKey)
  if (!host) return Promise.reject(new Error('Invalid Clerk publishable key'))

  clerkPromise = Promise.all([
    loadScript(browser, `https://${host}/npm/@clerk/ui@${CLERK_UI_MAJOR}/dist/ui.browser.js`),
    loadScript(
      browser,
      `https://${host}/npm/@clerk/clerk-js@${CLERK_JS_MAJOR}/dist/clerk.browser.js`,
      {
        'data-clerk-publishable-key': publishableKey
      }
    )
  ])
    .then(async () => {
      await browser.Clerk.load({ ui: { ClerkUI: browser.__internal_ClerkUICtor } })
      return browser.Clerk
    })
    .catch((error) => {
      clerkPromise = null
      throw error
    })
  return clerkPromise
}

// markSignedOut tells the next login page to end the Clerk session too,
// so logging out of Libredesk does not sign the agent straight back in.
export const markSignedOut = (storage) => {
  try {
    storage?.setItem(SIGNED_OUT_KEY, '1')
  } catch {
    // Storage can be unavailable in private windows.
  }
}

export const takeSignedOut = (storage) => {
  try {
    const value = storage?.getItem(SIGNED_OUT_KEY)
    storage?.removeItem(SIGNED_OUT_KEY)
    return value === '1'
  } catch {
    return false
  }
}

// Clerk reloads the page when it signs out, so a login error is kept here
// and shown by the next login page.
export const keepLoginError = (storage, message) => {
  try {
    storage?.setItem(LOGIN_ERROR_KEY, message)
  } catch {
    // Storage can be unavailable in private windows.
  }
}

export const takeLoginError = (storage) => {
  try {
    const value = storage?.getItem(LOGIN_ERROR_KEY)
    storage?.removeItem(LOGIN_ERROR_KEY)
    return value || ''
  } catch {
    return ''
  }
}
