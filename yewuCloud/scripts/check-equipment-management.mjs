import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { createRouter, createMemoryHistory } from 'vue-router'
import { transformSync } from 'esbuild'
import { parse, compileTemplate } from '@vue/compiler-sfc'
import { parse as parseJavaScript } from '@babel/parser'

// 离线执行真实路由、守卫、组件及接口封装，只替换浏览器和网络，不访问生产数据库。
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const read = name => fs.readFileSync(path.join(root, name), 'utf8')
const clone = value => structuredClone(value)
const response = rows => ({ code: 200, return: { data: rows } })
const flush = () => new Promise(resolve => setImmediate(resolve))
let checks = 0
async function check(name, run) {
  await run()
  checks++
  console.log(`通过：${name}`)
}

function evaluate(source, dependencies = {}, globals = {}) {
  const module = { exports: {} }
  const code = transformSync(source, { format: 'cjs', define: { 'import.meta.env': '{}' } }).code
  const require = name => {
    if (Object.hasOwn(dependencies, name)) return dependencies[name]
    throw new Error(`未声明的测试依赖：${name}`)
  }
  new Function('module', 'exports', 'require', ...Object.keys(globals), code)(module, module.exports, require, ...Object.values(globals))
  return module.exports
}

function component(relativePath, overrides = {}, globals = {}) {
  const source = read(relativePath)
  const { descriptor, errors } = parse(source, { filename: relativePath })
  assert.deepEqual(errors, [])
  assert.deepEqual(compileTemplate({ source: descriptor.template.content, filename: relativePath, id: 'equipment-management-test' }).errors, [])
  const script = descriptor.script.content
  const ast = parseJavaScript(script, { sourceType: 'module' })
  const bindings = { ...overrides, ...globals }
  for (const node of ast.program.body.filter(node => node.type === 'ImportDeclaration')) {
    for (const specifier of node.specifiers) {
      const name = specifier.local.name
      if (!(name in bindings)) bindings[name] = () => assert.fail(`未配置的组件测试调用：${name}`)
    }
  }
  let body = script
  for (const node of [...ast.program.body].reverse()) {
    if (node.type === 'ImportDeclaration') body = body.slice(0, node.start) + body.slice(node.end)
  }
  body = body.replace('export default', 'return').replaceAll('import.meta.env.DEV', 'false')
  return new Function(...Object.keys(bindings), body)(...Object.values(bindings))
}

function instance(options, properties = {}) {
  const vm = { $nextTick: async () => {}, ...properties }
  Object.assign(vm, options.data?.call(vm) || {})
  for (const [name, fn] of Object.entries(options.methods || {})) vm[name] = fn.bind(vm)
  for (const [name, fn] of Object.entries(options.computed || {})) {
    Object.defineProperty(vm, name, { get: () => fn.call(vm), configurable: true })
  }
  return vm
}

const routeState = evaluate(read('src/utils/dynamic-route-state.js'))
const routeModule = evaluate(read('src/router/index.js'), {
  'vue-router': { createRouter, createWebHashHistory: createMemoryHistory },
  '@/layout': { name: 'Layout' },
  '@/utils/dynamic-route-state.js': routeState
})
const router = routeModule.default
const platform = evaluate(read('src/constants/platform.js'))
const accessibility = evaluate(read('src/utils/resolve-accessible-route.js'), { '@/constants/platform': platform })
const passwordState = evaluate(read('src/utils/password-change-state.js'))
const equipmentRow = { id: 42, equipmentCode: 'battery_01', equipmentName: '站点 & #? 01', apiKey: '旧数据池' }
const rowQuery = JSON.stringify(equipmentRow)

await check('原管理按钮以命名路由打开新窗口，完整保留中文和特殊字符设备参数', () => {
  const opened = []
  const options = component('src/views/equipMana/equipMana.vue', {}, { window: { open: (...args) => opened.push(args) } })
  options.methods.goDetail.call({ $router: router }, equipmentRow)
  assert.equal(opened.length, 1)
  assert.equal(opened[0][1], '_blank')
  const resolved = router.resolve(opened[0][0])
  assert.equal(resolved.name, 'equipManaPage')
  assert.equal(resolved.path, '/equipManaPage')
  assert.deepEqual(JSON.parse(resolved.query.row), equipmentRow)
  assert.ok(accessibility.isRouteAccessible(router, resolved.path))
  assert.equal(accessibility.resolveAccessiblePath(router, resolved.path, [], { platform: platform.PLATFORM.DCSY_XC }), resolved.path)
})

await check('三轮新窗口冷启动及重新登录保留管理页和参数，季度到期优先跳本人改密', async () => {
  let guard
  router.beforeEach = callback => { guard = callback }
  router.afterEach = () => {}
  let roles = []
  let required = false
  const calls = []
  const store = {
    getters: { get roles() { return roles } },
    state: { permission: { addRoutes: routeModule.asyncRoutes } },
    dispatch: async action => {
      calls.push(action)
      if (action === 'user/checkPasswordChange') return { passwordChangeRequired: required }
      if (action === 'user/getInfo') { roles = ['manager']; return { roles, routers: routeModule.asyncRoutes } }
      if (action === 'permission/generateRoutes') return routeModule.asyncRoutes
      throw new Error(`意外的状态请求：${action}`)
    }
  }
  evaluate(read('src/permission.js'), {
    './router': router, './store': store,
    '@/utils/element-shim': { Message: { error: message => assert.fail(message) } },
    nprogress: { configure() {}, start() {}, done() {} }, 'nprogress/nprogress.css': {},
    '@/utils/auth.js': { getToken: () => '离线测试令牌' }, '@/utils/get-page-title': () => '',
    '@/config.js': { platform: platform.PLATFORM.DCSY_XC }, '@/constants/platform': platform,
    '@/utils/resolve-accessible-route.js': accessibility,
    '@/utils/banner-route.js': { resolveBannerIdFromPath: () => '' },
    '@/collection/utils/authAccess.js': { ensureTolinkAccess: () => assert.fail('不应加载采集云权限') },
    '@/collection/stores/auth.js': { useAuthStore: () => ({ hasAnyPermission: () => true }) },
    '@/utils/dynamic-route-state.js': routeState, '@/utils/password-change-state.js': passwordState
  }, { window: {}, document: {}, localStorage: { setItem() {} } })
  const navigate = async () => {
    const targets = []
    await guard(router.resolve({ name: 'equipManaPage', query: { row: rowQuery } }), {}, target => targets.push(target))
    assert.equal(targets.length, 1)
    return targets[0]
  }
  for (let round = 1; round <= 3; round++) {
    routeModule.resetRouter(); roles = []; calls.length = 0
    assert.ok(router.hasRoute('equipManaPage'))
    const target = await navigate()
    assert.equal(target.path, '/equipManaPage')
    assert.equal(target.query.row, rowQuery)
    assert.equal(target.replace, true)
    assert.deepEqual(calls, ['user/checkPasswordChange', 'user/getInfo', 'permission/generateRoutes'])
    assert.equal(await navigate(), undefined)
  }
  required = true; roles = []; calls.length = 0
  assert.deepEqual(await navigate(), { path: '/modifyPas', replace: true })
  assert.deepEqual(calls, ['user/checkPasswordChange'])
})

function apiHarness(routes = {}) {
  const calls = []
  const notices = []
  const Message = notice => notices.push(notice)
  for (const type of ['success', 'error', 'warning']) Message[type] = message => notices.push({ type, message })
  const api = evaluate(read('src/api/api.js'), {
    '@/utils/request.js': async params => {
      calls.push(clone(params))
      const route = routes[params.url.split('?')[0]]
      assert.ok(route, `未声明的测试接口：${params.url}`)
      return typeof route === 'function' ? route(params) : clone(route)
    },
    '@/utils/element-shim': { Message },
    '@/utils/auth.js': { getToken: () => '离线测试令牌' },
    '@/utils/session-expiration': { expireSession: () => assert.fail('不应触发会话过期') },
    '@/utils/password-change.js': { requirePasswordChange: () => assert.fail('不应触发改密拦截') },
    '@/utils/password-change-state.js': passwordState
  })
  return {
    calls, notices, Message,
    equipment: evaluate(read('src/api/equipMana.js'), { '@/api/api.js': api }),
    trigger: evaluate(read('src/api/trigger.js'), { '@/api/api.js': api })
  }
}

await check('管理页拒绝错误地址参数和缺少设备ID，查询失败或设备不存在不显示空标签', async () => {
  const notices = []
  const requested = []
  let result = response([{ id: 42, apiKey: '当前数据池', equipmentName: '最新设备名称' }])
  const options = component('src/views/equipMana/equipManaPage.vue', {
    Message: { error: message => notices.push(message) },
    findEquipmentListExactMatch: async params => {
      requested.push(params)
      if (result instanceof Error) throw result
      return result
    }
  })
  for (const row of ['{', '{}', 'null', '[]', '{"id":0}', '{"id":-1}', '{"id":"bad"}', '{"id":9007199254740992}']) {
    const vm = instance(options, { $route: { query: { row } } })
    options.mounted.call(vm)
    assert.equal(vm.showTabs, false)
    assert.ok(vm.loadError)
  }
  assert.equal(requested.length, 0)
  const valid = instance(options, { $route: { query: { row: rowQuery } } })
  options.mounted.call(valid)
  await flush()
  assert.deepEqual(requested, [{ id: 42 }])
  assert.equal(valid.showTabs, true)
  assert.equal(valid.form.apiKey, '当前数据池', '应使用后端当前配置，不能使用地址中的过期数据池')
  assert.equal(valid.tabLabel, '数据表')
  for (result of [response([]), response([{ id: 999 }]), new Error('设备查询失败')]) {
    const vm = instance(options, { $route: { query: { row: rowQuery } } })
    options.mounted.call(vm)
    await flush()
    assert.equal(vm.showTabs, false)
    assert.ok(vm.loadError)
  }
  assert.deepEqual(notices, ['设备查询失败'])
})

await check('关闭管理页后晚返回的设备查询不会重新显示标签', async () => {
  let release
  const options = component('src/views/equipMana/equipManaPage.vue', {
    findEquipmentListExactMatch: () => new Promise(resolve => { release = resolve })
  })
  const vm = instance(options, { $route: { query: { row: rowQuery } } })
  options.mounted.call(vm)
  options.beforeUnmount.call(vm)
  release(response([{ id: 42, apiKey: '新数据池' }]))
  await flush()
  assert.equal(vm.showTabs, false)
})

const ccpcData = evaluate(read('src/utils/ccpcData.js'), {
  '@/utils/utils.js': { isNumber: value => Number.isFinite(Number(value)), formatDateString: value => value }
})
const dataRows = [{
  id: 7, sortNum: 1, equipmentId: 42, equipmentName: '设备42', name: '电压',
  labelId: 'voltage', labelName: '原始电压',
  contentDataDetails: [{ decimalPlaces: 0, limitValue: 1, color: '#7AF7BE', extraSetting: '保留', statusLightArr: [{ key: '>', value: 5, color: '#FF0000' }] }]
}]
function dataHarness(routes = {}) {
  const api = apiHarness({
    'business/datadetails/find': response(dataRows),
    'business/datadetails/add': { code: 200 },
    'business/datadetails/update': { code: 200 },
    'business/datadetails/delete': { code: 200 },
    '/tolink/stream/getlabel': { code: 200, return: { list: [{ labelId: 'voltage', name: '原始电压' }] } },
    ...routes
  })
  const options = component('src/views/equipMana/components/DataShee.vue', {
    ...api.equipment, ...ccpcData, Message: api.Message,
    markRaw: value => value, meAlert: async () => true, checkIsColor: value => /^#[\da-f]{6}$/i.test(value)
  })
  const emitted = []
  const vm = instance(options, {
    formData: { id: 42, equipmentName: '设备42', apiKey: '数据池42' },
    $refs: { dataForm: { validate: async () => true, clearValidate() {} } },
    $emit: (...args) => emitted.push(args)
  })
  return { ...api, vm, options, emitted }
}

await check('数据表查询和属性编辑保留0位小数，保存数字类型并保留未编辑属性', async () => {
  const h = dataHarness()
  await h.vm.getDataList()
  assert.equal(h.vm.tableData[0].decimalPlaces, 0)
  assert.equal(h.vm.tableData[0].contentDataDetails[0].decimalPlaces, 0)
  await h.vm.setMsg(h.vm.tableData[0])
  assert.equal(h.vm.form.decimalPlaces, 0)
  h.vm.form.sortNum = '1'
  h.vm.form.decimalPlaces = '0'
  h.vm.form.limitValue = '2.5'
  await h.vm.setData()
  const saved = h.calls.find(item => item.url === 'business/datadetails/update').data
  assert.equal(saved.sortNum, 1)
  assert.equal(saved.contentDataDetails[0].decimalPlaces, 0)
  assert.equal(saved.contentDataDetails[0].limitValue, 2.5)
  assert.equal(saved.contentDataDetails[0].extraSetting, '保留')
  assert.equal('name' in saved.contentDataDetails[0].statusLightArr[0], false)
  assert.equal('isMsg' in saved.contentDataDetails[0].statusLightArr[0], false)
  assert.equal(h.vm.showDrawer, false)
  assert.equal(h.vm.saving, false)
  assert.equal(h.vm.pendingRequests, 0)
  assert.ok(h.notices.some(item => item.type === 'success'))
  assert.deepEqual(h.emitted, [['data-change']])
})

await check('数据点新增和修改使用当前设备及标签，200响应完成删除并刷新列表', async () => {
  const h = dataHarness()
  await h.vm.getDataList()
  await h.vm.add()
  h.vm.form.labelId = 'voltage'
  h.vm.form.sortNum = '2'
  h.vm.form.name = '新增电压'
  await h.vm.setData()
  const created = h.calls.find(item => item.url === 'business/datadetails/add').data
  assert.equal(created.equipmentId, 42)
  assert.equal(created.labelName, '原始电压')
  assert.equal(created.sortNum, 2)
  await h.vm.modify(h.vm.tableData[0])
  h.vm.form.name = '修改名称'
  await h.vm.setData()
  assert.equal(h.calls.find(item => item.url === 'business/datadetails/update').data.name, '修改名称')
  await h.vm.remove(h.vm.tableData[0])
  assert.deepEqual(h.calls.find(item => item.url === 'business/datadetails/delete').data, { id: 7 })
  assert.equal(h.emitted.length, 3)
  assert.ok(h.notices.some(item => item.type === 'success' && item.message.includes('删除成功')))
})

await check('数据点持久化非200或网络失败保留抽屉，不提示成功或触发数据已变更', async () => {
  for (const route of [{ code: 500, msg: '保存失败' }, () => { throw new Error('网络失败') }]) {
    const h = dataHarness({ 'business/datadetails/update': route })
    await h.vm.getDataList()
    await h.vm.setMsg(h.vm.tableData[0])
    await h.vm.setData()
    assert.equal(h.vm.showDrawer, true)
    assert.equal(h.vm.saving, false)
    assert.equal(h.vm.pendingRequests, 0)
    assert.equal(h.notices.some(item => item.type === 'success'), false)
    assert.ok(h.notices.some(item => item.type === 'error'))
    assert.deepEqual(h.emitted, [])
  }
})

await check('数据查询异常保留已有列表并禁止写操作，真实空列表仍允许新增', async () => {
  let result = response([
    ...dataRows,
    { ...dataRows[0], id: 8, sortNum: 2, name: '电流', labelId: 'current', labelName: '原始电流' }
  ])
  const h = dataHarness({ 'business/datadetails/find': () => clone(result) })
  await h.vm.getDataList()
  assert.equal(h.vm.dataLoaded, true)
  await h.vm.setMsg(h.vm.tableData[0])
  const displayed = clone(h.vm.tableData)
  const allRows = clone(h.vm.allTableData)
  result = { code: 200, return: null }
  await h.vm.getDataList()
  assert.equal(h.vm.dataLoaded, false)
  assert.deepEqual(h.vm.tableData, displayed)
  assert.deepEqual(h.vm.allTableData, allRows)
  assert.ok(h.notices.some(item => item.type === 'error' && /[\u4e00-\u9fff]/.test(item.message)))
  // 刷新失败时已有编辑窗口也不能绕过保护继续提交。
  await h.vm.setData()
  assert.equal(h.vm.showDrawer, true)
  h.vm.showDrawer = false
  await h.vm.add()
  assert.equal(h.vm.showDrawer, false)
  await h.vm.remove(h.vm.tableData[0])
  h.vm.selectedTableData = [...h.vm.tableData]
  await h.vm.batchRemove()
  await h.vm.persistRowOrder({ oldIndex: 1, newIndex: 0 })
  assert.deepEqual(h.vm.tableData, displayed)
  assert.equal(h.calls.some(item => /business\/datadetails\/(add|update|delete)$/.test(item.url)), false)
  assert.equal(h.notices.some(item => item.type === 'success'), false)
  assert.deepEqual(h.emitted, [])
  assert.equal(h.vm.pendingRequests, 0)
  result = response([])
  await h.vm.getDataList()
  assert.equal(h.vm.dataLoaded, true)
  assert.deepEqual(h.vm.tableData, [])
  assert.deepEqual(h.vm.allTableData, [])
  await h.vm.add()
  assert.equal(h.vm.showDrawer, true)
  assert.equal(h.vm.form.sortNum, 1)
})

await check('复制数据点属性深复制状态灯，拖拽顺序保存所有受影响行', async () => {
  const h = dataHarness()
  await h.vm.getDataList()
  h.vm.form = { copyDataPointId: 7 }
  h.vm.copyDataPointChange()
  assert.equal(h.vm.form.decimalPlaces, 0)
  h.vm.form.statusLightArr[0].color = '#000000'
  h.vm.form.contentDataDetails[0].extraSetting = '取消编辑'
  assert.equal(h.vm.tableData[0].statusLightArr[0].color, '#FF0000')
  assert.equal(h.vm.tableData[0].contentDataDetails[0].extraSetting, '保留')
  h.vm.tableData = [{ id: 1, sortNum: 5 }, { id: 2, sortNum: 8 }, { id: 3, sortNum: 10 }]
  h.vm.allTableData = [...h.vm.tableData]
  await h.vm.persistRowOrder({ oldIndex: 2, newIndex: 0 })
  const order = h.calls.filter(item => item.url === 'business/datadetails/update').map(item => ({ id: item.data.id, sortNum: item.data.sortNum }))
  assert.deepEqual(order, [{ id: 3, sortNum: 5 }, { id: 1, sortNum: 8 }, { id: 2, sortNum: 10 }])
})

const triggerRow = (id, equipmentId, alarmType, extra = {}) => ({
  id, equipmentId, apiKey: '设备数据池', siteId: 9, labelId: 'target-voltage', labelName: 'TARGET-电压',
  code: `TARGET-规则${id}`, cycle: 0, triggerType: 2,
  logics: [{ id: 101, triggerId: id, logic: '>', triggerValue: 5 }],
  strMatchs: [{ id: 102, triggerId: id, matchStr: '高电压', replaceCodeFlag: true, replaceCode: '高电压代码' }],
  alarmUser: { id: 103, triggerId: id, alarmType, alarmStage: 1, alarmChannel: '0', alarmUsers: '6,7' },
  statisticConfig: { id: 104, triggerId: id, useStatisticValue: true, statisticType: '1', statisticValue: 2, statisticValueType: 0 },
  ...extra
})
const triggerRows = [
  triggerRow(1, 42, '0'), triggerRow(2, 42, '1'), triggerRow(3, '42', '0, 1', { triggerType: 1 }),
  triggerRow(4, 42, '10'), triggerRow(5, 99, '0'), triggerRow(6, 42, null)
]
function triggerHarness(alarmTypeName = '告警', routes = {}, extraBindings = {}) {
  const api = apiHarness({
    'business/datadetails/find': ({ data }) => response([
      { id: 7, equipmentId: data.equipmentId, labelId: data.equipmentId === 42 ? 'target-voltage' : 'source-voltage',
        name: '电压', labelName: data.equipmentId === 42 ? 'TARGET-电压' : 'SOURCE-电压' }
    ]),
    'business/trigger/find': response(triggerRows),
    'business/trigger/add': { code: 200 },
    'business/trigger/update': { code: 200 },
    'business/trigger/delete': { code: 200 },
    'business/equipment/find': response([{ id: 42, equipmentName: '当前设备' }, { id: 84, equipmentName: '来源设备' }]),
    ...routes
  })
  const options = component('src/views/equipMana/components/TriggerTool.vue', {
    ...api.equipment, ...api.trigger, ...ccpcData, Message: api.Message,
    meConfirm: async () => true, getUserId: () => '9', ...extraBindings
  })
  const emitted = []
  const vm = instance(options, {
    formData: { id: 42, equipmentName: '当前设备', equipmentCode: 'TARGET', apiKey: '当前数据池', belongToSiteId: 9 },
    alarmTypeName, realTimeData: {},
    $refs: { dataForm: { validate: async () => true, clearValidate() {} } },
    $emit: (...args) => emitted.push(args)
  })
  return { ...api, vm, options, emitted }
}

await check('告警与预警只显示当前设备完整类型匹配规则，共用规则和纯统计配置正确保留', async () => {
  for (const [type, ids] of [['告警', [1, 3]], ['预警', [2, 3]]]) {
    const h = triggerHarness(type)
    await h.vm.init()
    assert.deepEqual(h.vm.tableData.map(row => row.id), ids)
    assert.equal(h.vm.total, 2)
    assert.equal(h.vm.labelsReady, true)
    assert.equal(h.vm.triggersReady, true)
    assert.equal(h.vm.matchAlarmType('10', '1'), false)
    assert.equal(h.vm.matchAlarmType('0, 1', '1'), true)
    assert.deepEqual(h.calls.find(item => item.url === 'business/trigger/find').data, { equipmentId: 42 })
    const foreign = triggerRows.find(row => row.equipmentId === 99)
    h.vm.modify(foreign)
    assert.equal(h.vm.showDrawer, false)
    await h.vm.remove(foreign)
    assert.equal(h.calls.some(item => item.url === 'business/trigger/delete'), false)
  }
})

await check('新告警使用登录账号收件人并提交当前设备，周期换算和修改旧关联保持正确', async () => {
  const h = triggerHarness()
  await h.vm.init()
  h.vm.add()
  assert.equal(h.vm.form.alarmUser.alarmUsers, '9')
  assert.equal(h.vm.form.alarmUser.alarmChannel, '0')
  h.vm.form.code = '  新规则  '
  h.vm.form.chufaType = 2
  h.vm.form.cycleType = 2
  h.vm.form.realtimeCycle = '3'
  assert.deepEqual(h.vm.form.logics, [], '新配置必须由用户明确添加条件，不默认创建告警')
  h.vm.addlogics()
  h.vm.form.logics[0].triggerValue = '7.5'
  await h.vm.setData()
  const created = h.calls.find(item => item.url === 'business/trigger/add').data
  assert.equal(created.equipmentId, 42)
  assert.equal(created.apiKey, '当前数据池')
  assert.equal(created.code, '新规则')
  assert.equal(created.cycle, 180)
  assert.equal(created.logics[0].triggerValue, 7.5)
  assert.equal(h.vm.showDrawer, false)
  h.vm.modify(h.vm.tableData[0])
  h.vm.form.logics[0].triggerValue = '8.5'
  await h.vm.setData()
  const updated = h.calls.find(item => item.url === 'business/trigger/update').data
  assert.equal(updated.id, 1)
  assert.equal(updated.alarmUser.id, 103)
  assert.equal(updated.logics[0].id, 101)
  assert.equal(updated.logics[0].triggerValue, 8.5)
  assert.equal(h.emitted.length, 2)
  assert.equal(h.vm.operationLoading, false)
})

await check('告警保存非200与网络失败不提示成功不关抽屉，设备切换后旧表单无法保存', async () => {
  for (const route of [{ code: 500, msg: '告警保存失败' }, () => { throw new Error('告警网络失败') }]) {
    const h = triggerHarness('告警', { 'business/trigger/update': route })
    await h.vm.init()
    h.vm.modify(h.vm.tableData[0])
    await h.vm.setData()
    assert.equal(h.vm.showDrawer, true)
    assert.equal(h.vm.operationLoading, false)
    assert.equal(h.notices.some(item => item.type === 'success'), false)
    assert.ok(h.notices.some(item => item.type === 'error'))
    assert.deepEqual(h.emitted, [])
  }
  const h = triggerHarness()
  await h.vm.init()
  h.vm.modify(h.vm.tableData[0])
  h.vm.formData = { ...h.vm.formData, id: 99 }
  await h.vm.setData()
  assert.equal(h.calls.some(item => item.url === 'business/trigger/update'), false)
})

await check('复制其他设备规则映射当前数据点和设备，不携带来源主键并保留实际收件人', async () => {
  const source = triggerRow(900, 84, '0,1', {
    code: 'SOURCE-高电压', labelId: 'source-voltage', labelName: 'SOURCE-电压', cycle: 3600
  })
  const componentPayloads = []
  const h = triggerHarness('告警', {
    'business/trigger/find': ({ data }) => response(data.equipmentId === 84 ? [source, triggerRow(901, 99, '0')] : triggerRows)
  })
  // 同时核对组件准备的配置和真实接口封装最终发送的数据。
  const options = component('src/views/equipMana/components/TriggerTool.vue', {
    ...h.equipment, ...h.trigger, ...ccpcData, Message: h.Message, meConfirm: async () => true, getUserId: () => '9',
    createtrigger: async payload => { componentPayloads.push(clone(payload)); return h.trigger.createtrigger(payload) }
  })
  h.vm.sureCopy = options.methods.sureCopy.bind(h.vm)
  await h.vm.init()
  await h.vm.copyOtherEquipAlarm()
  assert.equal(h.vm.copyEquipId, 84)
  assert.equal(h.vm.copyLabelsReady, true)
  await h.vm.sureCopy()
  assert.equal(componentPayloads.length, 1)
  const payload = componentPayloads[0]
  assert.equal('id' in payload, false)
  assert.equal('id' in payload.alarmUser, false)
  assert.equal('triggerId' in payload.alarmUser, false)
  assert.equal('id' in payload.logics[0], false)
  assert.equal('triggerId' in payload.logics[0], false)
  assert.equal('id' in payload.strMatchs[0], false)
  assert.equal(payload.code, 'TARGET-高电压')
  assert.equal(payload.labelId, 'target-voltage')
  assert.equal(payload.alarmUser.alarmType, '0')
  assert.equal(payload.alarmUser.alarmUsers, '6,7')
  const sent = h.calls.find(item => item.url === 'business/trigger/add').data
  assert.equal(sent.equipmentId, 42)
  assert.equal(sent.apiKey, '当前数据池')
  assert.equal(sent.cycle, 3600)
  for (const part of [sent, sent.alarmUser, sent.statisticConfig, ...sent.logics, ...sent.strMatchs]) {
    assert.equal('id' in part, false)
    assert.equal('triggerId' in part, false)
  }
  assert.equal(h.vm.copyOtherEquipAlarmDialog, false)
  assert.deepEqual(h.emitted, [['data-change']])
  assert.deepEqual(source.alarmUser.alarmType, '0,1', '复制不得修改来源设备配置')
})

await check('复制失败保持选择弹窗，数据查询失败禁止新增或复制配置', async () => {
  const source = triggerRow(900, 84, '0', { code: 'SOURCE-高电压', labelId: 'source-voltage' })
  const h = triggerHarness('告警', {
    'business/trigger/find': ({ data }) => response(data.equipmentId === 84 ? [source] : triggerRows),
    'business/trigger/add': { code: 500, msg: '复制失败' }
  })
  await h.vm.init()
  await h.vm.copyOtherEquipAlarm()
  await h.vm.sureCopy()
  assert.equal(h.vm.copyOtherEquipAlarmDialog, true)
  assert.equal(h.vm.operationLoading, false)
  assert.equal(h.notices.some(item => item.type === 'success'), false)
  assert.deepEqual(h.emitted, [])
  const failed = triggerHarness('告警', { 'business/datadetails/find': { code: 200, return: null } })
  await failed.vm.init()
  assert.equal(failed.vm.labelsReady, false)
  failed.vm.add()
  assert.equal(failed.vm.showDrawer, false)
  assert.equal(failed.notices.some(item => item.type === 'success'), false)
})

console.log(`设备管理离线回归：${checks} 项通过。`)
