import router from '@/router'
import { getToken } from '@/utils/auth.js'
import {
  expireSession,
  LAST_ACTIVITY_KEY,
  SESSION_TIMEOUT_EVENT_KEY
} from '@/utils/session-expiration'

const IDLE_TIMEOUT_MS = 30 * 60 * 1000
const ACTIVITY_EVENTS = ['click', 'keydown', 'mousedown', 'mousemove', 'scroll', 'touchstart', 'wheel']

let timer = null
let redirecting = false
let lastRecordedAt = 0

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
    redirectPath: redirect
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
}

function handleVisibilityChange() {
  if (document.visibilityState === 'visible') {
    checkIdle()
  }
}

function handleStorage(event) {
  if (event.key === LAST_ACTIVITY_KEY && event.newValue) {
    scheduleCheck()
    return
  }
  if (event.key === SESSION_TIMEOUT_EVENT_KEY && event.newValue) {
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
