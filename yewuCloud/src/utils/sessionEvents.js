import md5 from 'js-md5'

export const LAST_ACTIVITY_KEY = 'dicloud:last-user-activity-at'
export const SESSION_TIMEOUT_EVENT_KEY = 'dicloud:session-timeout-at'

/** 摘要仅用于区分浏览器会话，不用于认证，也不在退出通知中保存原始令牌。 */
export function broadcastSessionTimeout(token) {
  if (!token) return
  localStorage.setItem(SESSION_TIMEOUT_EVENT_KEY, JSON.stringify({
    sessionId: md5(token),
    at: Date.now()
  }))
}

/** 忽略其他登录会话以及旧版本只带时间戳的退出通知。 */
export function isCurrentSessionTimeout(value, token) {
  if (!value || !token) return false
  try {
    const event = JSON.parse(value)
    return event?.sessionId === md5(token)
  } catch {
    return false
  }
}
