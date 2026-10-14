import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { parse, compileTemplate } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'

// 内存执行真实模块；地图、请求仅用测试替身，不访问地图平台或生产后端。
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const read = name => fs.readFileSync(path.join(root, name), 'utf8')
function evaluate(source, dependencies = {}, globals = {}, env = {}) {
  const module = { exports: {} }
  const code = transformSync(source, { format: 'cjs', define: { 'import.meta.env': JSON.stringify(env) } }).code
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
function loader(load, env = { VITE_AMAP_KEY: 'local-example', VITE_AMAP_SECURITY_CODE: 'local-example' }) {
  const window = {}
  return { window, ...evaluate(read('src/utils/amap-loader.js'), { '@amap/amap-jsapi-loader': { load } }, { window }, env) }
}

await check('地图未配置时返回中文错误且不请求平台', async () => {
  const api = loader(() => assert.fail('不应请求地图平台'), {})
  await assert.rejects(api.loadBusinessMap(), /未配置高德地图/)
})
await check('并发地图组件只加载一次并传入配置', async () => {
  let calls = 0
  const sdk = {}
  const api = loader(async options => {
    calls++
    assert.equal(options.key, 'local-example')
    assert.ok(options.plugins.includes('AMap.Geocoder'))
    return sdk
  })
  assert.equal(api.loadBusinessMap(), api.loadBusinessMap())
  assert.equal(await api.loadBusinessMap(), sdk)
  assert.equal(calls, 1)
  assert.equal(api.window._AMapSecurityConfig.securityJsCode, 'local-example')
})
await check('地图加载失败后可再次加载', async () => {
  let attempts = 0
  const api = loader(() => ++attempts === 1 ? Promise.reject(new Error('offline')) : Promise.resolve('ready'))
  await assert.rejects(api.loadBusinessMap(), /地图加载失败/)
  assert.equal(await api.loadBusinessMap(), 'ready')
})

function mapComponent(file, loadBusinessMap = async () => ({}), AMap = {}) {
  const { descriptor, errors } = parse(read(file), { filename: file })
  assert.deepEqual(errors, [])
  assert.deepEqual(compileTemplate({ source: descriptor.template.content, filename: file, id: 'map-test' }).errors, [])
  const options = evaluate(descriptor.script.content, {
    vue: { markRaw: value => value },
    '@/utils/amap-loader': { loadBusinessMap },
    '@/utils/container-resize.js': evaluate(read('src/utils/container-resize.js'))
  }, { AMap }).default
  const messages = []
  const state = { ...options.data(), mapMsg: {}, $message: { error: msg => messages.push(msg), warning: msg => messages.push(msg) } }
  for (const [name, fn] of Object.entries(options.methods)) state[name] = fn.bind(state)
  return { state, messages }
}
await check('选址地图未就绪时查询不会触发空对象异常', () => {
  const { state, messages } = mapComponent('src/components/a_myComp/selectMap/SelectMapArea.vue')
  state.searchAddrMsg()
  state.searchSite()
  assert.equal(messages.length, 2)
})
await check('地图加载失败释放加载状态并提示', async () => {
  for (const [file, method] of [
    ['mapDriving/MapDriving.vue', 'initMap'], ['mapComplex/DetailMap.vue', 'createMarker'], ['selectMap/SelectMapArea.vue', 'createMarker']
  ]) {
    const { state, messages } = mapComponent(`src/components/a_myComp/${file}`, async () => { throw new Error('地图加载失败') })
    await state[method]()
    assert.deepEqual(messages, ['地图加载失败'])
    if ('mapLoading' in state) assert.equal(state.mapLoading, false)
  }
})
await check('组件销毁后不初始化异步返回的地图', async () => {
  for (const [file, method] of [
    ['mapDriving/MapDriving.vue', 'initMap'], ['mapComplex/DetailMap.vue', 'createMarker'], ['selectMap/SelectMapArea.vue', 'createMarker']
  ]) {
    const { state } = mapComponent(`src/components/a_myComp/${file}`)
    state.mapDisposed = true
    await state[method]()
    assert.equal(Object.hasOwn(state, 'map') ? state.map : state.mapKey, null)
  }
})
await check('区域查询乱序不会恢复旧地图', async () => {
  const callbacks = []
  const created = []
  const { state } = mapComponent('src/components/a_myComp/mapComplex/DetailMap.vue', async () => ({}), {
    DistrictSearch: class { search(name, callback) { callbacks.push(callback) } },
    Map: class { constructor(id, options) { created.push(options.center) } setFitView() {} }
  })
  state.replaceMarkers = () => {}
  await state.createMarker()
  await state.createMarker()
  const result = lng => ({ districtList: [{ center: { lng, lat: 30 }, boundaries: [] }] })
  callbacks[1]('complete', result(104))
  callbacks[0]('complete', result(105))
  assert.deepEqual(created, [[104, 30]])
})
await check('19 个业务页面都有真实存在的原菜单图标', () => {
  const router = evaluate(read('src/router/index.js'), {
    'vue-router': { createRouter: options => options, createWebHashHistory: () => ({}) },
    '@/utils/dynamic-route-state.js': { resetDynamicRoutesReady() {} },
    '@/layout': {}, '@/collection/constants/permissions.js': { TTDC_TEMPLATE_PERMISSIONS: {} }
  })
  const leaves = []
  const visit = routes => routes.forEach(route => route.children ? visit(route.children) : leaves.push(route))
  visit(router.asyncRoutes)
  assert.equal(leaves.length, 19)
  for (const route of leaves) {
    assert.ok(route.meta.icon, route.name)
    assert.ok(['svg', 'mySvg'].some(folder => fs.existsSync(path.join(root, `src/icons/${folder}/${route.meta.icon}.svg`))), route.meta.icon)
  }
})
await check('模板下载和运维导出请求保留全部筛选并返回文件流', async () => {
  const requests = []
  const api = evaluate(read('src/api/lldcsy.js'), { '@/api/api.js': { JsonPost: (...args) => requests.push(args) } })
  api.siteImportTemplate()
  api.warehouseImportTemplate()
  const filters = { equipmentCode: '测试电池', equipmentTypeName: '钠电池', minimumCount: 5 }
  api.equipmentmitrcdExport({ ...filters, startTime: 100, endTime: 200 })
  assert.equal(requests[0][0], '/business/site/import/template')
  assert.equal(requests[1][0], '/business/warehouse/import/template')
  assert.equal(requests[2][0], '/business/equipmentmitrcd/export?startTime=100&endTime=200')
  assert.deepEqual(requests[2][1], { equipmentId: undefined, ...filters })
  requests.forEach(request => assert.equal(request[2].responseType, 'blob'))
})
console.log(`业务云集成专项：${checks} 项通过。`)
