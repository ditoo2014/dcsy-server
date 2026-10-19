<script setup>
// 与原采集云一致：注册中文日期规则，保证日历从周一开始显示。
import 'dayjs/locale/zh-cn'
import collectionCss from '../styles/style.css?inline'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Connection, Expand, Fold, Grid, Monitor, Moon, Sunny, UserFilled } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'
import { useTagsViewStore } from '../stores/tagsView.js'
import { redirectToMainLogin } from '../utils/mainLoginRedirect'
import { mountCollectionPageStyles } from '../../styles/pageStyles'

// 原采集云全局样式只在布局存活时生效，包含挂载到 body 的弹层。
const unmountCollectionStyles = mountCollectionPageStyles(collectionCss)
const previousTheme = document.documentElement.getAttribute('data-theme')
onBeforeUnmount(() => {
  unmountCollectionStyles()
  // 同一标签页返回业务云时，恢复进入采集云之前的主题状态。
  if (previousTheme === null) document.documentElement.removeAttribute('data-theme')
  else document.documentElement.setAttribute('data-theme', previousTheme)
})

const COLLAPSE_KEY = 'layout_sidebar_collapsed'
const THEME_KEY = 'layout_theme'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const tagsStore = useTagsViewStore()
// 临时仅向 ditoo 显示模板入口，等待登录信息加载完成以避免旧账号缓存造成误显示。
const showTtdcTemplateManagement = computed(
  () => authStore.accessLoaded && authStore.account === 'ditoo'
)

const sidebarCollapsed = ref(localStorage.getItem(COLLAPSE_KEY) === '1')
const isDarkTheme = ref(localStorage.getItem(THEME_KEY) === 'dark')

watch(sidebarCollapsed, v => {
  localStorage.setItem(COLLAPSE_KEY, v ? '1' : '0')
})

function applyTheme() {
  const theme = isDarkTheme.value ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', theme)
  localStorage.setItem(THEME_KEY, theme)
}

watch(isDarkTheme, () => {
  applyTheme()
}, { immediate: true })

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

function toggleTheme() {
  isDarkTheme.value = !isDarkTheme.value
}

const themeToggleButtonStyle = computed(() => {
  if (!isDarkTheme.value) return {}
  return {
    background: 'linear-gradient(180deg, #16314d 0%, #12263d 100%)',
    borderColor: '#4f7fab',
    color: '#eaf4ff',
    boxShadow: 'inset 0 0 0 1px rgba(143, 212, 255, 0.2), 0 2px 8px rgba(2, 6, 23, 0.28)'
  }
})

const showSideMenu = computed(
  () => route.name !== 'gatewayDetail' && route.name !== 'gatewayDetailLegacy'
)

const menuActive = computed(() => {
  if (route.name === 'ttdcTemplates') return 'ttdcTemplates'
  if (route.name !== 'collectionHome') return ''
  return route.query.module === 'stream' ? 'stream' : 'gateway'
})

function onMenuSelect(index) {
  if (index === 'gateway') {
    void router.push({ path: '/collection', query: { module: 'gateway' }})
  } else if (index === 'stream') {
    void router.push({ path: '/collection', query: { module: 'stream' }})
  } else if (index === 'ttdcTemplates' && showTtdcTemplateManagement.value) {
    void router.push({ name: 'ttdcTemplates' })
  }
}

const breadcrumbTitle = computed(() => tagsStore.resolveTitle(route))

function logout() {
  redirectToMainLogin({ notifyBackend: true, broadcast: true })
}

function handleTagClick(item) {
  if (!tagsStore.isTabActive(route, item)) {
    void router.push(item.fullPath)
  }
}

function handleTagClose(item, e) {
  e.stopPropagation()
  if (tagsStore.visitedViews.length <= 1) return
  const isActive = tagsStore.isTabActive(route, item)
  tagsStore.delView(item.tabKey)
  if (isActive) {
    const next = tagsStore.visitedViews[tagsStore.visitedViews.length - 1]
    void router.push(next?.fullPath || '/collection')
  }
}

// 独立布局自行维护采集云标签，不与业务云标签栏混用。
watch(() => route.fullPath, () => {
  tagsStore.addVisitedView(route)
}, { immediate: true })
</script>

<template>
  <div class="admin-layout">
    <aside v-if="showSideMenu" class="sider" :class="{ 'is-collapsed': sidebarCollapsed }">
      <div class="logo">
        <span class="logo-text">{{ sidebarCollapsed ? 'T' : '锂硫电池溯源平台' }}</span>
      </div>
      <div v-show="!sidebarCollapsed" class="side-title">功能菜单</div>
      <el-menu
        :key="menuActive || 'aside'"
        class="side-menu"
        :collapse="sidebarCollapsed"
        :default-active="menuActive || undefined"
        :collapse-transition="false"
        background-color="#001529"
        text-color="rgba(255, 255, 255, 0.72)"
        active-text-color="#1890ff"
        @select="onMenuSelect"
      >
        <el-menu-item index="gateway">
          <el-icon><Connection /></el-icon>
          <span>网关管理</span>
        </el-menu-item>
        <el-menu-item index="stream">
          <el-icon><Monitor /></el-icon>
          <span>数据池管理</span>
        </el-menu-item>
        <el-menu-item v-if="showTtdcTemplateManagement" index="ttdcTemplates">
          <el-icon><Grid /></el-icon>
          <span>TTDC公共模板</span>
        </el-menu-item>
      </el-menu>
    </aside>

    <div class="main-column">
      <header class="content-header">
        <div class="breadcrumb-row">
          <button
            v-if="showSideMenu"
            type="button"
            class="sider-trigger"
            aria-label="折叠菜单"
            @click="toggleSidebar"
          >
            <el-icon :size="18">
              <Fold v-if="!sidebarCollapsed" />
              <Expand v-else />
            </el-icon>
          </button>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/collection' }">主页</el-breadcrumb-item>
            <el-breadcrumb-item>{{ breadcrumbTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
          <div class="header-actions">
            <el-button class="theme-toggle" :style="themeToggleButtonStyle" size="small" @click="toggleTheme">
              <el-icon><Sunny v-if="isDarkTheme" /><Moon v-else /></el-icon>
              <span>{{ isDarkTheme ? '浅色' : '深色' }}</span>
            </el-button>
            <span class="user-name">
              <el-icon :size="14" aria-hidden="true"><UserFilled /></el-icon>
              <span>{{ authStore.username || '未登录' }}</span>
            </span>
            <el-button type="primary" size="small" @click="logout">退出</el-button>
          </div>
        </div>

        <div
          v-if="tagsStore.visitedViews.length"
          class="tags-view"
          role="tablist"
          aria-label="已打开页面"
        >
          <div
            v-for="item in tagsStore.visitedViews"
            :key="item.tabKey"
            role="tab"
            class="tags-item"
            :class="{ 'is-active': tagsStore.isTabActive(route, item) }"
            @click="handleTagClick(item)"
          >
            <span class="tags-dot" aria-hidden="true" />
            <span class="tags-title">{{ item.title }}</span>
            <button
              type="button"
              class="tags-close"
              aria-label="关闭标签"
              :disabled="tagsStore.visitedViews.length <= 1"
              @click="handleTagClose(item, $event)"
            >
              ×
            </button>
          </div>
        </div>
      </header>

      <div class="content-body">
        <RouterView />
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-layout {
  display: flex;
  width: 100%;
  height: 100vh;
  min-height: 100vh;
  background: #f0f2f5;
  overflow: hidden;
}

.sider {
  flex-shrink: 0;
  width: 220px;
  background: #001529;
  display: flex;
  flex-direction: column;
  transition: width 0.2s ease;
  overflow: hidden;
}

.sider.is-collapsed {
  width: 64px;
}

.logo {
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.logo-text {
  font-size: 15px;
  font-weight: 600;
  color: #fff;
  letter-spacing: 0.5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sider.is-collapsed .logo-text {
  font-size: 18px;
}

.side-title {
  padding: 12px 18px 6px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.45);
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

.side-menu {
  flex: 1;
  border-right: none !important;
}

.side-menu :deep(.el-menu) {
  border-right: none;
}

.side-menu :deep(.el-menu-item) {
  margin: 4px 8px;
  border-radius: 4px;
  gap: 8px;
}

.side-menu :deep(.el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.08) !important;
}

.side-menu :deep(.el-menu-item.is-active) {
  background: rgba(24, 144, 255, 0.18) !important;
}

.main-column {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.content-header {
  flex-shrink: 0;
  background: #fff;
  border-bottom: 1px solid #f0f0f0;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.06);
}

.breadcrumb-row {
  display: flex;
  /* 浏览器缩放后允许顶栏自然换行，避免账号信息挤出可见区域。 */
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  min-height: 48px;
  padding: 0 16px;
  border-bottom: 1px solid #f0f0f0;
}

.sider-trigger {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: rgba(0, 0, 0, 0.65);
  cursor: pointer;
}

.sider-trigger:hover {
  background: #f5f5f5;
  color: #1890ff;
}

.breadcrumb-row :deep(.el-breadcrumb) {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: 14px;
}

.header-actions {
  display: flex;
  flex-wrap: wrap;
  min-width: 0;
  align-items: center;
  gap: 12px;
  margin-left: auto;
}

.theme-toggle {
  display: none;
  align-items: center;
  gap: 6px;
}

/* 深色主题下，主题切换按钮改为深色样式 */
:global(html[data-theme='dark']) .theme-toggle,
:global([data-theme='dark']) .theme-toggle,
:global(html.dark) .theme-toggle,
:global(body.dark) .theme-toggle,
:global(.theme-dark) .theme-toggle,
:global(.dark-theme) .theme-toggle {
  background: linear-gradient(180deg, #16314d 0%, #12263d 100%) !important;
  border-color: #4f7fab !important;
  color: #eaf4ff !important;
  box-shadow:
    inset 0 0 0 1px rgba(143, 212, 255, 0.2),
    0 2px 8px rgba(2, 6, 23, 0.28) !important;
}

:global(html[data-theme='dark']) .theme-toggle:hover,
:global([data-theme='dark']) .theme-toggle:hover,
:global(html.dark) .theme-toggle:hover,
:global(body.dark) .theme-toggle:hover,
:global(.theme-dark) .theme-toggle:hover,
:global(.dark-theme) .theme-toggle:hover {
  background: linear-gradient(180deg, #1b3a5c 0%, #15304d 100%) !important;
  border-color: #6a9cc8 !important;
  color: #ffffff !important;
}

.user-name {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 14px;
  background: #fafafa;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.65);
  max-width: min(240px, 40vw);
  min-width: 0;
}

.user-name > span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tags-view {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  padding: 8px 12px 10px;
  background: #fafafa;
  min-height: 40px;
}

.tags-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 200px;
  padding: 4px 10px 4px 8px;
  font-size: 13px;
  line-height: 1.4;
  color: rgba(0, 0, 0, 0.65);
  background: #fff;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  cursor: pointer;
  user-select: none;
  transition: border-color 0.15s ease, color 0.15s ease, background 0.15s ease;
}

.tags-item:hover {
  color: #1890ff;
  border-color: #1890ff;
}

.tags-item.is-active {
  color: #fff;
  background: #1890ff;
  border-color: #1890ff;
}

.tags-dot {
  display: none;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.tags-item.is-active .tags-dot {
  display: inline-block;
  background: #fff;
}

.tags-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tags-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  margin-left: 2px;
  padding: 0;
  border: none;
  border-radius: 2px;
  background: transparent;
  color: inherit;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
  opacity: 0.65;
}

.tags-close:hover:not(:disabled) {
  opacity: 1;
  background: rgba(0, 0, 0, 0.06);
}

.tags-item.is-active .tags-close:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.2);
}

.tags-close:disabled {
  cursor: not-allowed;
  opacity: 0.35;
}

.content-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
  padding: 0;
}

@media (max-width: 768px) {
  .admin-layout {
    flex-direction: column;
  }

  .sider {
    width: 100% !important;
    flex-direction: row;
    flex-wrap: wrap;
    align-items: center;
  }

  .sider.is-collapsed {
    width: 100% !important;
  }

  .side-menu {
    width: 100%;
  }

  .side-menu :deep(.el-menu) {
    display: flex;
    flex-wrap: wrap;
  }
}
</style>
