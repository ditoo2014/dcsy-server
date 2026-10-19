import store from '@/store'
import router from '@/router'
import { getToken } from '@/utils/auth.js'
import { notifyBackendLogout } from '@/utils/sessionLogout'
import { broadcastSessionTimeout, LAST_ACTIVITY_KEY } from '@/utils/sessionEvents'

export { LAST_ACTIVITY_KEY, SESSION_TIMEOUT_EVENT_KEY } from '@/utils/sessionEvents'

let expirationPromise = null
let expirationToken = ''

// 其他窗口可能已经移除了共享 Cookie，此时仍需清理本窗口的旧登录状态。
export function getSessionToken() {
  return getToken() || store.state.user.token || ''
}

function currentRedirectPath() {
  const hashPath = window.location.hash.replace(/^#/, '')
  if (!hashPath || hashPath.startsWith('/login')) {
    return '/'
  }
  return hashPath
}

export function expireSession(options = {}) {
  const {
    broadcast = true,
    notifyBackend = false,
    redirectPath = currentRedirectPath(),
    token = getSessionToken()
  } = options

  if (!token || token !== getSessionToken()) return Promise.resolve()
  if (expirationPromise && expirationToken === token) return expirationPromise

  if (broadcast) {
    broadcastSessionTimeout(token)
  }
  localStorage.removeItem(LAST_ACTIVITY_KEY)

  if (notifyBackend) {
    notifyBackendLogout(token)
  }

  expirationToken = token
  expirationPromise = Promise.resolve()
    .then(async() => {
      // 异步清理开始前再次核对，避免旧响应清除刚刚登录的新会话。
      if (token !== getSessionToken()) return false
      await store.dispatch('user/sessionTimeout')
      return true
    })
    .catch(error => {
      console.error('[session] clear session state failed', error)
    })
    .then(cleared => {
      if (cleared && !getToken()) {
        return router.replace({ path: '/login', query: { redirect: redirectPath || '/' }})
      }
    })
    .finally(() => {
      // 只合并当前正在执行的退出操作，重新登录后允许再次退出。
      if (expirationToken === token) {
        expirationPromise = null
        expirationToken = ''
      }
    })

  return expirationPromise
}
