import { login, findSelf, getUserPermissions } from '@/api/user.js'
import { getToken, setToken, removeToken, setUserInfoStr, setUserId,
  setPageList, removeUsername, removeUserInfoStr, removeUserId,
  removeRoleId, removePageList, removeCjyToken } from '@/utils/auth.js'
import { resetRouter, asyncRoutes } from '@/router'
import { unique, objectSort } from '@/utils/utils.js'
import { buildRoutesFromPermissions } from '@/utils/build-permission-routes.js'
import { Message } from '@/utils/element-shim'
import { notifyBackendLogout } from '@/utils/session-logout'
import config from '@/config.js'
import { PLATFORM } from '@/constants/platform'

function clearStoredSession() {
  removeToken()
  removeUsername()
  removeUserInfoStr()
  removeUserId()
  removeRoleId()
  removePageList()
  removeCjyToken()
  // The gateway project stores its SSO state under the same production origin.
  localStorage.removeItem('access_token')
  localStorage.removeItem('login_username')
  localStorage.removeItem('login_account')
}

const state = {
  token: getToken(),
  roles: []
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
  }
}

const actions = {
  login({ commit }, userInfo) { // 登录
    return new Promise((resolve, reject) => {
      login(userInfo).then(response => {
        const access_token = response.return && response.return.token
        if (!access_token) {
          const errMsg = response.msg || '登录成功但未返回 token'
          Message({ message: errMsg, type: 'error', duration: 3 * 1000 })
          reject(new Error(errMsg))
          return
        }
        commit('SET_TOKEN', access_token) // 将 access_token 存入vuex中
        setToken(access_token) // 将 access_token 存入cookie中
        resolve(true)
      }).catch(error => {
        // 业务/网络错误提示已由 api.js 或 request.js 处理，避免重复弹窗
        reject(error)
      })
    })
  },

  getInfo({ commit }) { // 获取用户信息
    return new Promise((resolve, reject) => {
      findSelf().then(response => {
        const roles = ['manager'] // 当roles中包含'admin'时，则自动获取全部权限，在role.js中不需要配置'admin'
        const userInfo = response.return.user
        setUserInfoStr(JSON.stringify(userInfo))
        if (userInfo && userInfo.id != null) {
          setUserId(userInfo.id)
        }
        let pageNameList = null // 要显示的页面的名称组成的数组
        let routers = []
        const roleType = userInfo.isadmin
        const applyPermissions = (permiRes) => {
          if (roleType === 1) {
            routers = asyncRoutes
          } else {
            const permisList = objectSort((permiRes.return && permiRes.return.data) || [], 'parentId') // 排序
            const allPermissions = unique(permisList, 'id') // 去重
            pageNameList = allPermissions.filter(item => item && item.type === 1)
            routers = buildRoutesFromPermissions(asyncRoutes, pageNameList)
          }
          // DCSY 仅使用本文件顶部声明的锂硫电池路由，不再动态注入运维、报表等旧平台页面。
          let gysjkRoteIndex = null // 工艺数据库 的路由位置
          let gysjkRoteMsg = null // 工艺数据库 的路由数据
          let bbtjRoteIndex = null // 报表统计 的路由位置
          let bbtjRoteMsg = null // 报表统计 的路由数据
          routers.forEach((item, index) => {
            if (item.path === '/gysjk') {
              gysjkRoteIndex = index
              gysjkRoteMsg = item
            }
            if (item.path === '/bbtj') {
              bbtjRoteIndex = index
              bbtjRoteMsg = item
            }
            if (config.platform === PLATFORM.DCSY_INTRANET && config.port === '5002') {
              if (item.path === '/ytbData') { // 不显示 乡镇厂月填报
                item.hidden = true
              }
            }
          })

          if (gysjkRoteMsg && bbtjRoteMsg && (gysjkRoteIndex < bbtjRoteIndex)) { // 如果工艺数据库在报表统计的上面，则交换顺序
            routers[bbtjRoteIndex] = gysjkRoteMsg
            routers[gysjkRoteIndex] = bbtjRoteMsg
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

  async logout({ dispatch }) { // 退出登录
    notifyBackendLogout(getToken())
    await dispatch('sessionTimeout')
  },

  async sessionTimeout({ commit, dispatch }) { // 登录空闲超时
    commit('SET_TOKEN', '')
    commit('SET_ROLES', [])
    clearStoredSession()
    resetRouter()
    await Promise.all([
      dispatch('tagsView/delAllViews', null, { root: true }),
      dispatch('permission/resetRoutes', null, { root: true })
    ])
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
