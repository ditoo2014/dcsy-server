/** 路由前缀与顶部平台 bannerId 的对应关系 */
const ROUTE_PREFIX_TO_BANNER = [
  { prefix: '/jsc', id: '11' },
  { prefix: '/dczt', id: '12' },
  { prefix: '/gjzz', id: '12' },
  { prefix: '/ywgl', id: '12' },
  { prefix: '/jkztjc', id: '13' },
  { prefix: '/gjxx', id: '13' },
  { prefix: '/yjxx', id: '13' },
  { prefix: '/lssjcx', id: '13' },
  { prefix: '/jcxxgl', id: '14' },
  { prefix: '/zdxxgl', id: '14' },
  { prefix: '/scxxgl', id: '14' },
  { prefix: '/dcsjgl', id: '14' },
  { prefix: '/rygl', id: '14' },
  { prefix: '/moniList', id: '3' },
  { prefix: '/jurisdCenter', id: '4' },
  { prefix: '/sszdgl', id: '5' },
  { prefix: '/sbxxgl', id: '5' },
  { prefix: '/gyjsbxj', id: '5' },
  { prefix: '/bbzx', id: '7' },
  { prefix: '/siteMana', id: '6' },
  { prefix: '/personMana', id: '6' },
  { prefix: '/setMana', id: '6' }
]

export function resolveBannerIdFromPath(path) {
  if (!path) {
    return null
  }
  const matched = ROUTE_PREFIX_TO_BANNER.find(
    ({ prefix }) => path === prefix || path.startsWith(`${prefix}/`)
  )
  return matched ? matched.id : null
}

export function getDefaultBannerId() {
  const stored = localStorage.getItem('bannerId')
  if (stored && stored !== '11') {
    return stored
  }
  return '14'
}

/** 顶部平台按钮对应的默认首页（须为路由表真实页面路径） */
export const BANNER_HOME_PATH = {
  11: '/jsc',
  12: '/dczt',
  13: '/jkztjc',
  14: '/jcxxgl/kfxxgl'
}

export function getBannerHomePath(activeIndex) {
  return BANNER_HOME_PATH[String(activeIndex)] || null
}
