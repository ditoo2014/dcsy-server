import axios from 'axios'
import { Message } from '@/utils/element-shim'
import { expireSession } from '@/utils/session-expiration'
import { requirePasswordChange } from '@/utils/password-change.js'
import { isPasswordChangeFailure, requestBearerToken, PASSWORD_CHANGE_REQUIRED } from '@/utils/password-change-state.js'
import { getToken } from '@/utils/auth.js'

async function passwordFailure(body, config, readBlob = false) {
  // 导出下载接口的错误也可能以 Blob 返回，需要先读取中文错误和专用标识。
  if (typeof Blob !== 'undefined' && body instanceof Blob) {
    if (!readBlob) return null
    try {
      body = JSON.parse(await body.text())
    } catch {
      return null
    }
  }
  if (!isPasswordChangeFailure(body)) return null
  requirePasswordChange(body, requestBearerToken(config))
  const error = new Error(body.msg || '请先修改密码后继续使用平台')
  error.code = 403
  error.errorCode = PASSWORD_CHANGE_REQUIRED
  error.responseData = body
  return error
}

// 创建axios实例
const service = axios.create({
  timeout: 30000 // 请求延时
})

// 请求拦截
service.interceptors.request.use( // 发出请求前执行
  config => {
    return config
  },
  error => {
    // 处理请求错误
    console.log('request error:' + error) // 用于调试
    return Promise.reject('request error:' + error)
  }
)

// response interceptor
service.interceptors.response.use(
  /**
  *如果要获取HTTP信息，如头或状态
  *请返回响应=>响应
  */
  /**
  *通过自定义代码确定请求状态
  *这只是一个例子
  *您还可以通过HTTP状态代码判断状态
  */

  async response => {
    const contentType = response.headers?.get?.('Content-Type') || response.headers?.['content-type'] || ''
    const failure = await passwordFailure(response.data, response.config, String(contentType).includes('json'))
    if (failure) return Promise.reject(failure)
    if (response.config && response.config.responseType === 'blob') {
      return response
    }
    return response.data
  },
  async error => {
    const failure = await passwordFailure(error.response?.data, error.config, true)
    if (failure) return Promise.reject(failure)
    const status = error.response && error.response.status
    if (status === 401) {
      if (requestBearerToken(error.config) !== (getToken() || '')) return Promise.reject(error)
      Message({
        message: '登录已过期，请重新登录',
        type: 'warning',
        duration: 3 * 1000
      })
      expireSession()
    } else {
      const serverMsg = error.response?.data?.msg ||
        error.response?.data?.message ||
        error.message
      Message({
        message: serverMsg || '网络请求失败',
        type: 'error',
        duration: 3 * 1000
      })
    }
    return Promise.reject(error)
  }
)

export default service
