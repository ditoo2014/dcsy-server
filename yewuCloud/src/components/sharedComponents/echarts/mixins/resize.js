import { createContainerResizeWatcher } from '@/utils/containerResize.js'

export default {
  mounted() {
    this.__chartSizeWatcher = createContainerResizeWatcher(
      () => this.chart ? this.chart.getDom() : this.$el,
      () => this.$_resizeChartContainer()
    )
    this.__chartResizeActive = true
    // 混入 mounted 先于组件自身执行，等待图表实例和最终布局就绪。
    this.$nextTick(() => {
      if (this.__chartResizeActive) this.__chartSizeWatcher.start()
    })
    this.$_bindSidebarResize()
  },
  activated() {
    this.__chartResizeActive = true
    this.$nextTick(() => {
      if (this.__chartResizeActive && this.__chartSizeWatcher) this.__chartSizeWatcher.start()
    })
    this.$_bindSidebarResize()
  },
  deactivated() {
    this.__chartResizeActive = false
    this.__chartSizeWatcher && this.__chartSizeWatcher.stop()
    this.$_unbindSidebarResize()
  },
  beforeUnmount() {
    this.__chartResizeActive = false
    this.__chartSizeWatcher && this.__chartSizeWatcher.stop()
    this.$_unbindSidebarResize()
  },
  methods: {
    $_bindSidebarResize() {
      if (this.$_sidebarElm) return
      this.$_sidebarElm = document.getElementsByClassName('sidebar-container')[0]
      this.$_sidebarElm && this.$_sidebarElm.addEventListener('transitionend', this.$_sidebarResizeHandler)
    },
    $_unbindSidebarResize() {
      this.$_sidebarElm && this.$_sidebarElm.removeEventListener('transitionend', this.$_sidebarResizeHandler)
      this.$_sidebarElm = null
    },
    $_scheduleChartResize() {
      this.__chartSizeWatcher && this.__chartSizeWatcher.schedule()
    },
    $_resizeChartContainer() {
      if (!this.chart || this.chart.isDisposed()) return false
      try {
        this.chart.resize()
      } catch (error) {
        console.warn('[图表尺寸调整失败]', error)
        // 保留原有异常恢复能力，同时让问题在控制台可见。
        if (typeof this.reSetOption === 'function') this.reSetOption()
        return false
      }
      if (typeof this.afterChartResize === 'function') this.afterChartResize()
      return true
    },
    $_sidebarResizeHandler(e) {
      if (e.propertyName === 'width') {
        this.$_scheduleChartResize()
      }
    }
  }
}
