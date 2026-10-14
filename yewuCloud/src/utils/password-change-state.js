export const PASSWORD_CHANGE_REQUIRED = 'PASSWORD_CHANGE_REQUIRED'
export const PASSWORD_CHANGE_PATH = '/modifyPas'

/** 只使用后端给出的密码状态，不在浏览器按日期推算季度。 */
export function readPasswordChangeStatus(body) {
  const payload = body?.return || body || {}
  return {
    passwordChangeRequired: payload.passwordChangeRequired === true || payload.errorCode === PASSWORD_CHANGE_REQUIRED,
    passwordChangeReason: String(payload.passwordChangeReason || ''),
    passwordExpiresAt: String(payload.passwordExpiresAt || '')
  }
}

export function isPasswordChangeRequired(body) {
  return readPasswordChangeStatus(body).passwordChangeRequired
}

/** 登录和本人信息可以成功返回改密状态；只有专用失败响应才进入拦截流程。 */
export function isPasswordChangeFailure(body) {
  return body?.return?.errorCode === PASSWORD_CHANGE_REQUIRED ||
    (Number(body?.code) === 403 && isPasswordChangeRequired(body))
}

export function passwordChangeMessage(reason) {
  if (reason === 'admin_reset') return '管理员已重置您的密码，请先修改密码后继续使用平台。'
  if (reason === 'password_unset') return '当前季度账号尚未完成密码更新，请先修改密码后继续使用平台。'
  if (reason === 'future_time') return '当前账号的密码更新时间异常，请先修改密码后继续使用平台。'
  return '已进入新的季度，请先修改密码后继续使用平台。'
}

/** 防止上一轮登录的慢请求影响重新登录后的会话。 */
export function requestBearerToken(config) {
  const headers = config?.headers
  const authorization = headers?.get?.('Authorization') || headers?.Authorization || headers?.authorization || ''
  return String(authorization).replace(/^Bearer\s+/i, '')
}
