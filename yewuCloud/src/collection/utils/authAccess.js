import { fetchAuthInfo, fetchTolinkPermissions, readAuthUser } from '../api/auth.js'
import { useAuthStore } from '../stores/auth.js'
import { getToken } from '@/utils/auth.js'
import { isPasswordChangeRequired, PASSWORD_CHANGE_REQUIRED } from '@/utils/password-change-state.js'
import { requirePasswordChange } from '@/utils/password-change.js'

let loadingToken = ''
let loadingPromise

/** 加载采集云权限，避免每次切换内部页面都重复请求。 */
export function ensureTolinkAccess(token, force = false) {
  const authStore = useAuthStore()
  if (!force && authStore.accessLoaded && token === authStore.token) {
    return Promise.resolve({ isadmin: authStore.isAdmin ? 1 : 0, username: authStore.account })
  }
  if (!force && loadingPromise && loadingToken === token) return loadingPromise

  loadingToken = token
  authStore.setToken(token)
  loadingPromise = (async() => {
    const infoResponse = await fetchAuthInfo()
    if (isPasswordChangeRequired(infoResponse)) {
      requirePasswordChange(infoResponse, token)
      const error = new Error('请先修改密码后继续使用平台')
      error.errorCode = PASSWORD_CHANGE_REQUIRED
      error.responseData = infoResponse
      throw error
    }
    if (getToken() !== token) throw new Error('登录会话已变化，请重新登录')
    const user = readAuthUser(infoResponse)
    const isAdmin = Number(user.isadmin || 0) === 1
    const username = user.username || user.userName || user.nickname || user.nickName || user.name || ''
    if (username) {
      authStore.setUsername(username)
      authStore.setAccount(username)
    }

    let permissionUrls = []
    if (!isAdmin) {
      const permissionResponse = await fetchTolinkPermissions()
      permissionUrls = (permissionResponse?.return?.data || []).map(item => String(item.url || ''))
    }
    if (getToken() !== token) throw new Error('登录会话已变化，请重新登录')
    authStore.setAccess(isAdmin, permissionUrls)
    return user
  })().finally(() => {
    loadingToken = ''
    loadingPromise = undefined
  })
  return loadingPromise
}
