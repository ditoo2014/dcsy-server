import request from '@/utils/request.js'
import { Message } from '@/utils/element-shim'
import { getToken } from '@/utils/auth.js'
import { expireSession } from '@/utils/session-expiration'
import { requirePasswordChange } from '@/utils/password-change.js'
import { isPasswordChangeRequired, requestBearerToken, PASSWORD_CHANGE_REQUIRED } from '@/utils/password-change-state.js'

export function get(url, data, requestMsg) {
  return http({
    method: 'GET',
    content_type: '',
    urlMsg: url,
    data,
    requestMsg
  })
}

export function JsonPost(url, data, requestMsg) {
  return http({
    method: 'POST',
    content_type: 'application/json; charset=UTF-8', // content-type：raw也是这个
    urlMsg: url,
    data,
    requestMsg
  })
}

export function wwwPost(url, data, requestMsg) {
  return http({
    method: 'POST',
    content_type: 'application/x-www-form-urlencoded',
    urlMsg: url,
    data,
    requestMsg
  })
}

export function formDataPost(url, data, requestMsg) {
  return http({
    method: 'POST',
    content_type: 'multipart/form-data',
    urlMsg: url,
    data,
    requestMsg
  })
}

export function JsonPut(url, data, requestMsg) {
  return http({
    method: 'PUT',
    content_type: 'JSON (application/json)',
    urlMsg: url,
    data,
    requestMsg
  })
}

export function Delete(url, data, requestMsg) {
  return http({
    method: 'DELETE',
    content_type: '',
    urlMsg: url,
    data,
    requestMsg
  })
}

export function http(params) {
  const { method, content_type, urlMsg, data, requestMsg } = params
  const url = urlMsg
  return new Promise((resolve, reject) => {
    const requestParams = {
      method,
      headers: {
        Authorization: `Bearer ${getToken()}`
      },
      url,
      data
    }
    if (content_type) {
      requestParams.headers['Content-Type'] = content_type
    }
    if (typeof FormData !== 'undefined' && data instanceof FormData) {
      delete requestParams.headers['Content-Type']
    }
    if (requestMsg) {
      for (const key in requestMsg) {
        requestParams[key] = requestMsg[key]
      }
    }
    request(requestParams).then(response => {
      const res = response
      if (requestParams.responseType === 'blob') {
        resolve(response)
      } else if (Number(res.code) !== 200) {
        const errMsg = res.msg || res.message || `请求失败(${res.code})`
        const businessError = new Error(errMsg)
        businessError.code = Number(res.code)
        businessError.errorCode = res.return && res.return.errorCode
        businessError.responseData = res
        if (isPasswordChangeRequired(res)) {
          businessError.errorCode = PASSWORD_CHANGE_REQUIRED
          requirePasswordChange(res, requestBearerToken(requestParams))
          reject(businessError)
          return
        }
        const sessionInvalid = Number(res.code) === 401 ||
          res.msg === 'token过期或未启用' ||
          res.msg === 'token验证失败' ||
          res.msg === 'auth header格式错误' ||
          res.msg === '登录超时，请重新登录'
        if (Number(res.code) === 401) {
          Message({
            message: `账号在其他地方登录，请重新登录`,
            type: 'warning',
            duration: 3 * 1000
          })
          reject(businessError)
        } else {
          if (businessError.errorCode !== 'PASSWORD_POLICY_VIOLATION') {
            Message({
              message: errMsg,
              type: 'error',
              duration: 3 * 1000
            })
          }
          reject(businessError)
        }
        if (sessionInvalid && requestBearerToken(requestParams) === (getToken() || '')) {
          expireSession()
        }
      } else {
        resolve(res)
      }
    }).catch(error => {
      reject(error)
    })
  })
}
