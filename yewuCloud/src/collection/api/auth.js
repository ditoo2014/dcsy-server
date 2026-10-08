import { JsonPost } from '@/api/api.js'

// 采集云与业务云共用登录令牌，仅补充读取采集云按钮权限。
export function fetchAuthInfo() {
  return JsonPost('auth/info', {})
}

export function fetchTolinkPermissions() {
  return JsonPost('auth/tolink/user/permissions', {})
}

export function readAuthUser(response) {
  const payload = response?.return
  return payload?.user || payload || {}
}
