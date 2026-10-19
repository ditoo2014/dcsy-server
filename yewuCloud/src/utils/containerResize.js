/**
 * 监听容器实际尺寸，合并窗口缩放、布局变化和属性更新产生的重复通知。
 * 隐藏容器不执行调整；再次显示、缓存页激活后重新测量。
 */
export function createContainerResizeWatcher(getContainer, onResize) {
  let observer = null
  let frameId = null
  let listening = false
  let lastSize = null

  function schedule() {
    if (!listening || frameId !== null) return
    frameId = requestAnimationFrame(() => {
      frameId = null
      if (!listening) return
      const container = getContainer()
      if (!container || container.clientWidth <= 0 || container.clientHeight <= 0) {
        lastSize = null
        return
      }
      const nextSize = [container.clientWidth, container.clientHeight, window.devicePixelRatio || 1]
      if (lastSize && nextSize.every((value, index) => value === lastSize[index])) return
      // 回调失败时保留未完成状态，后续尺寸通知仍能重试。
      if (onResize(container) !== false) lastSize = nextSize
    })
  }

  function start() {
    if (listening) return
    listening = true
    lastSize = null
    const container = getContainer()
    if (container && typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver(schedule)
      observer.observe(container)
    }
    // 保留窗口监听，兼容不支持 ResizeObserver 的浏览器和缩放变化。
    window.addEventListener('resize', schedule)
    schedule()
  }

  function stop() {
    listening = false
    window.removeEventListener('resize', schedule)
    if (observer) {
      observer.disconnect()
      observer = null
    }
    if (frameId !== null) {
      cancelAnimationFrame(frameId)
      frameId = null
    }
    lastSize = null
  }

  return { start, stop, schedule }
}
