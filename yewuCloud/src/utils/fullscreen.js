import screenfull from 'screenfull'

/** 浏览器是否支持 Fullscreen API */
export function isFullscreenSupported() {
  return screenfull.isEnabled || !!(
    document.fullscreenEnabled ||
    document.webkitFullscreenEnabled ||
    document.mozFullScreenEnabled
  )
}

function getFullscreenElement() {
  return document.fullscreenElement ||
    document.webkitFullscreenElement ||
    document.mozFullScreenElement ||
    document.msFullscreenElement ||
    null
}

/** 当前是否处于全屏 */
export function isFullscreenActive() {
  if (screenfull.isEnabled) {
    return screenfull.isFullscreen
  }
  return !!getFullscreenElement()
}

function nativeRequestFullscreen(element) {
  const el = element || document.documentElement
  const req =
    el.requestFullscreen ||
    el.webkitRequestFullscreen ||
    el.webkitRequestFullScreen ||
    el.mozRequestFullScreen ||
    el.msRequestFullscreen
  if (!req) {
    return Promise.resolve(false)
  }
  const result = req.call(el)
  if (result && typeof result.then === 'function') {
    return result.then(() => true).catch(() => false)
  }
  return Promise.resolve(true)
}

function doRequestFullscreen(target) {
  if (screenfull.isEnabled) {
    return Promise.resolve(screenfull.request(target))
      .then(() => true)
      .catch(() => false)
  }
  return nativeRequestFullscreen(target)
}

/**
 * 安全请求全屏（捕获 Promise 拒绝，避免白屏）
 * @param {HTMLElement} element 目标元素，默认 documentElement
 * @returns {Promise<boolean>} 是否成功进入全屏
 */
export function requestFullscreenSafe(element) {
  if (!isFullscreenSupported()) {
    return Promise.resolve(false)
  }
  const target = element || document.documentElement
  const fsEl = getFullscreenElement()

  if (fsEl === target) {
    return Promise.resolve(true)
  }
  if (fsEl) {
    return exitFullscreenSafe().then(() => doRequestFullscreen(target))
  }
  return doRequestFullscreen(target)
}

/** 监听全屏状态变化（含 ESC 退出） */
export function onFullscreenChange(handler) {
  if (screenfull.isEnabled) {
    screenfull.on('change', handler)
    return () => screenfull.off('change', handler)
  }
  const events = ['fullscreenchange', 'webkitfullscreenchange', 'mozfullscreenchange', 'MSFullscreenChange']
  events.forEach(name => document.addEventListener(name, handler))
  return () => events.forEach(name => document.removeEventListener(name, handler))
}

/** 驾驶舱缩放容器（不含顶部 Banner） */
export function getDashboardScreenWrapper() {
  return document.querySelector('.app-wrapper--dashboard .screen-wrapper')
}

/** 是否处于驾驶舱 API 全屏 */
export function isDashboardApiFullscreen() {
  const wrapper = getDashboardScreenWrapper()
  return !!(wrapper && getFullscreenElement() === wrapper)
}

/** 是否像浏览器 F11 全屏（非 Fullscreen API） */
export function looksLikeBrowserF11Fullscreen() {
  if (getFullscreenElement()) {
    return false
  }
  const chromeGap = window.outerHeight - window.innerHeight
  const tallViewport = window.innerHeight >= (window.screen.availHeight * 0.92)
  return chromeGap < 80 && tallViewport
}

/** 请求驾驶舱区域全屏（路由切换后等待 .screen-wrapper 挂载） */
export function requestDashboardFullscreen(options = {}) {
  const { maxRetries = 20, interval = 50 } = options
  return new Promise((resolve) => {
    const attempt = (left) => {
      const el = getDashboardScreenWrapper()
      if (el) {
        requestFullscreenSafe(el).then(resolve)
        return
      }
      if (left <= 0) {
        resolve(false)
        return
      }
      setTimeout(() => attempt(left - 1), interval)
    }
    attempt(maxRetries)
  })
}

/** 切换驾驶舱区域全屏 */
export function toggleDashboardFullscreen() {
  if (!isFullscreenSupported()) {
    return Promise.resolve(false)
  }
  const el = getDashboardScreenWrapper()
  if (!el) {
    return Promise.resolve(false)
  }
  if (screenfull.isEnabled) {
    return Promise.resolve(screenfull.toggle(el)).catch(() => false)
  }
  if (isDashboardApiFullscreen()) {
    return exitFullscreenSafe().then(() => true)
  }
  return requestFullscreenSafe(el)
}

export function exitFullscreenSafe() {
  if (!isFullscreenActive()) {
    return Promise.resolve()
  }
  if (screenfull.isEnabled) {
    return Promise.resolve(screenfull.exit()).catch(() => {})
  }
  const exit =
    document.exitFullscreen ||
    document.webkitExitFullscreen ||
    document.webkitCancelFullScreen ||
    document.mozCancelFullScreen ||
    document.msExitFullscreen
  if (exit) {
    const result = exit.call(document)
    if (result && typeof result.then === 'function') {
      return result.catch(() => {})
    }
  }
  return Promise.resolve()
}
