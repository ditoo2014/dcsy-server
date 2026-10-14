import { createRouter, createWebHashHistory } from 'vue-router'
import Layout from '@/layout'
import { resetDynamicRoutesReady } from '@/utils/dynamic-route-state.js'
// import { TTDC_TEMPLATE_PERMISSIONS } from '@/collection/constants/permissions.js'

// 保留原业务云的菜单图标；路由精简不应改变原来的侧栏样式。
const dcsyIcons = {
  jsc: 'kanBanMana', dczt: 'bigPin', gjzz: 'guide', ywgl: 'component',
  jkztjc: 'equipMana', lssjcx: 'diagnosReport', gjxxssgj: 'star',
  gjxxlsgj: 'nodeOnlineRecord', yjxxssyj: 'feed', yjxxlsyj: '故障信息数据库',
  kfxxgl: 'keySites', zdxxgl: 'nodemanage', scxxgl: 'equipOver',
  jrgl: 'link', tycfqgl: 'procSet', jsgl: 'dashboard', yhgl: 'user', xtrz: 'file'
}

/** DCSY固定路由，只注册锂硫电池溯源系统实际使用的页面。 */
export const constantRoutes = [
  { path: '/login', component: () => import('@/views/login/login.vue'), hidden: true },
  { path: '/modifyPas', component: () => import('@/views/personMana/modifyPas.vue'), hidden: true },
  { path: '/dcDetail', component: () => import('@/views/lldcsy/dcjkgl/dcDetail.vue'), name: 'dcDetail', hidden: true },
  // 数据接入的“管理”在新窗口打开，固定注册以支持刷新和重新登录。
  { path: '/equipManaPage', component: () => import('@/views/equipMana/equipManaPage.vue'), name: 'equipManaPage', hidden: true, meta: { title: '电池管理' } },
  {
    path: '/collection',
    component: () => import('@/collection/layouts/MainLayout.vue'),
    hidden: true,
    meta: { title: '采集云平台', collectionCloud: true, routeContainer: true },
    children: [
      {
        path: '',
        name: 'collection-home',
        component: () => import('@/collection/views/HomeView.vue'),
        meta: { title: '工作台', collectionCloud: true }
      },
      {
        path: 'gateway/:modelSlug/:gatewayId',
        name: 'gateway-detail',
        component: () => import('@/collection/views/GatewayDetailEntryView.vue'),
        meta: { title: '网关详情', collectionCloud: true }
      },
      {
        path: 'gateway/:gatewayId',
        name: 'gateway-detail-legacy',
        component: () => import('@/collection/views/GatewayLegacyRedirectView.vue'),
        meta: { title: '网关详情', collectionCloud: true }
      },
      {
        path: 'stream/:streamCode',
        name: 'stream-detail',
        component: () => import('@/collection/views/StreamDetailView.vue'),
        meta: { title: '数据池详情', collectionCloud: true }
      }
      // {
      //   path: 'ttdc-templates',
      //   name: 'ttdc-templates',
      //   component: () => import('@/collection/views/TtdcTemplateView.vue'),
      //   meta: {
      //     title: 'TTDC公共模板',
      //     collectionCloud: true,
      //     collectionPermission: TTDC_TEMPLATE_PERMISSIONS.view
      //   }
      // }
    ]
  },
  { path: '/', redirect: '/jsc', hidden: true }
]

/** DCSY权限路由，保留原侧边栏标记以维持现有页面布局。 */
export const asyncRoutes = [
  dcsyPage('/jsc', 'jsc', '驾驶舱', () => import('@/views/lldcsy/jsc/jsc.vue'), 'jsc'),
  dcsyPage('/dczt', 'dczt', '电池状态', () => import('@/views/lldcsy/smzqgl/dczt.vue'), 'qsmzqgl'),
  dcsyPage('/gjzz', 'gjzz', '轨迹追踪', () => import('@/views/lldcsy/smzqgl/gjzz.vue'), 'qsmzqgl'),
  dcsyPage('/ywgl', 'ywgl', '运维管理', () => import('@/views/lldcsy/smzqgl/ywgl.vue'), 'qsmzqgl'),
  dcsyPage('/jkztjc', 'jkztjc', '健康状态监测', () => import('@/views/lldcsy/dcjkgl/jkztjc.vue'), 'dcjkgl'),
  dcsyPage('/lssjcx', 'lssjcx', '历史数据查询', () => import('@/views/lldcsy/dcjkgl/lssjcx.vue'), 'dcjkgl'),
  {
    path: '/gjxx', name: 'layout-gjxx', component: Layout, redirect: '/gjxxssgj', dcjkgl: true,
    meta: { title: '告警信息', icon: 'alarmLog', routeContainer: true },
    children: [
      dcsyChild('/gjxxssgj', 'gjxxssgj', '实时告警', () => import('@/views/lldcsy/dcjkgl/gjxxssgj.vue'), 'dcjkgl'),
      dcsyChild('/gjxxlsgj', 'gjxxlsgj', '历史告警', () => import('@/views/lldcsy/dcjkgl/gjxxlsgj.vue'), 'dcjkgl')
    ]
  },
  {
    path: '/yjxx', name: 'layout-yjxx', component: Layout, redirect: '/yjxxssyj', dcjkgl: true,
    meta: { title: '预警信息', icon: 'warning', routeContainer: true },
    children: [
      dcsyChild('/yjxxssyj', 'yjxxssyj', '实时预警', () => import('@/views/lldcsy/dcjkgl/yjxxssyj.vue'), 'dcjkgl'),
      dcsyChild('/yjxxlsyj', 'yjxxlsyj', '历史预警', () => import('@/views/lldcsy/dcjkgl/yjxxlsyj.vue'), 'dcjkgl')
    ]
  },
  {
    path: '/jcxxgl', name: 'layout-jcxxgl', component: Layout, redirect: '/jcxxgl/kfxxgl', jcxxgl: true,
    meta: { title: '基础信息管理', icon: 'basicmanage', routeContainer: true },
    children: [
      dcsyChild('kfxxgl', 'kfxxgl', '库房信息管理', () => import('@/views/lldcsy/jcxxgl/kfxxgl.vue'), 'jcxxgl'),
      dcsyChild('/zdxxgl/index', 'zdxxgl', '站点信息管理', () => import('@/views/lldcsy/jcxxgl/zdxxgl.vue'), 'jcxxgl'),
      dcsyChild('/scxxgl/index', 'scxxgl', '电池出厂信息', () => import('@/views/lldcsy/jcxxgl/scxxgl.vue'), 'jcxxgl'),
      {
        path: '/dcsjgl', redirect: '/dcsjgl/jrgl', alwaysShow: true, jcxxgl: true,
        meta: { title: '电池数据管理', icon: 'collectCloud' },
        children: [
          {
            path: 'wgpz',
            name: 'collection-launcher',
            component: () => import('@/views/lldcsy/jcxxgl/wgpz.vue'),
            jcxxgl: true,
            meta: { title: '网关配置', icon: 'procSet', noCache: true }
          },
          dcsyChild('jrgl', 'jrgl', '数据接入', () => import('@/views/lldcsy/jcxxgl/jrgl.vue'), 'jcxxgl'),
          dcsyChild('tycfqgl', 'tycfqgl', '通用触发器管理', () => import('@/views/lldcsy/jcxxgl/tycfqgl.vue'), 'jcxxgl')
        ]
      },
      {
        path: '/rygl', redirect: '/rygl/yhgl', alwaysShow: true, jcxxgl: true,
        meta: { title: '人员管理', icon: 'peoples' },
        children: [
          dcsyChild('jsgl', 'jsgl', '角色管理', () => import('@/views/lldcsy/jcxxgl/jsgl.vue'), 'jcxxgl'),
          dcsyChild('yhgl', 'yhgl', '用户管理', () => import('@/views/lldcsy/jcxxgl/yhgl.vue'), 'jcxxgl')
        ]
      },
      {
        path: '/jcsj', redirect: '/jcsj/xtrz', alwaysShow: true, jcxxgl: true,
        meta: { title: '基础数据', icon: 'chart' },
        children: [dcsyChild('xtrz', 'xtrz', '系统日志', () => import('@/views/lldcsy/jcxxgl/xtrz.vue'), 'jcxxgl')]
      }
    ]
  }
]

function dcsyPage(path, name, title, component, section) {
  // 父路由使用独立名称，退出时可整条移除；页面名称和地址保持原样。
  return { path, name: `layout-${name}`, component: Layout, meta: { routeContainer: true }, [section]: true, children: [dcsyChild('', name, title, component, section)] }
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
