import { login, findSelf, getUserPermissions } from '@/api/user.js'
import { getToken, setToken, removeToken, setUserInfoStr, setUserId,
  setPageList, removeUsername, removeUserInfoStr, removeUserId,
  removeRoleId, removePageList, removeCollectionToken } from '@/utils/auth.js'
import { resetRouter, asyncRoutes } from '@/router'
import { unique, objectSort } from '@/utils/utils.js'
import { buildRoutesFromPermissions } from '@/utils/build-permission-routes.js'
import { Message } from '@/utils/element-shim'
import { notifyBackendLogout } from '@/utils/session-logout'
import config from '@/config.js'
import { PLATFORM } from '@/constants/platform'
import { readPasswordChangeStatus } from '@/utils/password-change-state.js'
import { useAuthStore } from '@/collection/stores/auth.js'
import { useTagsViewStore } from '@/collection/stores/tagsView.js'

function changedSessionError() {
  return Object.assign(new Error('登录会话已更新，已忽略旧请求'), { errorCode: 'SESSION_CHANGED' })
}

function clearStoredSession() {
  removeToken()
  removeUsername()
  removeUserInfoStr()
  removeUserId()
  removeRoleId()
  removePageList()
  removeCollectionToken()
  // The gateway project stores its SSO state under the same production origin.
  localStorage.removeItem('access_token')
  localStorage.removeItem('login_username')
  localStorage.removeItem('login_account')
}

const state = {
  token: getToken(),
  roles: [],
  passwordChangeRequired: false,
  passwordChangeReason: '',
  passwordExpiresAt: ''
}

const mutations = {
  SET_TOKEN: (state, token) => { // 设置 token
    state.token = token
  },
  SET_ROLETYPE: (state, roleType) => { // 设置角色类型
    state.roleType = roleType
  },
  SET_ROLES: (state, roles) => { // 设置角色
    state.roles = roles
  },
  SET_MANUFACTOR: (state, manufactor) => { // 设置角色
    state.manufactor = manufactor
  },
  SET_AVATAR: (state, avatar) => { // 设置头像图片
    state.avatar = avatar
  },
  SET_CODENAME: (state, codeName) => { // 设置头像图片
    state.codeName = codeName
  },
  SET_PASSWORD_STATUS: (state, status) => {
    Object.assign(state, readPasswordChangeStatus(status))
  }
}

const actions = {
  async login({ commit, dispatch }, userInfo) { // 登录
    const requestedToken = getToken()
    const response = await login(userInfo)
    if (getToken() !== requestedToken) throw changedSessionError()
    const access_token = response.return && response.return.token
    if (!access_token) {
      const errMsg = response.msg || '登录成功但未返回 token'
      Message({ message: errMsg, type: 'error', duration: 3 * 1000 })
      throw new Error(errMsg)
    }
    await dispatch('clearBusinessAccess')
    if (getToken() !== requestedToken) throw changedSessionError()
    commit('SET_PASSWORD_STATUS', response)
    commit('SET_TOKEN', access_token)
    setToken(access_token)
    return true
  },

  getInfo({ commit, dispatch }) { // 获取用户信息
    return new Promise((resolve, reject) => {
      const requestedToken = getToken()
      findSelf().then(async response => {
        if (getToken() !== requestedToken) throw changedSessionError()
        commit('SET_PASSWORD_STATUS', response)
        const roles = ['manager'] // 当roles中包含'admin'时，则自动获取全部权限，在role.js中不需要配置'admin'
        const userInfo = response.return.user
        setUserInfoStr(JSON.stringify(userInfo))
        if (userInfo && userInfo.id != null) {
          setUserId(userInfo.id)
        }
        // 受限登录仅加载本人信息，不请求业务权限或注入业务菜单。
        if (readPasswordChangeStatus(response).passwordChangeRequired) {
          await dispatch('requirePasswordChange', response)
          if (getToken() !== requestedToken) throw changedSessionError()
          resolve({ roles: [], routers: [], ...readPasswordChangeStatus(response) })
          return
        }
        let pageNameList = null // 要显示的页面的名称组成的数组
        let routers = []
        const roleType = userInfo.isadmin
        const applyPermissions = (permiRes) => {
          if (getToken() !== requestedToken) {
            reject(changedSessionError())
            return
          }
          if (readPasswordChangeStatus(permiRes).passwordChangeRequired) {
            dispatch('requirePasswordChange', permiRes).then(() => {
              if (getToken() !== requestedToken) throw changedSessionError()
              resolve({ roles: [], routers: [], ...readPasswordChangeStatus(permiRes) })
            }).catch(reject)
            return
          }
          if (roleType === 1) {
            routers = asyncRoutes
          } else {
            const permisList = objectSort((permiRes.return && permiRes.return.data) || [], 'parentId') // 排序
            const allPermissions = unique(permisList, 'id') // 去重
            pageNameList = allPermissions.filter(item => item && item.type === 1)
            routers = buildRoutesFromPermissions(asyncRoutes, pageNameList)
          }
          // DCSY 仅使用本文件顶部声明的锂硫电池路由，不再动态注入运维、报表等旧平台页面。
          let processDataRouteIndex = null // 工艺数据库 的路由位置
          let processDataRoute = null // 工艺数据库 的路由数据
          let reportStatisticsRouteIndex = null // 报表统计 的路由位置
          let reportStatisticsRoute = null // 报表统计 的路由数据
          routers.forEach((item, index) => {
            if (item.path === '/gysjk') {
              processDataRouteIndex = index
              processDataRoute = item
            }
            if (item.path === '/bbtj') {
              reportStatisticsRouteIndex = index
              reportStatisticsRoute = item
            }
            if (config.platform === PLATFORM.DCSY_INTRANET && config.port === '5002') {
              if (item.path === '/ytbData') { // 不显示 乡镇厂月填报
                item.hidden = true
              }
            }
          })

          if (processDataRoute && reportStatisticsRoute && (processDataRouteIndex < reportStatisticsRouteIndex)) { // 如果工艺数据库在报表统计的上面，则交换顺序
            routers[reportStatisticsRouteIndex] = processDataRoute
            routers[processDataRouteIndex] = reportStatisticsRoute
          }
          if (import.meta.env.DEV) { // 开发模式
            roles.push('developer')
          }
          commit('SET_AVATAR', 'https://wpimg.wallstcn.com/f778738c-e4f8-4870-b634-56703b4acafe.gif') // 设置用户头像
          commit('SET_ROLES', roles)
          let pageList = JSON.parse(JSON.stringify(routers)) // 配置被保存的页面
          if (pageNameList) {
            pageNameList.forEach(item => {
              if (
                item.permissionName === '全生命周期管理' ||
                item.permissionName === '电池健康管理' ||
                item.permissionName === '基础信息管理' ||
                item.permissionName === '驾驶舱'
              ) {
                pageList.push(item)
              }
            })
          } else {
            pageList = [roleType]
          }
          setPageList(JSON.stringify(pageList))
          resolve({
            roles,
            routers
          })
        }

        getUserPermissions().then(permiRes => {
          applyPermissions(permiRes)
        }).catch(error => {
          if (getToken() !== requestedToken) {
            reject(changedSessionError())
            return
          }
          if (error?.errorCode === 'PASSWORD_CHANGE_REQUIRED') {
            dispatch('requirePasswordChange', error.responseData).then(() => {
              if (getToken() !== requestedToken) throw changedSessionError()
              resolve({ roles: [], routers: [], ...readPasswordChangeStatus(error.responseData) })
            }).catch(reject)
            return
          }
          if (roleType === 1) {
            applyPermissions(null)
            return
          }
          const errMsg = (error && error.message) || error || '获取用户权限失败'
          Message({
            message: typeof errMsg === 'string' ? errMsg : '获取用户权限失败',
            type: 'error',
            duration: 3 * 1000
          })
          reject(error)
        })
      }).catch(error => {
        if (getToken() !== requestedToken) {
          reject(changedSessionError())
          return
        }
        const errMsg = (error && error.message) || error || '获取用户信息失败'
        Message({
          message: typeof errMsg === 'string' ? errMsg : '获取用户信息失败',
          type: 'error',
          duration: 3 * 1000
        })
        reject(error)
      })
    })
  },

  async checkPasswordChange({ commit, dispatch }) {
    const requestedToken = getToken()
    const response = await findSelf()
    if (!requestedToken || getToken() !== requestedToken) throw changedSessionError()
    commit('SET_PASSWORD_STATUS', response)
    const userInfo = response.return?.user
    if (userInfo) {
      setUserInfoStr(JSON.stringify(userInfo))
      if (userInfo.id != null) setUserId(userInfo.id)
    }
    const status = readPasswordChangeStatus(response)
    if (status.passwordChangeRequired) await dispatch('requirePasswordChange', response)
    if (getToken() !== requestedToken) throw changedSessionError()
    return status
  },

  async requirePasswordChange({ commit, dispatch }, body) {
    commit('SET_PASSWORD_STATUS', body)
    await dispatch('clearBusinessAccess')
  },

  async clearBusinessAccess({ commit, dispatch }) {
    // 到期时清业务菜单、标签及采集权限，保留认证令牌供本人改密。
    commit('SET_ROLES', [])
    removePageList()
    removeCollectionToken()
    useAuthStore().clearAccess()
    useTagsViewStore().clearVisitedViews()
    resetRouter()
    await Promise.all([
      dispatch('tagsView/delAllViews', null, { root: true }),
      dispatch('permission/resetRoutes', null, { root: true })
    ])
  },

  async logout({ dispatch }) { // 退出登录
    notifyBackendLogout(getToken())
    await dispatch('sessionTimeout')
  },

  async sessionTimeout({ commit, dispatch }) { // 登录空闲超时
    commit('SET_TOKEN', '')
    commit('SET_PASSWORD_STATUS', {})
    useAuthStore().logout()
    clearStoredSession()
    await dispatch('clearBusinessAccess')
  },

  // remove token
  resetToken({ dispatch }) {
    return dispatch('sessionTimeout')
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
