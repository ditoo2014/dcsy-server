import { fetchAuthInfo, fetchTolinkPermissions, readAuthUser } from '../api/auth.js'
import { useAuthStore } from '../stores/auth.js'

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
  loadingPromise = (async() => {
    const infoResponse = await fetchAuthInfo()
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
    authStore.setAccess(isAdmin, permissionUrls)
    return user
  })().finally(() => {
    loadingToken = ''
    loadingPromise = undefined
  })
  return loadingPromise
}
