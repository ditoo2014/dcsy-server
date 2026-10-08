/** Re-export element-plus APIs with element-ui compatible names. */
import {
  ElMessage,
  ElMessageBox,
  ElNotification,
  ElLoading
} from 'element-plus'

const DEDUPE_WINDOW_MS = 3000

let lastMessage = ''
let lastType = ''
let lastAt = 0

function normalizeOptions(options) {
  if (typeof options === 'string' || typeof options === 'number') {
    return { message: String(options), type: 'info' }
  }
  const opts = options && typeof options === 'object' ? options : {}
  return {
    ...opts,
    message: opts.message == null ? '' : String(opts.message),
    type: opts.type || 'info'
  }
}

function shouldSkip(message, type) {
  const now = Date.now()
  if (message === lastMessage && type === lastType && now - lastAt < DEDUPE_WINDOW_MS) {
    return true
  }
  lastMessage = message
  lastType = type
  lastAt = now
  return false
}

function createDedupedMessage() {
  const wrapped = function Message(options) {
    const normalized = normalizeOptions(options)
    if (shouldSkip(normalized.message, normalized.type)) {
      return
    }
    return ElMessage(normalized)
  }

  ;['success', 'warning', 'info', 'error'].forEach(type => {
    wrapped[type] = (options) => {
      if (typeof options === 'string' || typeof options === 'number') {
        return wrapped({ message: String(options), type })
      }
      const opts = options && typeof options === 'object' ? options : {}
      return wrapped({ ...opts, type })
    }
  })

  wrapped.closeAll = (...args) => ElMessage.closeAll(...args)
  if (typeof ElMessage.close === 'function') {
    wrapped.close = (...args) => ElMessage.close(...args)
  }

  return wrapped
}

export const Message = createDedupedMessage()
export const MessageBox = ElMessageBox
export const Notification = ElNotification
export const Loading = ElLoading
