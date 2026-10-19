import { PLATFORM } from '@/constants/platform'
import { normalizeLegacyRoutePath } from './legacyRoutePaths.js'

// 本文件选择登录落地页时也要调用，先导入，再保留路由守卫使用的导出入口。
export { normalizeLegacyRoutePath }

/** 登录后不作为默认落地页的路径（仍可手动从菜单、顶部 Banner 进入） */
const LOGIN_EXCLUDED_PATHS = [
  '/siteMana'
]

/** 信创开发环境登录不再硬编码优先路径，由权限路由表顺序决定 */
const XC_LOGIN_CANDIDATES = []

/** 内网锂硫登录候选 */
const INTRANET_LOGIN_CANDIDATES = [
  '/home'
]

function isLoginExcludedPath(path) {
  if (LOGIN_EXCLUDED_PATHS.includes(path)) {
    return true
  }
  return false
}

/**
 * 导航到 path 时是否会 redirect 到 targetPath（用于守卫打断重定向死循环）
 * @param {import('vue-router').Router} router
 * @param {string} path
 * @param {string} targetPath
 */
export function wouldRedirectTo(router, path, targetPath) {
  if (!router || !path || !targetPath) {
    return false
  }
  try {
    const resolved = router.resolve({ path })
    for (let i = resolved.matched.length - 1; i >= 0; i--) {
      const record = resolved.matched[i]
      if (typeof record.redirect === 'string') {
        return record.redirect === targetPath
      }
      if (record.redirect && typeof record.redirect === 'object' && record.redirect.path) {
        return record.redirect.path === targetPath
      }
    }
  } catch {
    return false
  }
  return false
}

/** 是否为信创锂硫业务相关路由节点 */
function isDcsyTraceRoute(route) {
  if (!route) {
    return false
  }
  if (route.basicInformationSection || route.batteryHealthSection || route.lifecycleSection || route.path === '/dashboard') {
    return true
  }
  return (route.children || []).some(child => isDcsyTraceRoute(child))
}

/**
 * 信创环境登录时仅使用锂硫业务路由，避免落到 /siteMana 等系统管理页
 * @param {import('vue-router').RouteRecordRaw[]} accessRoutes
 * @param {string} [platform]
 */
export function filterAccessRoutesForLogin(accessRoutes, platform) {
  if (platform !== PLATFORM.DCSY_XC || !Array.isArray(accessRoutes)) {
    return accessRoutes || []
  }
  return accessRoutes.filter(route => isDcsyTraceRoute(route))
}

/**
 * 路由是否可访问（已注册且能解析到带页面的匹配记录）
 * @param {import('vue-router').Router} router
 * @param {string} path
 */
function matchedHasPageComponent(record) {
  if (!record) {
    return false
  }
  const components = record.components || {}
  return Boolean(components.default || Object.keys(components).length > 0)
}

export function isRouteAccessible(router, path) {
  if (!router || !path) {
    return false
  }
  try {
    return resolvedRouteHasPage(router, { path }, new Set())
  } catch {
    return false
  }
}

function resolvedRouteHasPage(router, target, visited) {
  const resolved = router.resolve(target)
  if (!resolved.matched.length || visited.has(resolved.fullPath)) {
    return false
  }
  visited.add(resolved.fullPath)
  const leaf = resolved.matched[resolved.matched.length - 1]
  // 分组本身不是页面，只有重定向确实能到达页面才允许导航，并阻断循环。
  if (leaf.redirect) {
    const redirect = typeof leaf.redirect === 'function' ? leaf.redirect(resolved) : leaf.redirect
    return Boolean(redirect) && resolvedRouteHasPage(router, redirect, visited)
  }
  // 名称和外层 Layout 都不能代替实际页面；权限过滤后的空布局也应拒绝。
  return !leaf.meta?.routeContainer && !leaf.children?.length && matchedHasPageComponent(leaf)
}

/**
 * 将不可访问路径纠正到所属模块的默认可访问页（如 /basicInformationSection → /basicInformationSection/kfxxgl）
 * @param {import('vue-router').Router} router
 * @param {string} path
 */
export function resolveSectionHomePath(router, path) {
  if (!router || !path) {
    return null
  }
  const prefix = getRouteSectionPrefix(path)
  if (!prefix) {
    return null
  }
  if (isRouteAccessible(router, path)) {
    return path
  }
  if (isRouteAccessible(router, prefix)) {
    return prefix
  }
  try {
    const resolved = router.resolve({ path: prefix })
    for (let i = resolved.matched.length - 1; i >= 0; i--) {
      const record = resolved.matched[i]
      if (typeof record.redirect === 'string' && isRouteAccessible(router, record.redirect)) {
        return record.redirect
      }
    }
  } catch {
    return null
  }
  return null
}

/** 取路径所属一级模块前缀，如 /jkztjc/jkztjc -> /jkztjc */
function getRouteSectionPrefix(path) {
  if (!path) {
    return null
  }
  const segment = path.split('/').filter(Boolean)[0]
  return segment ? `/${segment}` : null
}

/**
 * 从权限路由表中收集可导航路径
 * @param {import('vue-router').RouteRecordRaw[]} routes
 * @param {string} parentPath
 * @returns {string[]}
 */
export function collectAccessiblePaths(routes, parentPath = '') {
  if (!Array.isArray(routes)) {
    return []
  }
  const paths = []

  routes.forEach(route => {
    if (!route || route.hidden) {
      return
    }
    if (typeof route.redirect === 'string' && !isLoginExcludedPath(route.redirect)) {
      paths.push(route.redirect)
    }

    // 空路径子页复用父级地址，否则只有单页模块权限的账号会漏掉默认首页。
    let fullPath = route.path || parentPath
    if (route.path && !fullPath.startsWith('/')) {
      const base = parentPath.replace(/\/$/, '')
      fullPath = `${base}/${fullPath}`
    }

    const children = route.children || []
    if (children.length) {
      paths.push(...collectAccessiblePaths(children, fullPath || parentPath))
      return
    }

    if (fullPath && (route.component || route.name) && !isLoginExcludedPath(fullPath)) {
      paths.push(fullPath)
    }
  })

  return paths
}

/**
 * @param {string} platform
 * @returns {string[]}
 */
export function getPreferredLoginPaths(platform) {
  if (platform === PLATFORM.DCSY_XC) {
    return XC_LOGIN_CANDIDATES.slice()
  }
  if (platform === PLATFORM.DCSY_INTRANET) {
    return INTRANET_LOGIN_CANDIDATES.slice()
  }
  return ['/home']
}

/**
 * 解析登录/守卫应使用的路径：目标路径不存在时依次尝试候选，最后用权限表首条可访问路由
 * @param {import('vue-router').Router} router
 * @param {string} targetPath 期望跳转路径
 * @param {import('vue-router').RouteRecordRaw[]} accessRoutes 当前用户可访问的动态路由
 * @param {{ platform?: string }} [options]
 * @returns {string}
 */
export function resolveAccessiblePath(router, targetPath, accessRoutes, options = {}) {
  const { platform, skipLoginPreferred } = options
  const loginRoutes = filterAccessRoutesForLogin(accessRoutes, platform)
  const candidates = []
  const sectionPrefix = skipLoginPreferred ? getRouteSectionPrefix(targetPath) : null

  const pushCandidate = (path) => {
    if (path && !isLoginExcludedPath(path) && !candidates.includes(path)) {
      candidates.push(path)
    }
  }

  pushCandidate(normalizeLegacyRoutePath(targetPath))
  if (sectionPrefix) {
    pushCandidate(sectionPrefix)
  }
  if (!skipLoginPreferred) {
    getPreferredLoginPaths(platform).forEach(pushCandidate)
  }
  const collectedPaths = collectAccessiblePaths(loginRoutes)
  if (sectionPrefix) {
    collectedPaths
      .filter(path => path === sectionPrefix || path.startsWith(`${sectionPrefix}/`))
      .forEach(pushCandidate)
  }
  collectedPaths.forEach(pushCandidate)
  pushCandidate('/home')

  for (const path of candidates) {
    if (isRouteAccessible(router, path)) {
      return path
    }
  }

  return '/home'
}
