import { createApp } from 'vue'
import Cookies from 'js-cookie'

import './styles/pageStyles'
import './styles/element-variables.styl'

import App from './App.vue'
import store from './store'
import router from './router'

import { registerIcons } from './icons'
import './permission'
import './assets/ttf/font.css'

import * as filters from './filters'

import elDrawer from '@/components/elDrawer/index.vue'

import echarts from '@/utils/echarts'

import JsEncrypt from 'jsencrypt'
import { Base64 } from 'js-base64'

import { registerDatav } from '@/plugins/datav'
import { registerAmap } from '@/plugins/amap'
import {
  ElMessage,
  ElMessageBox,
  ElNotification,
  ElLoading
} from 'element-plus'

import { registerGlobalFilters } from './plugins/filters'
import { registerElementGlobals } from './plugins/element-globals'
import { registerElementComponents } from './plugins/element-components'
import { registerElementTable } from './plugins/element-table'
import { resolveLegacyIcon } from './utils/legacy-element-icons'
import { setupIdleLogout } from './utils/idle-logout'

// 统计脚本默认关闭；确需启用时由部署环境注入地址，避免在源码中固化第三方标识。
const analyticsUrl = import.meta.env.VITE_ANALYTICS_URL
if (analyticsUrl) {
  const analytics = document.createElement('script')
  analytics.src = analyticsUrl
  analytics.async = true
  document.head.appendChild(analytics)
}

// 忽略 Element Plus 下拉/表格布局触发的 ResizeObserver 告警（dev 与发布环境一致）
const isResizeObserverLoopErr = (message) =>
  typeof message === 'string' && /ResizeObserver loop/.test(message)
window.addEventListener(
  'error',
  (event) => {
    if (isResizeObserverLoopErr(event.message)) {
      event.stopImmediatePropagation()
    }
  },
  true
)
window.addEventListener('unhandledrejection', (event) => {
  const reason = event.reason
  const message = reason?.message || String(reason || '')
  if (isResizeObserverLoopErr(message)) {
    event.preventDefault()
  }
})

const app = createApp(App)

app.use(store)
app.use(router)
app.component('ElDrawer', elDrawer) // 全局标签 elDrawer，与 ElDrawerWrap 组件对应
app.component('ElDrawerWrap', elDrawer) // 避免 unplugin 在发布构建中误解析为 element-plus 组件
registerDatav(app)
registerAmap(app)
registerIcons(app)

registerGlobalFilters(app, filters)
registerElementGlobals(app, {
  ElMessage,
  ElMessageBox,
  ElNotification,
  ElLoading
})
registerElementComponents(app)
registerElementTable(app)
app.config.globalProperties.$legacyIcon = resolveLegacyIcon
setupIdleLogout()

app.config.globalProperties.$echarts = echarts
app.config.globalProperties.JsEncrypt = JsEncrypt
app.config.globalProperties.$base64 = Base64

// 供 App.vue ElConfigProvider 读取默认尺寸
app.provide('defaultElementSize', Cookies.get('size') || 'default')

app.mount('#app')
