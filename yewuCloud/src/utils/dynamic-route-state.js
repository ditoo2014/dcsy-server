// 路由和权限守卫共享就绪状态，退出时同步复位，避免依赖整个登录或状态模块。
let dynamicRoutesReady = false

export function isDynamicRoutesReady() {
  return dynamicRoutesReady
}

export function markDynamicRoutesReady() {
  dynamicRoutesReady = true
}

export function resetDynamicRoutesReady() {
  dynamicRoutesReady = false
}
