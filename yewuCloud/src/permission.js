import router from './router'
import store from './store'
import { Message } from '@/utils/elementShim'
import NProgress from 'nprogress' // 一个进度条的插件
import 'nprogress/nprogress.css' // 进度条插件样式
import { getToken } from '@/utils/auth.js' // 封装了从cookie中获取token的方法
import getPageTitle from '@/utils/getPageTitle'
import config from '@/config.js'
import { PLATFORM } from '@/constants/platform'
import {
  isRouteAccessible,
  normalizeLegacyRoutePath,
  resolveAccessiblePath,
  resolveSectionHomePath,
  wouldRedirectTo
} from '@/utils/resolveAccessibleRoute.js'
import { resolveBannerIdFromPath } from '@/utils/bannerRoute.js'
import { ensureTolinkAccess } from '@/collection/utils/authAccess.js'
import { useAuthStore } from '@/collection/stores/auth.js'
import { isDynamicRoutesReady, markDynamicRoutesReady, resetDynamicRoutesReady } from '@/utils/dynamicRouteState.js'
import { PASSWORD_CHANGE_PATH, PASSWORD_CHANGE_REQUIRED } from '@/utils/passwordChangeState.js'

// 保留登录页使用的导出入口，路由清理也能同步复位该状态。
export { markDynamicRoutesReady, resetDynamicRoutesReady }

function isRouteAccessibleForGuard(path) {
  return isRouteAccessible(router, path)
}

NProgress.configure({ showSpinner: false }) // 是否有转圈效果

const whiteList = ['/login', '/auth-redirect'] // 没有重定向白名单

/** 单次导航链路上的守卫重定向次数，防止 redirect 配置异常时栈溢出 */
let guardRedirectCount = 0
const GUARD_REDIRECT_LIMIT = 12

function syncBannerIdByPath(path) {
  const bannerId = resolveBannerIdFromPath(path)
  if (bannerId) {
    localStorage.setItem('bannerId', bannerId)
  }
}

function guardNext(next, target) {
  if (target === undefined || target === true || target === false) {
    guardRedirectCount = 0
    next(target)
    return
  }
  guardRedirectCount += 1
  if (guardRedirectCount > GUARD_REDIRECT_LIMIT) {
    guardRedirectCount = 0
    console.error('[router] 导航重定向次数过多，已中止:', target)
    next('/home')
    NProgress.done()
    return
  }
  next(target)
}

router.beforeEach(async(to, from, next) => {
  // eslint-disable-next-line
  if (window._hmt) {
    if (to.path) {
      // eslint-disable-next-line
      window._hmt.push(['_trackPageview', '/#' + to.fullPath])
    }
  }

  NProgress.start() // 开始进度条
  document.title = getPageTitle(to.meta.title) // 设置页面标题
  if (to.path === '/running' && to.query.row) { // 监控运行详细画面
    try {
      const runningDetails = JSON.parse(to.query.row)
      if (runningDetails && runningDetails.equipmentName) {
        document.title = `详细运行画面-${runningDetails.equipmentName}`
      }
    } catch {
      // 非法 row 参数时不阻断导航
    }
  }
  const hasToken = getToken() // 确定用户是否已登录
  if (hasToken) { // 如果token存在
    const abortChangedSession = () => {
      if (getToken() === hasToken) return false
      // 旧导航的慢响应不能清理或限制刚刚重新登录的新账号。
      guardNext(next, false)
      NProgress.done()
      return true
    }
    // 同一会话短暂复用校验，改密页强制刷新；刷新页面及后端到期时间仍会触发重新核验。
    if (to.path !== '/login') {
      try {
        const status = await store.dispatch('user/checkPasswordChange', { force: to.path === PASSWORD_CHANGE_PATH })
        if (abortChangedSession()) return
        if (status.passwordChangeRequired) {
          guardNext(next, to.path === PASSWORD_CHANGE_PATH ? undefined : { path: PASSWORD_CHANGE_PATH, replace: true })
          NProgress.done()
          return
        }
      } catch (error) {
        if (abortChangedSession()) return
        if (error?.errorCode === PASSWORD_CHANGE_REQUIRED) {
          await store.dispatch('user/requirePasswordChange', error.responseData)
          if (abortChangedSession()) return
          guardNext(next, to.path === PASSWORD_CHANGE_PATH ? undefined : { path: PASSWORD_CHANGE_PATH, replace: true })
        } else if (Number(error?.response?.status || error?.code) === 401) {
          await store.dispatch('user/resetToken')
          guardNext(next, { path: '/login', query: { redirect: to.fullPath }, replace: true })
        } else {
          // 服务暂时不可用时拒绝本次导航，保留会话供恢复后重试。
          // 会话缓存暂时故障时也不追加导航提示，仍中止导航并保留会话。
          const serverMessage = error?.response?.data?.msg || error?.responseData?.msg || error?.message
          if (serverMessage !== '暂时无法校验登录状态，请稍后重试') {
            Message.error('暂时无法核验账号密码状态，请稍后重试或刷新页面')
          }
          guardNext(next, false)
        }
        NProgress.done()
        return
      }
    }
    const legacyPath = normalizeLegacyRoutePath(to.path)
    if (legacyPath !== to.path) {
      syncBannerIdByPath(legacyPath)
      guardNext(next, { path: legacyPath, replace: true, query: to.query, hash: to.hash })
      NProgress.done()
      return
    }
    if (to.matched.some(route => route.meta.collectionCloud)) {
      try {
        await ensureTolinkAccess(hasToken)
        if (abortChangedSession()) return
        const requiredPermission = String(to.meta.collectionPermission || '')
        if (requiredPermission && !useAuthStore().hasAnyPermission(requiredPermission)) {
          guardNext(next, { path: '/collection', replace: true })
          NProgress.done()
          return
        }
      } catch (error) {
        if (abortChangedSession()) return
        if (error?.errorCode === PASSWORD_CHANGE_REQUIRED) {
          await store.dispatch('user/requirePasswordChange', error.responseData)
          if (abortChangedSession()) return
          guardNext(next, { path: PASSWORD_CHANGE_PATH, replace: true })
          NProgress.done()
          return
        }
        // 采集云权限读取失败时仍允许进入首页，具体接口继续由后端鉴权。
        console.error('加载采集云权限失败:', error)
      }
    }
    if (to.path === '/login') { // 如果该路由指向登录页，则重定向到主页
      guardNext(next)
      NProgress.done()
    } else { // 如果该路由不指向登录页
      const hasRoles = store.getters.roles && store.getters.roles.length > 0 // 确定vuex中是否有角色
      if (hasRoles) { // 如果有角色
        if (to.path === '/bbzx/gysjk/gczbsjk' || to.path === '/bbzx/gysjk/hysjk' || to.path === '/bbzx/gysjk/scrbb') {
          /*
            /bbzx/gysjk/gczbsjk 对应 过程指标数据库 界面
            /bbzx/gysjk/hysjk 对应 化验数据库 界面
            /bbzx/gysjk/scrbb 对应 生产日报表 界面
          */
          if (localStorage.getItem('bbzxPageMsg')) {
            localStorage.setItem('bannerId', '7') // 报表中心界面
          }
        } else if (to.path === '/sszdgl/gysstz' || to.path === '/sszdgl/zdda') { // 设施运行数据
          if (localStorage.getItem('sszdglPageMsg')) {
            localStorage.setItem('bannerId', '5') // 运维中心界面
          }
        }
        if (isDynamicRoutesReady() && !isRouteAccessibleForGuard(to.path)) {
          const sectionHome = resolveSectionHomePath(router, to.path)
          if (sectionHome && sectionHome !== to.path) {
            syncBannerIdByPath(sectionHome)
            guardNext(next, { path: sectionHome, replace: true, query: to.query })
            return
          }
          const accessRoutes = store.state.permission.addRoutes || []
          const accessiblePath = resolveAccessiblePath(router, to.path, accessRoutes, {
            platform: config.platform,
            skipLoginPreferred: true
          })
          if (accessiblePath !== to.path) {
            if (wouldRedirectTo(router, accessiblePath, to.path)) {
              guardNext(next)
              return
            }
            syncBannerIdByPath(accessiblePath)
            guardNext(next, { path: accessiblePath, replace: true, query: to.query })
            return
          }
        }
        guardNext(next)
      } else { // 如果没有角色
        if (isDynamicRoutesReady()) {
          guardNext(next)
          return
        }
        try {
          // 获取用户信息
          // 角色必须是对象数组！例如： ['admin'] or ,['developer','editor']
          const roleInfo = await store.dispatch('user/getInfo')
          if (abortChangedSession()) return
          if (roleInfo.passwordChangeRequired) {
            guardNext(next, { path: PASSWORD_CHANGE_PATH, replace: true })
            NProgress.done()
            return
          }
          if (roleInfo.roles.length !== 0) { // 如果有获取到角色
            const accessRoutes = await store.dispatch('permission/generateRoutes', roleInfo) // 生成可访问的路由表
            if (abortChangedSession()) return
            // 动态添加可访问路由表
            accessRoutes.forEach(route => router.addRoute(route))
            markDynamicRoutesReady()

            // 信创环境登录后默认不落到 /home，但若目标已是可访问的常量页（如 equipManaPage），须保留 to.path
            const preferred = isRouteAccessibleForGuard(to.path)
              ? to.path
              : (config.platform === PLATFORM.DCSY_XC ? '' : to.path)
            const accessiblePath = resolveAccessiblePath(router, preferred, accessRoutes, {
              platform: config.platform
            })
            syncBannerIdByPath(accessiblePath)

            if (accessiblePath === to.path) {
              guardNext(next, { ...to, replace: true })
            } else {
              guardNext(next, { path: accessiblePath, replace: true, query: to.query })
            }
          } else { // 如果没有获取到角色，则从定向到登录页
            guardNext(next, `/login?redirect=${to.path}`)
            NProgress.done()
          }
        } catch (error) {
          if (abortChangedSession()) return
          if (error?.errorCode === PASSWORD_CHANGE_REQUIRED) {
            await store.dispatch('user/requirePasswordChange', error.responseData)
            if (abortChangedSession()) return
            guardNext(next, { path: PASSWORD_CHANGE_PATH, replace: true })
            NProgress.done()
            return
          }
          if (Number(error?.response?.status || error?.code) === 401) {
            await store.dispatch('user/resetToken')
            guardNext(next, `/login?redirect=${to.path}`)
          } else {
            Message.error('暂时无法加载账号权限，请稍后重试或刷新页面')
            guardNext(next, false)
          }
          NProgress.done()
        }
      }
    }
  } else { // 没有token
    resetDynamicRoutesReady()
    if (whiteList.indexOf(to.path) !== -1) {
      // 在免费登录白名单，直接去
      guardNext(next)
    } else {
      // 其他没有访问权限的页面将重定向到登录页面
      guardNext(next, `/login?redirect=${to.path}`)
      NProgress.done()
    }
  }
})

router.afterEach(() => {
  // 完成进度条
  NProgress.done()
})
