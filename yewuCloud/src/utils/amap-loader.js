import AMapLoader from '@amap/amap-jsapi-loader'

// 沿用配置文件中的高德参数；不在组件中固化账号配置或重复加载地图脚本。
const plugins = ['AMap.DistrictSearch', 'AMap.Geocoder', 'AMap.PlaceSearch', 'AMap.AutoComplete']
let pendingLoad = null

export function loadBusinessMap() {
  if (pendingLoad) return pendingLoad
  const key = String(import.meta.env.VITE_AMAP_KEY || '').trim()
  if (!key) return Promise.reject(new Error('未配置高德地图，请在前端配置文件填写 VITE_AMAP_KEY'))

  const securityJsCode = String(import.meta.env.VITE_AMAP_SECURITY_CODE || '').trim()
  if (securityJsCode) {
    window._AMapSecurityConfig = { ...(window._AMapSecurityConfig || {}), securityJsCode }
  }
  pendingLoad = AMapLoader.load({ key, version: '2.0', plugins }).catch(() => {
    // 网络失败后允许用户重新打开页面重试，不永久缓存失败的 Promise。
    pendingLoad = null
    throw new Error('地图加载失败，请检查网络、高德配置和域名白名单')
  })
  return pendingLoad
}
