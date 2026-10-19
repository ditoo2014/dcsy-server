import { isExternal } from '@/utils/validate'

/**
 * 拼接路由路径（使用 POSIX 规则）。
 * Windows 下 path.resolve('/a', '/b') 会得到盘符路径（如 E:\b），导致 vue-router 无法跳转。
 */
export function resolveRoutePath(basePath = '', routePath = '') {
  if (isExternal(routePath)) {
    return routePath
  }
  if (isExternal(basePath)) {
    return basePath
  }
  // 绝对子路径已经包含完整层级，不能再次拼接父路径。
  const joined = routePath.startsWith('/')
    ? routePath
    : `${basePath || '/'}/${routePath || ''}`
  const result = []
  joined.split('/').forEach(part => {
    if (!part || part === '.') return
    if (part === '..') result.pop()
    else result.push(part)
  })
  return `/${result.join('/')}`
}
