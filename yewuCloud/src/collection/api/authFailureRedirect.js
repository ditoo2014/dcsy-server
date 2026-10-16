import { redirectToMainLogin } from '../utils/mainLoginRedirect'
import { requirePasswordChange } from '@/utils/password-change.js'
import { isPasswordChangeFailure, requestBearerToken, PASSWORD_CHANGE_REQUIRED } from '@/utils/password-change-state.js'
import { getToken } from '@/utils/auth.js'
// 仅兼容旧接口的明确认证错误，普通业务提示不能触发退出。
const AUTH_INVALID_MESSAGES = [
  'token过期或未启用',
  'token验证失败',
  'auth header格式错误',
  '登录超时，请重新登录',
  '登录已失效，请重新登录',
  '会话已失效，请重新登录',
  'token expired',
  'jwt expired',
  'invalid token',
  'token invalid',
  'token verification failed'
]
export function isAuthSessionInvalidMessage(msg) {
  if (typeof msg !== 'string') { return false }
  const text = msg.trim().toLowerCase()
  if (!text) { return false }
  return AUTH_INVALID_MESSAGES.includes(text)
}
function isAuthFailureBody(data) {
  if (!data || typeof data !== 'object') { return false }
  const body = data
  return Number(body.code) === 401 ||
    (Number(body.code) === 400 && isAuthSessionInvalidMessage(body.msg))
}
function shouldHandleAuthFailure(config) {
  const url = String(config?.url ?? '')
  if (url.includes('/auth/login')) { return false }
  return true
}
export function redirectToLoginOnAuthFailure(token = getToken()) {
  return redirectToMainLogin({ broadcast: true, token })
}
export function attachAuthFailureRedirectInterceptor(client) {
  client.interceptors.response.use((response) => {
    if (isPasswordChangeFailure(response.data)) {
      requirePasswordChange(response.data, requestBearerToken(response.config))
      const error = new Error(response.data.msg || '请先修改密码后继续使用平台')
      error.errorCode = PASSWORD_CHANGE_REQUIRED
      error.responseData = response.data
      return Promise.reject(error)
    }
    if (requestBearerToken(response.config) === (getToken() || '') && shouldHandleAuthFailure(response.config) && isAuthFailureBody(response.data)) {
      redirectToLoginOnAuthFailure(requestBearerToken(response.config))
    }
    return response
  }, error => {
    const config = error.config
    if (isPasswordChangeFailure(error.response?.data)) {
      requirePasswordChange(error.response.data, requestBearerToken(config))
      error.errorCode = PASSWORD_CHANGE_REQUIRED
      error.responseData = error.response.data
      return Promise.reject(error)
    }
    if (!shouldHandleAuthFailure(config)) {
      return Promise.reject(error)
    }
    const status = error.response?.status
    if (requestBearerToken(config) === (getToken() || '') && (status === 401 || isAuthFailureBody(error.response?.data))) {
      redirectToLoginOnAuthFailure(requestBearerToken(config))
    }
    return Promise.reject(error)
  })
}
