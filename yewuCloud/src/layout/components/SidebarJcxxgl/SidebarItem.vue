<template>
  <!-- 嵌套子项须为 el-sub-menu 的直接子节点，不能包 menu-wrapper -->
  <el-menu-item
    v-if="isNest && !item.hidden && item.jcxxgl && item.meta"
    :index="resolvePath(item.path)"
    :title="item.meta.title"
  >
    <item :icon="nestItemIcon" />
    <template #title>
      <span class="menu-item-title">{{ item.meta.title }}</span>
    </template>
  </el-menu-item>

  <template v-else-if="!item.hidden && item.jcxxgl">
    <template v-if="!showSubmenu">
      <el-menu-item
        v-if="flatChild && flatChild.meta"
        :index="resolvePath(flatChild.path)"
        :title="flatChild.meta.title"
        :class="{ 'submenu-title-noDropdown': !isNest }"
      >
        <item :icon="iconMsg(flatChild, item)" />
        <template #title>
          <span class="menu-item-title">{{ flatChild.meta.title }}</span>
        </template>
      </el-menu-item>
    </template>

    <!-- Element Plus 2 使用 el-sub-menu，el-submenu 为 Element UI 旧标签，#title 不会生效 -->
    <el-sub-menu v-else ref="subMenu" :index="resolvePath(item.path)">
      <template #title>
        <item v-if="item.meta && item.meta.icon" :icon="item.meta.icon" />
        <span v-if="item.meta" class="menu-item-title">{{ item.meta.title }}</span>
      </template>
      <sidebar-item
        v-for="child in visibleChildren"
        :key="child.path"
        :is-nest="true"
        :item="child"
        :base-path="resolvePath(item.path)"
        class="nest-menu"
        :active-menu="activeMenu"
      />
    </el-sub-menu>
  </template>
</template>

<script>
import { resolveRoutePath } from '@/utils/resolve-route-path'
import Item from './Item'
import FixiOSBug from './FixiOSBug'

export default {
  name: 'SidebarItem',
  components: { Item },
  mixins: [FixiOSBug],
  props: {
    item: {
      type: Object,
      required: true
    },
    isNest: {
      type: Boolean,
      default: false
    },
    basePath: {
      type: String,
      default: ''
    },
    activeMenu: {
      type: String,
      default: ''
    }
  },
  computed: {
    visibleChildren() {
      return (this.item.children || []).filter(child => !child.hidden && child.jcxxgl)
    },
    showSubmenu() {
      const count = this.visibleChildren.length
      if (count > 1) {
        return true
      }
      if (count === 1 && this.item.alwaysShow) {
        return true
      }
      return false
    },
    flatChild() {
      if (this.visibleChildren.length === 1) {
        return this.visibleChildren[0]
      }
      if (this.visibleChildren.length === 0) {
        if (this.item.component && this.item.meta) {
          return this.item
        }
        return { ...this.item, path: '', noShowingChildren: true }
      }
      return null
    },
    nestItemIcon() {
      let activeMenu = ''
      if (this.activeMenu.lastIndexOf('/') === -1) {
        activeMenu = this.activeMenu
      } else {
        activeMenu = this.activeMenu.slice(this.activeMenu.lastIndexOf('/') + 1)
      }
      return this.item.name === activeMenu && this.item.meta.iconActive
        ? this.item.meta.iconActive
        : this.item.meta.icon
    },
    iconMsg() {
      return (onlyOneChild, item) => {
        let activeMenu = ''
        if (this.activeMenu.lastIndexOf('/') === -1) {
          activeMenu = this.activeMenu
        } else {
          activeMenu = this.activeMenu.slice(this.activeMenu.lastIndexOf('/') + 1)
        }
        return onlyOneChild.name === activeMenu && onlyOneChild.meta.iconActive
          ? onlyOneChild.meta.iconActive
          : onlyOneChild.meta.icon || (item.meta && item.meta.icon)
      }
    }
  },
  methods: {
    resolvePath(routePath) {
      return resolveRoutePath(this.basePath, routePath)
    }
  }
}
</script>
