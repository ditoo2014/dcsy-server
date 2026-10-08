/** 顶部 Banner 切换时跨 Layout 实例共享，避免 goRoute 与 setBannerId 重复 push */
let bannerNavLock = false

export function beginBannerNav() {
  bannerNavLock = true
}

export function endBannerNav() {
  bannerNavLock = false
}

export function isBannerNavLocked() {
  return bannerNavLock
}
