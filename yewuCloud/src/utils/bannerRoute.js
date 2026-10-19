/** 路由前缀与顶部平台 bannerId 的对应关系 */
const ROUTE_PREFIX_TO_BANNER = [
  { prefix: '/dashboard', id: '11' },
  { prefix: '/batteryStatus', id: '12' },
  { prefix: '/batteryTracking', id: '12' },
  { prefix: '/maintenance', id: '12' },
  { prefix: '/healthMonitoring', id: '13' },
  { prefix: '/alarms', id: '13' },
  { prefix: '/warnings', id: '13' },
  { prefix: '/historicalData', id: '13' },
  { prefix: '/basicInformation', id: '14' },
  { prefix: '/sites', id: '14' },
  { prefix: '/batteryFactory', id: '14' },
  { prefix: '/batteryData', id: '14' },
  { prefix: '/accounts', id: '14' },
  { prefix: '/basicData', id: '14' },
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
  11: '/dashboard',
  12: '/batteryStatus',
  13: '/healthMonitoring',
  14: '/basicInformation/warehouses'
}

export function getBannerHomePath(activeIndex) {
  return BANNER_HOME_PATH[String(activeIndex)] || null
}
