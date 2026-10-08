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
            v-for="route in dcjkglMenuRoutes"
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

const DCJKGL_ROOT_PATHS = ['/jkztjc', '/lssjcx', '/gjxx', '/yjxx']

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
    dcjkglMenuRoutes() {
      return this.permission_routes.filter(
        route => route.dcjkgl && DCJKGL_ROOT_PATHS.includes(route.path)
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
