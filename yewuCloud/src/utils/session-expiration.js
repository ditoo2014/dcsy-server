import store from '@/store'
import router from '@/router'
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

  expirationPromise = Promise.resolve()
    .then(() => store.dispatch('user/sessionTimeout'))
    .catch(error => {
      console.error('[session] clear session state failed', error)
    })
    .then(() => router.replace({ path: '/login', query: { redirect: redirectPath || '/' }}))
    .finally(() => {
      // 只合并当前正在执行的退出操作，重新登录后允许再次退出。
      expirationPromise = null
    })

  return expirationPromise
}
