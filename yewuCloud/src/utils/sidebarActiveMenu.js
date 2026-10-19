/**
 * 侧栏 el-menu 的 default-active，与 redirect 子路径对齐，避免高亮异常触发反复更新
 * @param {import('vue-router').RouteLocationNormalizedLoaded} route
 */
export function resolveSidebarActiveMenu(route) {
  if (!route) {
    return ''
  }
  const { meta, path } = route
  if (meta && meta.activeMenu) {
    return meta.activeMenu
  }
  return path || ''
}
