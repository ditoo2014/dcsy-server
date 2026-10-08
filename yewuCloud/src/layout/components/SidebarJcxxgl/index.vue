<template>
  <div>
    <div :class="{'has-logo':showLogo}" :style="{'height': height}">
      <logo v-if="showLogo" :collapse="isCollapse" />
      <el-scrollbar>
        <el-menu
          :default-active="activeMenu"
          :collapse="isCollapse"
          :background-color="variables.menuBg"
          :text-color="variables.menuText"
          :unique-opened="false"
          :active-text-color="variables.menuActiveText"
          :collapse-transition="false"
          mode="vertical"
          @select="handleMenuSelect"
        >
          <sidebar-item
            v-for="route in jcxxglMenuRoutes"
            :key="route.path"
            :item="route"
            :base-path="resolveMenuBasePath(route)"
            :active-menu="activeMenu"
          />
        </el-menu>
      </el-scrollbar>
    </div>
  </div>
</template>

<script>
import { mapGetters } from '@/store/pinia-helpers'
import Logo from './Logo'
import SidebarItem from './SidebarItem'
import variables from '@/styles/variables.js'
import { resolveSidebarActiveMenu } from '@/utils/sidebar-active-menu.js'

/** 合并路由前各子模块曾为独立 Layout 的 path，仅作菜单回退 */
const JCXXGL_LEGACY_ROOT_PATHS = ['/jcxxgl', '/zdxxgl', '/scxxgl', '/dcsjgl', '/rygl', '/jcsj']

export default {
  components: { SidebarItem, Logo },
  data() {
    return {
      height: '2000px'
    }
  },
  computed: {
    ...mapGetters([
      'permission_routes',
      'sidebar'
    ]),
    jcxxglMenuRoutes() {
      const layoutRoute = this.permission_routes.find(
        route => route.jcxxgl && route.path === '/jcxxgl' && Array.isArray(route.children)
      )
      if (layoutRoute && layoutRoute.children.length) {
        return layoutRoute.children.filter(child => child.jcxxgl && !child.hidden)
      }
      return this.permission_routes.filter(
        route => route.jcxxgl && JCXXGL_LEGACY_ROOT_PATHS.includes(route.path)
      )
    },
    activeMenu() {
      return resolveSidebarActiveMenu(this.$route)
    },
    showLogo() {
      return this.$store.state.settings.sidebarLogo
    },
    variables() {
      return variables
    },
    isCollapse() {
      return !this.sidebar.opened
    }
  },
  created() {
    this.height = document.documentElement.clientHeight - 60 + 'px'
  },
  methods: {
    resolveMenuBasePath(route) {
      if (!route || !route.path) {
        return ''
      }
      if (route.path.startsWith('/')) {
        return route.path
      }
      return '/jcxxgl'
    },
    handleMenuSelect(index) {
      const path = String(index || '')
      if (!path || path === this.$route.path) {
        return
      }
      this.$router.push(path)
    }
  }
}
</script>

<style lang="stylus">
// 业务侧栏规则不覆盖采集云表格和下拉框的原生横向滚动。
body:not(.collection-cloud-active) .el-scrollbar__wrap {
  overflow-x: hidden;
}
</style>
