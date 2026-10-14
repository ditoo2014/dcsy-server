import store from '@/store'
import router from '@/router'
import { getToken } from '@/utils/auth.js'
import { PASSWORD_CHANGE_PATH, isPasswordChangeRequired } from '@/utils/password-change-state.js'

let redirectPromise = null
let redirectToken = ''

/** 多个业务、采集接口同时返回改密要求时，仅保留一次清理和跳转。 */
export function requirePasswordChange(body, requestToken = getToken()) {
  const token = getToken()
  if (!token || requestToken !== token || !isPasswordChangeRequired(body)) return Promise.resolve()
  if (redirectPromise && redirectToken === token) return redirectPromise
  redirectToken = token
  const work = Promise.resolve()
    .then(() => {
      if (getToken() === token) return store.dispatch('user/requirePasswordChange', body)
    })
    .then(() => {
      if (getToken() === token && router.currentRoute.value.path !== PASSWORD_CHANGE_PATH) {
        return router.replace({ path: PASSWORD_CHANGE_PATH })
      }
    })
    .catch(error => {
      console.error('[password] 引导修改密码失败:', error)
    })
    .finally(() => {
      if (redirectPromise === work) {
        redirectPromise = null
        redirectToken = ''
      }
    })
  redirectPromise = work
  return work
}
