<template>
  <div>
    <div class="sidebar-body" :class="{'has-logo':showLogo}">
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
            v-for="route in batteryHealthMenuRoutes"
            :key="route.path"
            :item="route"
            :base-path="route.path"
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

const BATTERY_HEALTH_ROOT_PATHS = ['/jkztjc', '/lssjcx', '/gjxx', '/yjxx']

export default {
  components: { SidebarItem, Logo },
  computed: {
    ...mapGetters([
      'permission_routes',
      'sidebar'
    ]),
    batteryHealthMenuRoutes() {
      return this.permission_routes.filter(
        route => route.dcjkgl && BATTERY_HEALTH_ROOT_PATHS.includes(route.path)
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
  methods: {
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

<style lang="stylus" scoped>
// 使用侧栏实际可用高度，窗口或缩放变化后由浏览器自动重新计算。
.sidebar-body {
  height: 100%;
  :deep(.el-scrollbar) {
    height: 100%;
  }
  &.has-logo :deep(.el-scrollbar) {
    height: calc(100% - 50px);
  }
  // 仅禁止侧栏横向滚动，避免影响业务页面的表格和下拉框。
  :deep(.el-scrollbar__wrap) {
    overflow-x: hidden;
  }
}
</style>
