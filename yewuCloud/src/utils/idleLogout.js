import router from '@/router'
import { getToken } from '@/utils/auth.js'
import store from '@/store'
import { requirePasswordChange } from '@/utils/passwordChange.js'
import { PASSWORD_CHANGE_REQUIRED } from '@/utils/passwordChangeState.js'
import {
  expireSession,
  getSessionToken,
  LAST_ACTIVITY_KEY,
  SESSION_TIMEOUT_EVENT_KEY
} from '@/utils/sessionExpiration'
import { isCurrentSessionTimeout } from '@/utils/sessionEvents'

const IDLE_TIMEOUT_MS = 30 * 60 * 1000
const SESSION_REFRESH_INTERVAL_MS = 5 * 60 * 1000
const ACTIVITY_EVENTS = ['click', 'keydown', 'mousedown', 'mousemove', 'scroll', 'touchstart', 'wheel']

let timer = null
let redirecting = false
let lastRecordedAt = 0
let lastRefreshAt = 0
let lastRefreshToken = ''
let refreshingToken = ''

function currentRoute() {
  return router.currentRoute && router.currentRoute.value
}

function isLoginRoute() {
  const route = currentRoute()
  return route && route.path === '/login'
}

function readLastActivity() {
  const value = Number(localStorage.getItem(LAST_ACTIVITY_KEY))
  return Number.isFinite(value) && value > 0 ? value : Date.now()
}

function writeLastActivity(value) {
  localStorage.setItem(LAST_ACTIVITY_KEY, String(value))
}

function redirectPath() {
  const route = currentRoute()
  return (route && route.fullPath) || window.location.hash.slice(1) || '/'
}

function clearTimer() {
  if (timer) {
    window.clearTimeout(timer)
    timer = null
  }
}

function scheduleCheck() {
  clearTimer()
  if (!getToken() || isLoginRoute()) {
    return
  }
  const remaining = IDLE_TIMEOUT_MS - (Date.now() - readLastActivity())
  timer = window.setTimeout(checkIdle, Math.max(remaining, 1000))
}

function timeoutLogout(options = {}) {
  const { broadcast = true } = options
  if (redirecting || isLoginRoute()) {
    return
  }
  redirecting = true
  const redirect = redirectPath()
  clearTimer()
  expireSession({
    broadcast,
    notifyBackend: broadcast,
    redirectPath: redirect,
    token: getSessionToken()
  }).finally(() => {
    // 单页应用重新登录不会重新加载本模块，下一次超时仍需能够退出。
    redirecting = false
  })
}

function checkIdle() {
  if (!getToken()) {
    clearTimer()
    return
  }
  if (Date.now() - readLastActivity() >= IDLE_TIMEOUT_MS) {
    timeoutLogout()
    return
  }
  scheduleCheck()
}

function markActivity() {
  if (!getToken() || isLoginRoute()) {
    return
  }
  const now = Date.now()
  if (now - lastRecordedAt < 1000) {
    return
  }
  lastRecordedAt = now
  writeLastActivity(now)
  scheduleCheck()
  refreshActiveSession()
}

/** 仅实际操作触发续期；后台轮询和自动刷新不会延长前端的无操作时间。 */
function refreshActiveSession(force = false) {
  const token = getToken()
  const now = Date.now()
  if (redirecting || !token || isLoginRoute() || document.visibilityState === 'hidden' ||
      now - readLastActivity() >= IDLE_TIMEOUT_MS || refreshingToken === token) return
  if (!force && lastRefreshToken === token && now - lastRefreshAt < SESSION_REFRESH_INTERVAL_MS) return

  // 失败也按本次尝试计时，防止服务故障时每次移动鼠标都重复请求。
  lastRefreshToken = token
  lastRefreshAt = now
  refreshingToken = token
  store.dispatch('user/checkPasswordChange').then(status => {
    if (getToken() === token && status.passwordChangeRequired) requirePasswordChange(status, token)
  }).catch(error => {
    if (getToken() === token && error?.errorCode === PASSWORD_CHANGE_REQUIRED) {
      requirePasswordChange(error.responseData, token)
    }
  }).finally(() => {
    if (refreshingToken === token) refreshingToken = ''
  })
}

function handleVisibilityChange() {
  if (document.visibilityState === 'visible') {
    checkIdle()
    // 恢复到前台时同时核验季度改密状态，不更新人工操作时间。
    refreshActiveSession(true)
  }
}

function handleStorage(event) {
  if (event.key === LAST_ACTIVITY_KEY && event.newValue) {
    scheduleCheck()
    return
  }
  if (event.key === SESSION_TIMEOUT_EVENT_KEY && isCurrentSessionTimeout(event.newValue, getSessionToken())) {
    timeoutLogout({ broadcast: false })
  }
}

export function setupIdleLogout() {
  if (!localStorage.getItem(LAST_ACTIVITY_KEY)) {
    writeLastActivity(Date.now())
  }
  ACTIVITY_EVENTS.forEach(eventName => {
    window.addEventListener(eventName, markActivity, { passive: true, capture: true })
  })
  document.addEventListener('visibilitychange', handleVisibilityChange)
  window.addEventListener('storage', handleStorage)
  router.afterEach((to, from) => {
    if (getToken() && !isLoginRoute()) {
      if (!localStorage.getItem(LAST_ACTIVITY_KEY) || (from && from.path === '/login')) {
        writeLastActivity(Date.now())
      }
      scheduleCheck()
    } else {
      clearTimer()
    }
  })
  scheduleCheck()
}
