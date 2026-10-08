import store from '@/store'
import { getToken } from '@/utils/auth.js'
import { notifyBackendLogout } from '@/utils/session-logout'

export const LAST_ACTIVITY_KEY = 'dicloud:last-user-activity-at'
export const SESSION_TIMEOUT_EVENT_KEY = 'dicloud:session-timeout-at'

let expirationPromise = null

function currentRedirectPath() {
  const hashPath = window.location.hash.replace(/^#/, '')
  if (!hashPath || hashPath.startsWith('/login')) {
    return '/'
  }
  return hashPath
}

function loginUrl(redirectPath) {
  const redirect = encodeURIComponent(redirectPath || '/')
  return `${window.location.origin}${window.location.pathname}#/login?redirect=${redirect}`
}

export function expireSession(options = {}) {
  if (expirationPromise) {
    return expirationPromise
  }

  const {
    broadcast = true,
    notifyBackend = false,
    redirectPath = currentRedirectPath()
  } = options

  if (broadcast) {
    localStorage.setItem(SESSION_TIMEOUT_EVENT_KEY, String(Date.now()))
  }
  localStorage.removeItem(LAST_ACTIVITY_KEY)

  if (notifyBackend) {
    notifyBackendLogout(getToken())
  }

  expirationPromise = Promise.resolve(store.dispatch('user/sessionTimeout'))
    .catch(error => {
      console.error('[session] clear session state failed', error)
    })
    .then(() => {
      // Full navigation releases stale route records and keep-alive instances.
      window.location.replace(loginUrl(redirectPath))
    })

  return expirationPromise
}
