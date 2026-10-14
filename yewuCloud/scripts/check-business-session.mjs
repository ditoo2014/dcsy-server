import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { createRouter, createMemoryHistory } from 'vue-router'
import { transformSync } from 'esbuild'

// 使用真实路由匹配器和源码，只替换浏览器、后端请求及状态仓库；不访问生产环境。
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const read = name => fs.readFileSync(path.join(root, name), 'utf8')
function evaluate(source, dependencies = {}, globals = {}) {
  const module = { exports: {}}
  const code = transformSync(source, { format: 'cjs' }).code
  const require = name => {
    if (Object.hasOwn(dependencies, name)) return dependencies[name]
    throw new Error(`未声明的测试依赖：${name}`)
  }
  new Function('module', 'exports', 'require', ...Object.keys(globals), code)(module, module.exports, require, ...Object.values(globals))
  return module.exports
}

let checks = 0
async function check(name, run) {
  await run()
  checks++
  console.log(`通过：${name}`)
}

const Layout = { name: 'Layout' }
const dynamicRouteState = evaluate(read('src/utils/dynamic-route-state.js'))
const routeModule = evaluate(read('src/router/index.js'), {
  'vue-router': { createRouter, createWebHashHistory: createMemoryHistory },
  '@/layout': Layout,
  '@/utils/dynamic-route-state.js': dynamicRouteState
})
const router = routeModule.default
const platform = evaluate(read('src/constants/platform.js'))
const accessibility = evaluate(read('src/utils/resolve-accessible-route.js'), {
  '@/constants/platform': platform,
  '@/layout': Layout
})

// 按真实路由表收集页面及完整地址，覆盖基础信息管理中的多级菜单。
function pageRoutes(routes, parentPath = '') {
  return routes.flatMap(route => {
    const fullPath = route.path.startsWith('/') ? route.path : `${parentPath.replace(/\/$/, '')}/${route.path}`
    if (route.children?.length) return pageRoutes(route.children, fullPath)
    return [{ path: fullPath.replace(/\/$/, '') || '/', name: route.name, component: route.component }]
  })
}
function routeSnapshot() {
  return router.getRoutes().map(route => `${route.path}:${String(route.name || '')}`).sort()
}
const baseline = routeSnapshot()
const pages = pageRoutes(routeModule.asyncRoutes)

await check('动态顶层路由均有唯一名称，确保整条删除', () => {
  const names = routeModule.asyncRoutes.map(route => route.name)
  assert.ok(names.every(Boolean), '动态顶层路由不能遗留匿名 Layout')
  assert.equal(new Set(names).size, names.length)
  assert.equal(pages.length, 19)
})

await check('连续三轮登录与退出后，19 个业务页面均匹配到实际组件', () => {
  for (let round = 1; round <= 3; round++) {
    routeModule.asyncRoutes.forEach(route => router.addRoute(route))
    dynamicRouteState.markDynamicRoutesReady()
    assert.equal(dynamicRouteState.isDynamicRoutesReady(), true)
    for (const page of pages) {
      const resolved = router.resolve(page.path)
      const leaf = resolved.matched.at(-1)
      assert.equal(leaf?.name, page.name, `第 ${round} 轮页面匹配错误：${page.path}`)
      assert.equal(leaf.components.default, page.component, `第 ${round} 轮只匹配了布局：${page.path}`)
      assert.equal(resolved.matched[0].components.default, Layout)
      assert.ok(accessibility.isRouteAccessible(router, page.path), page.path)
    }
    for (const redirectPath of ['/', '/gjxx', '/yjxx', '/jcxxgl', '/dcsjgl', '/rygl', '/jcsj']) {
      assert.ok(accessibility.isRouteAccessible(router, redirectPath), `合法跳转入口不可访问：${redirectPath}`)
    }
    routeModule.resetRouter()
    assert.equal(dynamicRouteState.isDynamicRoutesReady(), false, '退出后动态路由就绪状态必须同步复位')
    assert.deepEqual(routeSnapshot(), baseline, `第 ${round} 轮退出后残留了动态父路由`)
    // 重复清理应安全，采集云等固定路由必须仍存在。
    routeModule.resetRouter()
    assert.deepEqual(routeSnapshot(), baseline)
    assert.equal(router.resolve('/collection').matched.at(-1)?.name, 'collection-home')
  }
})

await check('可访问性拒绝只有布局的空壳，允许真实页面和有效跳转', () => {
  const plainPage = { name: 'TestPage' }
  const isolatedRouter = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/empty', name: 'layout-empty', component: Layout, meta: { routeContainer: true }},
      { path: '/page', component: plainPage },
      { path: '/empty-child', component: Layout, children: [{ path: '', name: 'empty-child' }] },
      { path: '/entry', name: 'layout-entry', component: Layout, redirect: '/page' },
      { path: '/object-entry', redirect: { path: '/page' }},
      { path: '/named-entry', redirect: { name: 'named-page' }},
      { path: '/named-page', name: 'named-page', component: plainPage }
    ]
  })
  assert.equal(accessibility.isRouteAccessible(isolatedRouter, '/empty'), false)
  assert.equal(accessibility.isRouteAccessible(isolatedRouter, '/empty-child'), false)
  for (const routePath of ['/page', '/named-page', '/entry', '/object-entry', '/named-entry']) {
    assert.equal(accessibility.isRouteAccessible(isolatedRouter, routePath), true, routePath)
  }
})

function deferred() {
  let resolve
  let reject
  const promise = new Promise((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

await check('退出并发只清理一次，完成后重新登录仍可再次退出', async() => {
  const storage = new Map([['dicloud:last-user-activity-at', '1']])
  const dispatches = []
  const redirects = []
  const notifications = []
  const errors = []
  const session = evaluate(read('src/utils/session-expiration.js'), {
    '@/store': {
      dispatch(action) {
        assert.equal(action, 'user/sessionTimeout')
        const work = deferred()
        dispatches.push(work)
        return work.promise
      }
    },
    '@/router': {
      replace(location) {
        redirects.push(location)
        return Promise.resolve()
      }
    },
    '@/utils/auth.js': { getToken: () => 'local-test-token' },
    '@/utils/session-logout': { notifyBackendLogout: token => notifications.push(token) }
  }, {
    window: { location: { hash: '#/jkztjc', origin: 'https://local.example', pathname: '/', replace: () => assert.fail('应通过路由器跳转登录') }},
    localStorage: {
      setItem: (key, value) => storage.set(key, value),
      removeItem: key => storage.delete(key)
    },
    console: { error: (...args) => errors.push(args) }
  })

  const first = session.expireSession({ notifyBackend: true })
  assert.equal(first, session.expireSession({ notifyBackend: true }))
  await Promise.resolve()
  assert.equal(dispatches.length, 1)
  assert.deepEqual(notifications, ['local-test-token'])
  assert.ok(storage.has(session.SESSION_TIMEOUT_EVENT_KEY))
  assert.equal(storage.has(session.LAST_ACTIVITY_KEY), false)
  dispatches[0].resolve()
  await first
  assert.deepEqual(redirects, [{ path: '/login', query: { redirect: '/jkztjc' }}])

  const second = session.expireSession({ broadcast: false, redirectPath: '/dczt' })
  await Promise.resolve()
  assert.notEqual(second, first)
  assert.equal(dispatches.length, 2, '上一次退出完成后必须释放并发标记')
  dispatches[1].resolve()
  await second
  assert.deepEqual(redirects.at(-1), { path: '/login', query: { redirect: '/dczt' }})

  // 清理失败仍跳转登录，并允许下次重试，避免永久卡在旧会话 Promise。
  const failed = session.expireSession({ broadcast: false, redirectPath: '/jsc' })
  await Promise.resolve()
  dispatches[2].reject(new Error('模拟本地状态清理失败'))
  await failed
  assert.equal(errors.length, 1)
  assert.deepEqual(redirects.at(-1), { path: '/login', query: { redirect: '/jsc' }})
  const retry = session.expireSession({ broadcast: false, redirectPath: '/' })
  await Promise.resolve()
  assert.equal(dispatches.length, 4)
  dispatches[3].resolve()
  await retry
  assert.equal(redirects.length, 4)
})

console.log(`业务云登录与路由专项：${checks} 项通过。`)
