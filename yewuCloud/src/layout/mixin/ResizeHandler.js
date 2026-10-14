import store from '@/store'

const WIDTH = 992 // 以浏览器可用的 CSS 像素判断侧栏模式，兼容分辨率和页面缩放。

export default {
  watch: {
    $route(route) {
      if (this.device === 'mobile' && this.sidebar.opened) {
        store.dispatch('app/closeSideBar', { withoutAnimation: false })
      }
    }
  },
  beforeMount() {
    window.addEventListener('resize', this.$_resizeHandler)
    document.addEventListener('visibilitychange', this.$_resizeHandler)
  },
  beforeUnmount() {
    window.removeEventListener('resize', this.$_resizeHandler)
    document.removeEventListener('visibilitychange', this.$_resizeHandler)
  },
  mounted() {
    this.$_resizeHandler()
  },
  methods: {
    // use $_ for mixins properties
    // https://vuejs.org/v2/style-guide/index.html#Private-property-names-essential
    $_isMobile() {
      // 页面内容可能具有最小宽度，不能用 body 的内容宽度代替真实视口。
      return document.documentElement.clientWidth < WIDTH
    },
    $_resizeHandler() {
      if (!document.hidden) {
        const isMobile = this.$_isMobile()
        const device = isMobile ? 'mobile' : 'desktop'
        const deviceChanged = this.device !== device
        store.dispatch('app/toggleDevice', device)

        // 仅在进入窄屏模式时收起侧栏，避免移动窗口时关掉用户刚打开的菜单。
        if (isMobile && deviceChanged) {
          store.dispatch('app/closeSideBar', { withoutAnimation: true })
        }
      }
    }
  }
}
