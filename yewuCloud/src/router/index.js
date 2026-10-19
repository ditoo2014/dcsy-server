import { createRouter, createWebHashHistory } from 'vue-router'
import Layout from '@/layout'
import { resetDynamicRoutesReady } from '@/utils/dynamicRouteState.js'
import { TTDC_TEMPLATE_PERMISSIONS } from '@/collection/constants/permissions.js'
import { LEGACY_FRONTEND_PATHS } from '@/utils/legacyRoutePaths.js'

// 保留原业务云的菜单图标；路由精简不应改变原来的侧栏样式。
const dcsyIcons = {
  Dashboard: 'dashboardOverview', BatteryStatus: 'largeScreen', BatteryTracking: 'guide', MaintenanceManagement: 'component',
  HealthMonitoring: 'equipmentManagement', HistoricalDataQuery: 'diagnosReport', RealtimeAlarms: 'star',
  AlarmHistory: 'nodeOnlineRecord', RealtimeWarnings: 'feed', WarningHistory: 'faultDatabase',
  WarehouseManagement: 'keySites', SiteManagement: 'nodemanage', BatteryFactoryInformation: 'equipOver',
  DataAccess: 'link', GeneralTriggerManagement: 'procSet', RoleManagement: 'dashboard', UserManagement: 'user', SystemLogs: 'file'
}

/** DCSY固定路由，只注册锂硫电池溯源系统实际使用的页面。 */
export const constantRoutes = [
  { path: '/login', component: () => import('@/views/login/login.vue'), hidden: true },
  { path: '/modifyPassword', component: () => import('@/views/accounts/ModifyPassword.vue'), hidden: true },
  { path: '/batteryDetail', component: () => import('@/views/batteryTraceability/batteryHealth/BatteryDetail.vue'), name: 'BatteryDetail', hidden: true },
  // 数据接入的“管理”在新窗口打开，固定注册以支持刷新和重新登录。
  { path: '/equipmentManagement', component: () => import('@/views/equipment/EquipmentManagementPage.vue'), name: 'EquipmentManagementPage', hidden: true, meta: { title: '电池管理' } },
  {
    path: '/collection',
    component: () => import('@/collection/layouts/MainLayout.vue'),
    hidden: true,
    meta: { title: '采集云平台', collectionCloud: true, routeContainer: true },
    children: [
      {
        path: '',
        name: 'collectionHome',
        component: () => import('@/collection/views/HomeView.vue'),
        meta: { title: '工作台', collectionCloud: true }
      },
      {
        path: 'gateway/:modelSlug/:gatewayId',
        name: 'gatewayDetail',
        component: () => import('@/collection/views/GatewayDetailEntryView.vue'),
        meta: { title: '网关详情', collectionCloud: true }
      },
      {
        path: 'gateway/:gatewayId',
        name: 'gatewayDetailLegacy',
        component: () => import('@/collection/views/GatewayLegacyRedirectView.vue'),
        meta: { title: '网关详情', collectionCloud: true }
      },
      {
        path: 'stream/:streamCode',
        name: 'streamDetail',
        component: () => import('@/collection/views/StreamDetailView.vue'),
        meta: { title: '数据池详情', collectionCloud: true }
      },
      // 恢复模板页面路由，仅在采集云菜单中临时向 ditoo 显示入口。
      {
        path: 'ttdcTemplates',
        name: 'ttdcTemplates',
        component: () => import('@/collection/views/TtdcTemplateView.vue'),
        meta: {
          title: 'TTDC公共模板',
          collectionCloud: true,
          collectionPermission: TTDC_TEMPLATE_PERMISSIONS.view
        }
      }
    ]
  },
  // 旧地址仅负责跳转；查询参数和锚点继续传给对应英文页面，权限仍由原守卫检查。
  ...Object.entries(LEGACY_FRONTEND_PATHS).map(([path, target], index) => ({
    path,
    name: `legacyPage${index}`,
    hidden: true,
    redirect: to => ({ path: target, query: to.query, hash: to.hash })
  })),
  { path: '/', redirect: '/dashboard', hidden: true }
]

/** DCSY权限路由：分组标记使用英文，页面标题和权限筛选规则保持不变。 */
export const asyncRoutes = [
  dcsyPage('/dashboard', 'Dashboard', '驾驶舱', () => import('@/views/batteryTraceability/dashboard/Dashboard.vue'), 'dashboardSection'),
  dcsyPage('/batteryStatus', 'BatteryStatus', '电池状态', () => import('@/views/batteryTraceability/lifecycle/BatteryStatus.vue'), 'lifecycleSection'),
  dcsyPage('/batteryTracking', 'BatteryTracking', '轨迹追踪', () => import('@/views/batteryTraceability/lifecycle/BatteryTracking.vue'), 'lifecycleSection'),
  dcsyPage('/maintenance', 'MaintenanceManagement', '运维管理', () => import('@/views/batteryTraceability/lifecycle/MaintenanceManagement.vue'), 'lifecycleSection'),
  dcsyPage('/healthMonitoring', 'HealthMonitoring', '健康状态监测', () => import('@/views/batteryTraceability/batteryHealth/HealthMonitoring.vue'), 'batteryHealthSection'),
  dcsyPage('/historicalData', 'HistoricalDataQuery', '历史数据查询', () => import('@/views/batteryTraceability/batteryHealth/HistoricalDataQuery.vue'), 'batteryHealthSection'),
  {
    path: '/alarms', name: 'layoutAlarms', component: Layout, redirect: '/alarms/realtime', batteryHealthSection: true,
    meta: { title: '告警信息', icon: 'alarmLog', routeContainer: true },
    children: [
      dcsyChild('/alarms/realtime', 'RealtimeAlarms', '实时告警', () => import('@/views/batteryTraceability/batteryHealth/RealtimeAlarms.vue'), 'batteryHealthSection'),
      dcsyChild('/alarms/history', 'AlarmHistory', '历史告警', () => import('@/views/batteryTraceability/batteryHealth/AlarmHistory.vue'), 'batteryHealthSection')
    ]
  },
  {
    path: '/warnings', name: 'layoutWarnings', component: Layout, redirect: '/warnings/realtime', batteryHealthSection: true,
    meta: { title: '预警信息', icon: 'warning', routeContainer: true },
    children: [
      dcsyChild('/warnings/realtime', 'RealtimeWarnings', '实时预警', () => import('@/views/batteryTraceability/batteryHealth/RealtimeWarnings.vue'), 'batteryHealthSection'),
      dcsyChild('/warnings/history', 'WarningHistory', '历史预警', () => import('@/views/batteryTraceability/batteryHealth/WarningHistory.vue'), 'batteryHealthSection')
    ]
  },
  {
    path: '/basicInformation', name: 'layoutBasicInformation', component: Layout, redirect: '/basicInformation/warehouses', basicInformationSection: true,
    meta: { title: '基础信息管理', icon: 'basicmanage', routeContainer: true },
    children: [
      dcsyChild('warehouses', 'WarehouseManagement', '库房信息管理', () => import('@/views/batteryTraceability/basicInformation/WarehouseManagement.vue'), 'basicInformationSection'),
      dcsyChild('/sites', 'SiteManagement', '站点信息管理', () => import('@/views/batteryTraceability/basicInformation/SiteManagement.vue'), 'basicInformationSection'),
      dcsyChild('/batteryFactory', 'BatteryFactoryInformation', '电池出厂信息', () => import('@/views/batteryTraceability/basicInformation/BatteryFactoryInformation.vue'), 'basicInformationSection'),
      {
        path: '/batteryData', redirect: '/batteryData/dataAccess', alwaysShow: true, basicInformationSection: true,
        meta: { title: '电池数据管理', icon: 'collectCloud' },
        children: [
          {
            path: 'gatewayConfig',
            name: 'collectionLauncher',
            component: () => import('@/views/batteryTraceability/basicInformation/GatewayConfigLauncher.vue'),
            basicInformationSection: true,
            meta: { title: '网关配置', icon: 'procSet', noCache: true }
          },
          dcsyChild('dataAccess', 'DataAccess', '数据接入', () => import('@/views/batteryTraceability/basicInformation/DataAccess.vue'), 'basicInformationSection'),
          dcsyChild('generalTriggers', 'GeneralTriggerManagement', '通用触发器管理', () => import('@/views/batteryTraceability/basicInformation/GeneralTriggerManagement.vue'), 'basicInformationSection')
        ]
      },
      {
        path: '/accounts', redirect: '/accounts/users', alwaysShow: true, basicInformationSection: true,
        meta: { title: '人员管理', icon: 'peoples' },
        children: [
          dcsyChild('roles', 'RoleManagement', '角色管理', () => import('@/views/batteryTraceability/basicInformation/RoleManagement.vue'), 'basicInformationSection'),
          dcsyChild('users', 'UserManagement', '用户管理', () => import('@/views/batteryTraceability/basicInformation/UserManagement.vue'), 'basicInformationSection')
        ]
      },
      {
        path: '/basicData', redirect: '/basicData/systemLogs', alwaysShow: true, basicInformationSection: true,
        meta: { title: '基础数据', icon: 'chart' },
        children: [dcsyChild('systemLogs', 'SystemLogs', '系统日志', () => import('@/views/batteryTraceability/basicInformation/SystemLogs.vue'), 'basicInformationSection')]
      }
    ]
  }
]

function dcsyPage(path, name, title, component, section) {
  // 父路由使用独立名称，退出时可整条移除；子路由名称与页面组件名称保持一致。
  return { path, name: `layout${name}`, component: Layout, meta: { routeContainer: true }, [section]: true, children: [dcsyChild('', name, title, component, section)] }
}

function dcsyChild(path, name, title, component, section) {
  return { path, name, component, [section]: true, meta: { title, icon: dcsyIcons[name] }}
}

export function createAppRouter() {
  return createRouter({ history: createWebHashHistory(), scrollBehavior: () => ({ top: 0 }), routes: constantRoutes })
}

const router = createAppRouter()

function collectRouteNames(routes, names = new Set()) {
  routes.forEach(route => {
    if (route.name) names.add(route.name)
    if (Array.isArray(route.children)) collectRouteNames(route.children, names)
  })
  return names
}

const constantRouteNames = collectRouteNames(constantRoutes)

export function resetRouter() {
  resetDynamicRoutesReady()
  // 删除具名父路由会同时删除其子页面及分组，避免重新登录匹配旧布局。
  router.getRoutes().forEach(route => {
    if (route.name && !constantRouteNames.has(route.name)) router.removeRoute(route.name)
  })
}

export default router
