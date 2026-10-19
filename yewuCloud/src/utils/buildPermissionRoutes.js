/** 权限名称 → 路由 meta.title 别名（后端短名与前端路由标题不一致时） */
const PERMISSION_TITLE_ALIASES = {
  站点: '站点信息管理',
  库房: '库房信息管理',
  站点信息: '站点信息管理',
  库房信息: '库房信息管理',
  站点管理: '站点信息管理',
  生产信息管理: '电池出厂信息',
  健康状态监测管理: '健康状态监测',
  日志信息: '系统日志'
}

function filterPagePermissions(pageNameList) {
  return (pageNameList || []).filter(item => item && item.type === 1)
}

function expandPermissionNames(pageNameList) {
  const names = new Set()
  ;(pageNameList || []).forEach(item => {
    const permissionName = item && item.permissionName
    if (!permissionName) {
      return
    }
    names.add(permissionName)
    const alias = PERMISSION_TITLE_ALIASES[permissionName]
    if (alias) {
      names.add(alias)
    }
  })
  return names
}

function routeTitleMatchesPermission(title, permNames) {
  if (!title || !permNames || !permNames.size) {
    return false
  }
  if (permNames.has(title)) {
    return true
  }
  for (const name of permNames) {
    if (name && title.startsWith(name) && name.length >= 2) {
      return true
    }
  }
  return false
}

function cloneRouteNode(route) {
  const cloned = { ...route }
  if (route.meta) {
    cloned.meta = { ...route.meta, roles: ['manager'] }
  }
  if (route.children) {
    cloned.children = route.children.map(child => ({ ...child }))
  }
  return cloned
}

/**
 * 按用户页面权限递归过滤 asyncRoutes（支持 /basicInformationSection 等多级嵌套菜单）
 * @param {Array} routes
 * @param {Set<string>} permNames
 * @returns {Array}
 */
export function filterRoutesByPermissions(routes, permNames) {
  const result = []
  ;(routes || []).forEach(route => {
    const tmp = cloneRouteNode(route)
    if (tmp.children && tmp.children.length) {
      tmp.children = filterRoutesByPermissions(tmp.children, permNames)
    }
    const titleMatch = tmp.meta && routeTitleMatchesPermission(tmp.meta.title, permNames)
    const hasVisibleChildren = tmp.children && tmp.children.length > 0
    if (titleMatch || hasVisibleChildren) {
      if (hasVisibleChildren) {
        result.push(tmp)
      } else if (titleMatch) {
        result.push(tmp)
      }
    }
  })
  return result
}

/**
 * @param {Array} asyncRoutes
 * @param {Array} pageNameList getUserPermissions 返回的去重权限列表
 * @returns {Array}
 */
export function buildRoutesFromPermissions(asyncRoutes, pageNameList) {
  const pagePermissions = filterPagePermissions(pageNameList)
  const permNames = expandPermissionNames(pagePermissions)
  return filterRoutesByPermissions(asyncRoutes, permNames)
}
