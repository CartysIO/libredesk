import { usePushNotifications } from '@main/composables/usePushNotifications'
import { clearNewConversationDrafts } from '@main/features/conversation/useNewConversationDraft.js'
import { markSignedOut } from '@main/utils/clerk.js'

const defaultBrowser = typeof window === 'undefined' ? {} : window

export const createLogout = (pushNotifications, browser = defaultBrowser) => async () => {
  await Promise.resolve(pushNotifications.disable()).catch(() => {})
  clearNewConversationDrafts(browser.localStorage)
  markSignedOut(browser.sessionStorage)
  browser.location.href = '/logout'
}

export const useLogout = () => createLogout(usePushNotifications())
