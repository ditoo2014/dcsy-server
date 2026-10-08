import axios from 'axios'
import { Message } from '@/utils/element-shim'
import { expireSession } from '@/utils/session-expiration'

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

  response => {
    if (response.config && response.config.responseType === 'blob') {
      return response
    }
    return response.data
  },
  error => {
    const status = error.response && error.response.status
    if (status === 401) {
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
