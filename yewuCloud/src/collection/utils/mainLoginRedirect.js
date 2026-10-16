import { expireSession } from '@/utils/session-expiration'

export { LAST_ACTIVITY_KEY, SESSION_TIMEOUT_EVENT_KEY } from '@/utils/session-events'

/** 采集界面复用主平台退出流程，统一清理状态和发送带会话标识的通知。 */
export function redirectToMainLogin(options = {}) {
  return expireSession({ ...options, redirectPath: '/collection' })
}
