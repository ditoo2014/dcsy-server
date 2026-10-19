/**
 * 安全解析 JSON，避免 localStorage / 路由参数为空时导致页面白屏
 */
export function safeParseJson(str, fallback = null) {
  if (str == null || str === '') {
    return fallback
  }
  try {
    return JSON.parse(str)
  } catch {
    return fallback
  }
}

const USER_INFO_STR_KEY = 'userInfoStr'

export function parseUserInfo(fallback = {}) {
  const parsed = safeParseJson(localStorage.getItem(USER_INFO_STR_KEY), fallback)
  return parsed && typeof parsed === 'object' ? parsed : fallback
}
