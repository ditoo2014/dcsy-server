import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'

// 执行真实 Vue 组件的方法，仅替换 HTTP、提示框与定时器；不连接后端或数据库。
// 运行：node scripts/check-business-history.mjs
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const timerJobs = new Map()
let timerID = 0
function evaluate(source, dependencies = {}) {
  const module = { exports: {} }
  const code = transformSync(source, { loader: 'js', format: 'cjs', define: { 'import.meta.env.DEV': 'false' } }).code
  const require = name => {
    if (Object.hasOwn(dependencies, name)) return dependencies[name]
    if (name.endsWith('.vue') || name === '@/components/Pagination') return { default: {} }
    if (/\.(png|svg)$/.test(name)) return name
    throw new Error(`测试依赖未声明：${name}`)
  }
  const schedule = callback => { timerJobs.set(++timerID, callback); return timerID }
  new Function('module', 'exports', 'require', 'localStorage', 'setTimeout', 'clearTimeout', 'setInterval', 'clearInterval', 'console', code)(
    module, module.exports, require, { getItem: () => null }, schedule, id => timerJobs.delete(id), schedule, id => timerJobs.delete(id), { error() {}, log() {}, warn() {} }
  )
  return module.exports
}
const chineseHelper = evaluate(fs.readFileSync(path.join(root, 'src/utils/ChineseHelperStr.js'), 'utf8'))
const utils = evaluate(fs.readFileSync(path.join(root, 'src/utils/utils.js'), 'utf8'), { './ChineseHelperStr.js': chineseHelper })
const batteryPoints = evaluate(fs.readFileSync(path.join(root, 'src/utils/battery-points.js'), 'utf8'))
const safeJSON = evaluate(fs.readFileSync(path.join(root, 'src/utils/safe-json.js'), 'utf8'))
function component(relativePath, api = {}) {
  const file = path.join(root, relativePath)
  const { descriptor, errors } = parse(fs.readFileSync(file, 'utf8'), { filename: file })
  assert.deepEqual(errors, [])
  assert.deepEqual(compileTemplate({ source: descriptor.template.content, filename: file, id: 'business-history-check' }).errors, [])
  const script = compileScript(descriptor, { id: 'business-history-check' })
  return evaluate(script.content, {
    '@/api/moniList.js': api,
    '@/api/trigger.js': api,
    '@/api/siteMana.js': { findSiteListExactMatch: async () => ({ code: 200, return: { data: [] } }) },
    '@/api/equipMana.js': { findEquipmentListExactMatch: async () => ({ code: 200, return: { data: [] } }) },
    '@/utils/utils.js': utils,
    '@/utils/battery-points.js': batteryPoints,
    '@/utils/safe-json.js': safeJSON
  }).default
}
function instance(options, extra = {}) {
  const messages = []
  const events = []
  const state = { ...options.data(), ...extra, $message: { error: message => messages.push(message) }, $emit: (...event) => events.push(event) }
  for (const [name, method] of Object.entries(options.methods)) state[name] = method.bind(state)
  return { state, messages, events }
}
function deferred() {
  let resolve
  const promise = new Promise(callback => { resolve = callback })
  return { promise, resolve }
}
const historyPath = 'src/views/lldcsy/dcjkgl/comp/HistrData.vue'
const sampleTime = '2026-09-21 12:00:00 +0800 CST'
const sampleSeconds = utils.dateConvStamp(utils.format0800CSTDateString(sampleTime)) / 1000
const historyOK = (value = 3.332) => ({ code: 200, return: { times: [sampleTime], values: [value] } })
let checks = 0
async function check(name, callback) {
  await callback()
  checks++
  console.log(`通过：${name}`)
}
function historyState(getdata = async () => historyOK()) {
  const options = component(historyPath, { getdata })
  const harness = instance(options, { formData: { id: 1, apiKey: 'test' }, realDataList: [] })
  Object.assign(harness.state, {
    dczcsSelectList: [{ name: '组电压', checked: true, labelDetails: { labelId: 'voltage', title: '组电压' } }],
    dtdySelectList: [],
    startTime: sampleSeconds - 600,
    endTime: sampleSeconds + 600,
    pointChartData: { xAxis: { data: [{ value: '2026-09-21 12:00:00', timeStmap: sampleSeconds }] }, series: [] }
  })
  return { options, ...harness }
}

await check('秒、分钟、小时、天统一换算为实际秒数，拒绝无效间隔', () => {
  const { state } = historyState()
  for (const [unit, seconds] of Object.entries({ second: 5, minute: 300, hour: 18000, day: 432000 })) {
    state.timeType = unit
    state.timeLength = 5
    assert.equal(state.getHistoryIntervalSeconds(), seconds)
  }
  for (const invalid of [0, -1, '', 'abc']) {
    state.timeLength = invalid
    assert.throws(() => state.getHistoryIntervalSeconds(), /查询间隔/)
  }
})

await check('有效历史数值缺状态时仍绘图，导出包含真实值和零值', async () => {
  for (const value of [3.332, 0]) {
    const { state, messages } = historyState(async () => historyOK(value))
    await state.setChartData('电池状态')
    assert.equal(state.pointChartData.series[0].data[0].value, value)
    assert.equal(state.pointChartData.series[0].data[0].itemStyle.color, 'rgba(122,247,190)')
    assert.equal(state.formatJson(state.pointChartData)[0][1], value)
    assert.deepEqual(messages, [])
  }
})

await check('状态记录、基准横轴和电压曲线使用同一换算间隔', async () => {
  const requests = []
  const { state } = historyState(async params => { requests.push(params); return historyOK(params.labelID === 'state' ? 1 : 3.332) })
  state.timeType = 'minute'
  state.timeLength = 5
  state.dcztMsg = { labelId: 'state' }
  await state.setChartData()
  assert.ok(requests.length >= 3)
  assert.ok(requests.every(item => item.timeType === 'second' && item.timeLength === 300))
  assert.equal(new Set(requests.map(item => item.Pagesize)).size, 1)
  assert.ok(requests.some(item => item.labelID === 'state'))
  assert.equal(state.pointChartData.series[0].data[0].value, 3.332)
})

await check('有状态时保留原版充放电颜色，没有组电压时仍能绘制选中的单体测点', async () => {
  const { state } = historyState()
  state.chartBatteryStateIntervals = [{ code: 1, startTimeStmap: sampleSeconds - 1, endTimeStmap: sampleSeconds + 1 }]
  await state.setChartData('电池状态')
  assert.equal(state.pointChartData.series[0].data[0].itemStyle.color, state.getBatteryStateColor(1))
  const fallback = historyState().state
  fallback.dtdySelectList = [{ name: '1节', checked: true, labelDetails: { labelId: 'cell-1', title: '1节' } }]
  fallback.dczcsSelectList = []
  await fallback.setChartData()
  assert.equal(fallback.pointChartData.series[0].name, '1节')
  assert.equal(fallback.pointChartData.series[0].data[0].value, 3.332)
})

await check('快速充放电切片保留原版一秒识别，不被界面小时单位放大', async () => {
  const requests = []
  const { state } = historyState(async params => { requests.push(params); return historyOK(1) })
  state.timeType = 'hour'
  state.dcztMsg = { labelId: 'state' }
  state.fdStartTime = state.startTime
  state.fdEndTime = state.endTime
  await state.getDcztAndJzMsg('放电切片快速选择', state.beginHistoryRequest('discharge'))
  assert.ok(requests.length)
  assert.ok(requests.every(item => item.timeType === 'second' && item.timeLength === 1))
})

await check('接口业务错误、数组错位、网络错误不会伪装成成功空数据', async () => {
  const { state } = historyState(async () => ({ code: 400, msg: '历史存储不可用' }))
  await assert.rejects(() => state.queryHistory({}), /历史存储不可用/)
  const malformed = historyState(async () => ({ code: 200, return: { times: [sampleTime], values: [] } })).state
  await assert.rejects(() => malformed.queryHistory({}), /格式异常/)
  const failed = historyState(async () => { throw new Error('网络连接失败') })
  await failed.state.setChartData('电池状态')
  assert.deepEqual(failed.messages, ['网络连接失败'])
  assert.equal(failed.state.curveSliceLoading, false)
})

await check('无历史数据成功返回空数组，不产生错误提示', async () => {
  const { state, messages } = historyState(async () => ({ code: 200, return: { times: [], values: [] } }))
  await state.setChartData('电池状态')
  assert.ok(state.pointChartData.series[0].data.every(item => item == null || item.value == null))
  assert.deepEqual(messages, [])
})

await check('较慢旧请求与已卸载组件均不能覆盖当前曲线', async () => {
  const first = deferred()
  let calls = 0
  const { state, options } = historyState(() => ++calls === 1 ? first.promise : Promise.resolve(historyOK(4.2)))
  const oldRequest = state.setChartData('电池状态')
  const newRequest = state.setChartData('电池状态')
  await newRequest
  first.resolve(historyOK(1.1))
  await oldRequest
  assert.equal(state.pointChartData.series[0].data[0].value, 4.2)
  const token = state.beginHistoryRequest()
  options.beforeUnmount.call(state)
  assert.equal(state.isCurrentHistoryRequest(token), false)
})

await check('修改查询间隔触发重查，连续输入只执行最后一次', async () => {
  const { state, options } = historyState()
  let refreshes = 0
  state.timeChange = async () => { refreshes++ }
  assert.equal(options.watch.timeLength, 'scheduleCurveRefresh')
  assert.equal(options.watch.timeType, 'scheduleCurveRefresh')
  state.scheduleCurveRefresh()
  const oldTimer = state.intervalRefreshTimer
  state.scheduleCurveRefresh()
  assert.equal(timerJobs.has(oldTimer), false)
  await timerJobs.get(state.intervalRefreshTimer)()
  assert.equal(refreshes, 1)
  options.beforeUnmount.call(state)
})

for (const file of ['gjxxssgj.vue', 'yjxxssyj.vue']) {
  await check(`${file} 切换电池同步筛选、重置页码且丢弃旧响应`, async () => {
    const first = deferred()
    const requests = []
    const options = component(`src/views/lldcsy/dcjkgl/${file}`, { findAlarmRecord: params => {
      requests.push(params)
      return requests.length === 1 ? first.promise : Promise.resolve({ code: 200, return: { data: [{ equipmentId: 2, alarmStage: 1 }], total: 1 } })
    } })
    const { state } = instance(options, { equipId: 1 })
    state.syncEquipmentFilter()
    const oldRequest = state.getDataList()
    state.equipId = 2
    state.page = 5
    options.watch.equipId.call(state)
    await state.getDataList()
    first.resolve({ code: 200, return: { data: [{ equipmentId: 1, alarmStage: 2 }], total: 9 } })
    await oldRequest
    assert.equal(requests[1].equipmentId, 2)
    assert.equal(requests[1].pageNum, 1)
    assert.equal(state.tableData[0].equipmentId, 2)
    assert.equal(state.total, 1)
    state.resetData()
    assert.equal(requests.at(-1).equipmentId, 2)
    options.beforeUnmount.call(state)
    assert.equal(state.disposed, true)
  })
}

await check('实时数据页的批量错误、单点回退错误和错误提示保持一致', async () => {
  const options = component('src/views/lldcsy/dcjkgl/comp/RealTimeData.vue', { getdata: async () => ({ code: 400, msg: '存储不可用' }) })
  const { state, messages } = instance(options, { formData: { apiKey: 'test' }, realDataList: [] })
  const requestList = [{ item: { labelId: 'voltage', dataName: '组电压' }, startTime: sampleSeconds - 100, endTime: sampleSeconds }]
  await assert.rejects(() => state.getHistoryDataBatch(requestList), /存储不可用/)
  const fallback = await state.getHistoryDataFallback(requestList)
  assert.equal(fallback[0].error, '存储不可用')
  assert.throws(() => state.buildHistorySeries({ error: '无效数据' }), /无效数据/)
  assert.throws(() => state.buildHistorySeries({ times: [sampleTime], values: [], status: [] }), /格式异常/)
  state.historyLabelList = [requestList[0].item]
  await state.getHistoryData()
  assert.equal(messages.length, 1)
  assert.match(messages[0], /1 个测点的历史数据查询失败/)
})

await check('关闭或切换电池后，实时数据页的旧历史响应不写回图表', async () => {
  const pending = deferred()
  const options = component('src/views/lldcsy/dcjkgl/comp/RealTimeData.vue', { getdata: () => pending.promise })
  const { state } = instance(options, { formData: { apiKey: 'test' }, realDataList: [] })
  state.historyLabelList = [{ labelId: 'voltage', dataName: '组电压' }]
  const initialKey = state.basicBardataChangeKey
  const loading = state.getHistoryData()
  options.beforeUnmount.call(state)
  pending.resolve({ code: 200, return: { data: [{ times: [sampleTime], values: [3.3], status: [1] }] } })
  await loading
  assert.equal(state.basicBardataChangeKey, initialKey)
})

console.log(`业务历史与告警回归完成：${checks} 项通过；未访问数据库或外部接口。`)
