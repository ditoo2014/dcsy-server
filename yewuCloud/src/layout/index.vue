<template>
  <div :class="classObj" class="app-wrapper">
    <LayoutBanner
      :banner-id="bannerId"
      @banner-click="bannerClick"
    />
    <div v-if="device==='mobile'&&sidebar.opened" class="drawer-bg" @click="handleClickOutside" />

    <!-- 单一 AppMain/router-view，避免多个 v-if 分支在生产环境反复销毁路由出口引发栈溢出 -->
    <div :class="shellClass">
      <component
        :is="activeSidebar"
        v-if="activeSidebar"
        :class="sidebarClass"
      />
      <div :class="mainWrapClass">
        <div v-if="showInnerNavbar" :class="{'fixed-header':fixedHeader}">
          <navbar />
          <div v-if="needTagsView" style="padding:10px">
            <TagsView />
          </div>
        </div>
        <AppMain />
      </div>
    </div>
  </div>
</template>

<script>
import { AppMain, Navbar, SidebarQsmzqgl, SidebarDcjkgl,
  SidebarJcxxgl, Banner as LayoutBanner } from './components'
import ResizeMixin from './mixin/ResizeHandler'
import { exitFullscreenSafe } from '@/utils/fullscreen.js'
import { getDefaultBannerId, resolveBannerIdFromPath } from '@/utils/banner-route.js'
import { mapState } from '@/store/pinia-helpers'

const INNER_NAVBAR_IDS = new Set(['12', '13', '14'])
const SIDEBAR_CONTAINER_IDS = new Set(['12', '13', '14'])

export default {
  name: 'Layout',
  components: {
    AppMain,
    Navbar,
    SidebarQsmzqgl,
    SidebarDcjkgl,
    SidebarJcxxgl,
    LayoutBanner,
  },
  mixins: [ResizeMixin],
  data() {
    return {
      overrideBannerId: null
    }
  },
  computed: {
    ...mapState({
      sidebar: state => state.app.sidebar,
      device: state => state.app.device,
      needTagsView: state => state.settings.tagsView,
      fixedHeader: state => state.settings.fixedHeader
    }),
    bannerId() {
      if (this.overrideBannerId) {
        return this.overrideBannerId
      }
      return resolveBannerIdFromPath(this.$route.path) || getDefaultBannerId()
    },
    activeSidebar() {
      const map = {
        '12': 'SidebarQsmzqgl',
        '13': 'SidebarDcjkgl',
        '14': 'SidebarJcxxgl'
      }
      return map[this.bannerId] || null
    },
    showInnerNavbar() {
      return INNER_NAVBAR_IDS.has(this.bannerId)
    },
    sidebarClass() {
      if (SIDEBAR_CONTAINER_IDS.has(this.bannerId)) {
        return 'sidebar-container'
      }
      return ''
    },
    shellClass() {
      return {
        'layout-shell': true,
        'layout-shell--jsc': this.bannerId === '11'
      }
    },
    mainWrapClass() {
      return {
        hasTagsView: this.needTagsView,
        'main-container': true
      }
    },
    classObj() {
      return {
        hideSidebar: !this.sidebar.opened,
        openSidebar: this.sidebar.opened,
        withoutAnimation: this.sidebar.withoutAnimation,
        mobile: this.device === 'mobile',
        'app-wrapper--jsc': this.isJscRoute
      }
    },
    isJscRoute() {
      return this.$route.path === '/jsc'
    }
  },
  watch: {
    '$route.path': {
      immediate: true,
      handler(path, oldPath) {
        if (oldPath === '/jsc' && path !== '/jsc') {
          exitFullscreenSafe()
        }
        this.overrideBannerId = null
        const routeBannerId = resolveBannerIdFromPath(path)
        if (routeBannerId) {
          localStorage.setItem('bannerId', routeBannerId)
        }
      }
    }
  },
  methods: {
    handleClickOutside() {
      this.$store.dispatch('app/closeSideBar', { withoutAnimation: false })
    },
    bannerClick(e) {
      const id = String(e)
      localStorage.setItem('bannerId', id)
      this.overrideBannerId = id
    }
  }
}
</script>

<style lang="stylus" scoped>
  @import "../styles/mixin.styl";
  @import "../styles/variables.styl";

  .app-wrapper {
    // 页头、侧栏和固定导航共用高度，窄屏时不再各自使用不同的固定值。
    --business-banner-height: 3.5417vw;
    clearfix();
    display: flex;
    flex-direction: column;
    position: relative;
    height: 100vh;
    width: 100%;
    overflow: hidden;
    background-color: #112238;
    &.mobile.openSidebar {
      position: fixed;
      top: 0;
    }
    &.app-wrapper--jsc {
      background-color: #020B41;
    }
  }

  .layout-shell {
    flex: 1;
    min-height: 0;
    width: 100%;
    display: flex;
    flex-direction: column;
    &.layout-shell--jsc {
      flex: 1;
      min-height: 0;
      width: 100%;
      display: flex;
      flex-direction: column;
    }
  }

  .app-wrapper--jsc :deep(.main-container) {
    margin-left: 0;
    width: 100%;
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .app-wrapper :deep(.main-container) {
    flex: 1;
    min-height: 0;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }

  .app-wrapper :deep(.sidebar-container) {
    // 覆盖旧 #app 侧栏固定定位规则，确保页头高度只扣除一次。
    margin-top: 0 !important;
    top: var(--business-banner-height) !important;
    height: calc(100% - var(--business-banner-height)) !important;
  }

  .app-wrapper--jsc :deep(.app-main) {
    flex: 1;
    min-height: 0;
    height: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .app-wrapper--jsc :deep(.app-main > *) {
    flex: 1;
    min-height: 0;
    height: 100%;
  }

  .drawer-bg {
    background: #000;
    opacity: 0.3;
    width: 100%;
    top: 0;
    height: 100%;
    position: absolute;
    z-index: 999;
  }

  .fixed-header {
    position: fixed;
    top: var(--business-banner-height);
    right: 0;
    z-index: 9;
    width: calc(100% - 185px);
    transition: width 0.28s;
  }

  @media (max-width: 1280px) {
    .app-wrapper {
      --business-banner-height: 52px;
    }
  }

  .hideSidebar .fixed-header {
    width: calc(100% - 64px)
  }

  .mobile .fixed-header {
    width: 100%;
  }

  .mySidebarCss {
    margin-top: 3.5417vw;
    transition: width 0.28s;
    width: 150px !important;
    background-color: $menuBg;
    height: 100%;
    position: fixed;
    font-size: 0px;
    top: 0;
    bottom: 0;
    left: 0;
    z-index: 1001;
    overflow: hidden;
  }

  .myMainContainerCss {
    min-height: 100%;
    transition: margin-left .28s;
    margin-left: 150px;
    position: relative;
    background-color: #DAE3EC;
  }

</style>
