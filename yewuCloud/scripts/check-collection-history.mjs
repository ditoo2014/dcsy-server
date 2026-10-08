import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { compileScript, parse } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'
import * as Vue from 'vue'

// 从前端目录运行：node scripts/check-collection-history.mjs
// 编译并执行真实组件和接口文件，仅替换 HTTP 与画布，不访问数据库或网络。
// 这里不复制生产解析、分页、异常处理逻辑，避免测试与实现一起隐藏同一问题。
const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const componentPath = path.join(projectRoot, 'src/collection/components/gateway/PointHistoryChart.vue')
const apiPath = path.join(projectRoot, 'src/collection/api/gateway-realtime.js')
const { descriptor, errors } = parse(fs.readFileSync(componentPath, 'utf8'), { filename: componentPath })
assert.deepEqual(errors, [], '历史组件解析失败')
const compiledComponent = compileScript(descriptor, { id: 'history-regression' }).content

function evaluateModule(source, filename, dependencies, windowStub = {}) {
  const compiled = transformSync(source, {
    format: 'cjs',
    loader: 'js',
    sourcefile: filename,
    define: { 'import.meta.env': '{}' }
  }).code
  const module = { exports: {} }
  const localRequire = name => {
    assert.ok(Object.hasOwn(dependencies, name), `检查壳尚未声明依赖：${name}`)
    return dependencies[name]
  }
  new Function('require', 'module', 'exports', 'window', compiled)(localRequire, module, module.exports, windowStub)
  return module.exports
}

const authHeadersPath = path.join(projectRoot, 'src/collection/utils/request.js')
const realAuthHeaders = evaluateModule(fs.readFileSync(authHeadersPath, 'utf8'), authHeadersPath, {})
const ok = value => ({ data: { code: 200, msg: '查询成功', return: value } })
const countResult = total => ok({ count: total })
const historyResult = (values = [3.222, 3.111], extra = {}) => ok({
  times: values.map((_, index) => 1_789_999_000 - index),
  values,
  total: values.length,
  status: values.map((_, index) => index === 0 ? '1' : '-1'),
  ...extra
})
function deferred() {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}
// Vue 的侦听器及接口 await 使用微任务；不使用真实定时器或阻塞等待。
async function settle() {
  for (let i = 0; i < 30; i += 1) await Vue.nextTick()
}

function createHarness(initialProps = {}) {
  const calls = []
  const chartOptions = []
  const mounted = []
  const beforeUnmount = []
  const intervals = new Map()
  const listeners = new Map()
  let nextInterval = 1
  let disposals = 0
  let initializations = 0
  let unmounted = false
  const harness = {
    calls,
    chartOptions,
    reply: request => request.query.Function === 'countdata' ? countResult(2) : historyResult()
  }
  const client = {
    async post(url, body, options) {
      const request = { url, body, query: options.params, headers: options.headers }
      calls.push(request)
      return harness.reply(request)
    }
  }
  const api = evaluateModule(fs.readFileSync(apiPath, 'utf8'), apiPath, {
    axios: { create: () => client },
    '../config/gateway-detail': {},
    '../utils/gatewayTimesequenceResponse': {},
    '../utils/request': realAuthHeaders,
    // 身份失效跳转另有独立实现，本检查只覆盖历史业务状态与组件行为。
    './authFailureRedirect': { attachAuthFailureRedirectInterceptor() {} }
  })
  const chartStub = {
    use() {},
    init() {
      initializations += 1
      return {
        setOption: option => chartOptions.push(option),
        resize() {},
        dispose() { disposals += 1 }
      }
    },
    graphic: { LinearGradient: class {} }
  }
  const isolatedVue = {
    ...Vue,
    onMounted: callback => mounted.push(callback),
    onBeforeUnmount: callback => beforeUnmount.push(callback)
  }
  const component = evaluateModule(compiledComponent, componentPath, {
    vue: isolatedVue,
    'echarts-collection/core': chartStub,
    'echarts-collection/charts': {},
    'echarts-collection/components': {},
    'echarts-collection/renderers': {},
    '../../api/gateway-realtime': api
  }, {
    setInterval(callback) {
      const id = nextInterval++
      intervals.set(id, callback)
      return id
    },
    clearInterval: id => intervals.delete(id),
    addEventListener: (name, callback) => listeners.set(name, callback),
    removeEventListener: name => listeners.delete(name)
  }).default
  const props = Vue.reactive({
    title: '测试电压',
    gatewayId: '865619085125863',
    token: '仅用于回归检查的测试令牌',
    dataId: 'point-1',
    globalRefreshEnabled: false,
    darkMode: false,
    ...initialProps
  })
  const scope = Vue.effectScope()
  const state = scope.run(() => component.setup(props, { expose() {} }))
  state.chartRef.value = {}
  state.timeRange.value = [new Date(2026, 8, 19), new Date(2026, 8, 22, 23, 59, 59)]
  Object.assign(harness, {
    state,
    props,
    api,
    intervals,
    listeners,
    getDisposals: () => disposals,
    getInitializations: () => initializations,
    mount: async() => { for (const callback of mounted) await callback() },
    unmount() {
      if (unmounted) return
      unmounted = true
      for (const callback of beforeUnmount) callback()
      scope.stop()
      state.chartRef.value = null
    }
  })
  return harness
}

const suites = []
async function check(name, run, initialProps) {
  const harness = createHarness(initialProps)
  try {
    await run(harness)
    await settle()
    suites.push(name)
  } finally {
    harness.unmount()
  }
}
const latestOption = harness => harness.chartOptions.at(-1)
const lineValues = harness => latestOption(harness)?.series?.[0]?.data
function graphicTexts(harness) {
  const visit = value => {
    if (!value || typeof value !== 'object') return []
    return [typeof value.text === 'string' ? value.text : '', ...Object.values(value).flatMap(visit)]
  }
  return visit(latestOption(harness)?.graphic).filter(Boolean).join(' ')
}
function assertEmpty(harness, message) {
  assert.equal(harness.state.totalCount.value, 0, `${message}：总数没有清零`)
  assert.equal(harness.state.fullSeries.value.length, 0, `${message}：仍保留旧曲线`)
  assert.deepEqual(lineValues(harness), [], `${message}：图表仍保留旧点`)
  assert.deepEqual(latestOption(harness)?.series?.[0]?.markPoint?.data, [], `${message}：旧末点标记没有清除`)
  assert.equal(harness.state.loading.value, false, `${message}：加载状态没有结束`)
}

await check('初始状态与正常历史', async harness => {
  const { state, calls } = harness
  assert.equal(state.totalCount.value, 0, '未请求历史时不能显示模拟数量')
  assert.deepEqual(state.fullSeries.value, [], '未请求历史时不能显示模拟曲线')
  assert.equal(calls.length, 0, '组件初始化不应凭空获得历史数据')
  await state.resetToLatestPageAndFetch()
  assert.equal(state.totalCount.value, 2)
  assert.deepEqual(state.fullSeries.value.map(point => point.value), [3.111, 3.222], '历史应按时间从旧到新绘制')
  assert.deepEqual(state.fullSeries.value.map(point => point.status), ['-1', '1'], '异常状态不能丢失')
  assert.deepEqual(lineValues(harness), [3.111, 3.222])
  assert.equal(state.historyError.value, '')
  assert.equal(state.loading.value, false)
  assert.deepEqual(calls.map(call => call.query.Function), ['countdata', 'historydata'], '一次刷新不能重复请求计数或分页')
  const historyCall = calls[1]
  assert.equal(historyCall.url, '/tolink/timesequence/getdata')
  assert.equal(historyCall.body.dataId, 'point-1')
  assert.equal(historyCall.query.GatewayID, harness.props.gatewayId)
  assert.equal(historyCall.query.Pagenum, '1')
  assert.equal(historyCall.query.Pagesize, '30')
  assert.equal(historyCall.query.Datestart, String(Math.floor(state.timeRange.value[0].getTime() / 1000)))
  assert.equal(historyCall.query.Dateend, String(Math.floor(state.timeRange.value[1].getTime() / 1000)))
  assert.equal(historyCall.headers.Authorization, `Bearer ${harness.props.token}`)
  assert.equal(latestOption(harness).series[0].smooth, true, '保留原版曲线样式')
  assert.equal(latestOption(harness).dataZoom.length, 2, '保留原版滑块和图内缩放')
})

await check('空历史及旧数量清零', async harness => {
  await harness.state.resetToLatestPageAndFetch()
  harness.reply = request => request.query.Function === 'countdata' ? countResult(0) : ok(null)
  await harness.state.resetToLatestPageAndFetch()
  assertEmpty(harness, '查询返回 0 条')
  assert.equal(harness.state.historyError.value, '')
  assert.match(graphicTexts(harness), /暂无.*历史|暂无数据/, '空数据应在原图表区域明确提示')
})

await check('真实接口业务状态校验', async harness => {
  const params = {
    token: harness.props.token,
    gatewayId: harness.props.gatewayId,
    dataId: 'point-1',
    pageNo: 1,
    pageSize: 30,
    dateStart: 1,
    dateEnd: 2
  }
  harness.reply = () => ({ data: { code: 400, msg: '历史数据库暂不可用', return: null } })
  await assert.rejects(harness.api.fetchGatewayHistoryCount(params), /历史数据库暂不可用/)
  await assert.rejects(harness.api.fetchGatewayHistoryPaged(params), /历史数据库暂不可用/)
  harness.reply = () => ({ data: { code: 400, return: null } })
  await assert.rejects(harness.api.fetchGatewayHistoryPaged(params), /历史|查询|失败/)
})

await check('空切片及不合法返回格式', async harness => {
  harness.reply = request => request.query.Function === 'countdata'
    ? countResult(0)
    : ok({ total: 0, times: null, values: null })
  await harness.state.resetToLatestPageAndFetch()
  assertEmpty(harness, 'Go 空切片返回 null')
  assert.equal(harness.state.historyError.value, '')
  for (const body of [
    { total: 2, times: [1, 2], values: [3.3] },
    { total: 2, times: null, values: null },
    { total: 1, times: [1, 2], values: [3.3, 3.4] }
  ]) {
    harness.reply = request => request.query.Function === 'countdata' ? countResult(2) : ok(body)
    await harness.state.resetToLatestPageAndFetch()
    assertEmpty(harness, '返回明细格式或数量异常')
    assert.match(harness.state.historyError.value, /异常|不一致/)
  }
})

await check('以最新明细总数修正计数结果', async harness => {
  harness.reply = request => request.query.Function === 'countdata' ? countResult(12) : ok(null)
  await harness.state.resetToLatestPageAndFetch()
  assertEmpty(harness, '旧计数非零但真实明细已清空')
  harness.reply = request => request.query.Function === 'countdata'
    ? countResult(0)
    : historyResult(undefined, { total: 65 })
  await harness.state.resetToLatestPageAndFetch()
  assert.equal(harness.state.totalCount.value, 65)
  assert.equal(harness.state.historyPage.value, 3)
  assert.equal(harness.calls.at(-1).query.Pagenum, '1', '新增数据后仍应显示最新一页')
  assert.deepEqual(lineValues(harness), [3.111, 3.222])
})

await check('业务错误与网络异常清空旧曲线', async harness => {
  await harness.state.resetToLatestPageAndFetch()
  harness.reply = () => ({ data: { code: 400, msg: '历史数据库查询失败', return: null } })
  await harness.state.resetToLatestPageAndFetch()
  assertEmpty(harness, '数量查询业务错误')
  assert.match(harness.state.historyError.value, /历史数据库查询失败/)
  assert.match(graphicTexts(harness), /失败/, '业务错误应在原图表区域提示')
  harness.reply = request => request.query.Function === 'countdata'
    ? countResult(12)
    : ({ data: { code: 400, msg: '历史明细查询失败', return: null } })
  await harness.state.resetToLatestPageAndFetch()
  assertEmpty(harness, '明细查询业务错误')
  assert.match(harness.state.historyError.value, /历史明细查询失败/)
  harness.reply = () => { throw Object.assign(new Error('Network Error'), { isAxiosError: true }) }
  await harness.state.resetToLatestPageAndFetch()
  assertEmpty(harness, '网络请求失败')
  assert.match(harness.state.historyError.value, /历史|查询|网络|失败/, '网络异常必须有中文说明')
})

await check('原版页序与每页条数', async harness => {
  harness.reply = request => request.query.Function === 'countdata' ? countResult(65) : historyResult(undefined, { total: 65 })
  await harness.state.resetToLatestPageAndFetch()
  assert.equal(harness.state.totalCount.value, 65)
  assert.equal(harness.state.historyPage.value, 3, '展开后应停留在最后一页')
  assert.equal(harness.calls.at(-1).query.Pagenum, '1', 'UI 最后一页对应后端最新第一页')
  const beforePageChange = harness.calls.length
  harness.state.historyPage.value = 1
  await settle()
  assert.equal(harness.calls.length, beforePageChange + 2, '切页应只查询一组计数和明细')
  assert.equal(harness.calls.at(-1).query.Pagenum, '3', 'UI 第一页对应后端最旧一页')
  harness.state.pageSize.value = 60
  await settle()
  assert.equal(harness.state.historyPage.value, 2, '修改每页条数后应回到最新一页')
  assert.equal(harness.calls.at(-1).query.Pagenum, '1')
  assert.equal(harness.calls.at(-1).query.Pagesize, '60')
  harness.state.timeRange.value = [new Date(2026, 8, 19), new Date(2026, 8, 22)]
  harness.state.onTimeRangeChange()
  await settle()
  assert.equal(harness.state.timeRange.value[1].getHours(), 23, '保留原版结束日补全规则')
  assert.equal(harness.state.timeRange.value[1].getMinutes(), 59)
  assert.equal(harness.state.timeRange.value[1].getSeconds(), 59)
  assert.equal(harness.calls.at(-1).query.Pagenum, '1')
})

await check('旧明细晚返回不能覆盖新查询', async harness => {
  const oldPage = deferred()
  let pageCalls = 0
  harness.reply = request => {
    if (request.query.Function === 'countdata') return countResult(2)
    pageCalls += 1
    return pageCalls === 1 ? oldPage.promise : historyResult([9.222, 9.111])
  }
  const oldRequest = harness.state.resetToLatestPageAndFetch()
  await settle()
  assert.equal(pageCalls, 1)
  await harness.state.resetToLatestPageAndFetch()
  assert.deepEqual(lineValues(harness), [9.111, 9.222])
  const renderCount = harness.chartOptions.length
  oldPage.resolve(historyResult([1.222, 1.111]))
  await oldRequest
  assert.deepEqual(lineValues(harness), [9.111, 9.222], '旧请求覆盖了新曲线')
  assert.equal(harness.chartOptions.length, renderCount, '旧请求不应重绘图表')
  assert.equal(harness.state.totalCount.value, 2)
})

await check('旧计数晚返回及旧错误不干扰新查询', async harness => {
  const oldCount = deferred()
  let countCalls = 0
  harness.reply = request => {
    if (request.query.Function === 'countdata') {
      countCalls += 1
      return countCalls === 1 ? oldCount.promise : countResult(2)
    }
    return historyResult([8.222, 8.111])
  }
  const oldRequest = harness.state.resetToLatestPageAndFetch()
  await settle()
  await harness.state.resetToLatestPageAndFetch()
  const callsBeforeOldResponse = harness.calls.length
  oldCount.resolve(countResult(999))
  await oldRequest
  assert.equal(harness.calls.length, callsBeforeOldResponse, '过期计数返回后不应再发起分页请求')
  assert.equal(harness.state.totalCount.value, 2)
  assert.deepEqual(lineValues(harness), [8.111, 8.222])

  const oldFailure = deferred()
  let failureCountCalls = 0
  harness.reply = request => {
    if (request.query.Function === 'countdata') {
      failureCountCalls += 1
      return failureCountCalls === 1 ? oldFailure.promise : countResult(2)
    }
    return historyResult([7.222, 7.111])
  }
  const failedRequest = harness.state.resetToLatestPageAndFetch()
  await settle()
  await harness.state.resetToLatestPageAndFetch()
  oldFailure.reject(new Error('旧请求连接失败'))
  await failedRequest
  assert.equal(harness.state.historyError.value, '', '旧请求的失败不能覆盖新请求成功状态')
  assert.deepEqual(lineValues(harness), [7.111, 7.222])
})

await check('卸载后忽略结果且不重建图表', async harness => {
  await harness.state.resetToLatestPageAndFetch()
  const page = deferred()
  harness.reply = request => request.query.Function === 'countdata' ? countResult(8) : page.promise
  const pending = harness.state.resetToLatestPageAndFetch()
  await settle()
  harness.unmount()
  const snapshot = {
    total: harness.state.totalCount.value,
    values: harness.state.fullSeries.value.map(point => point.value),
    renders: harness.chartOptions.length,
    initializations: harness.getInitializations()
  }
  page.resolve(historyResult([6.222, 6.111]))
  await pending
  assert.equal(harness.getDisposals(), 1, '卸载应释放 ECharts 实例')
  assert.equal(harness.state.totalCount.value, snapshot.total, '卸载后不得写入计数')
  assert.deepEqual(harness.state.fullSeries.value.map(point => point.value), snapshot.values, '卸载后不得写入曲线')
  assert.equal(harness.chartOptions.length, snapshot.renders, '卸载后不得重绘')
  assert.equal(harness.getInitializations(), snapshot.initializations, '卸载后不得重建图表')
  assert.equal(harness.intervals.size, 0, '卸载后不得保留定时器')
  assert.equal(harness.listeners.size, 0, '卸载后不得保留窗口监听')
})

await check('挂载等待期间卸载不启动轮询', async harness => {
  const count = deferred()
  harness.props.globalRefreshEnabled = true
  await settle()
  // 上面的响应式切换可能触发一次刷新；此后只拦截挂载中的请求。
  harness.reply = () => count.promise
  const mounting = harness.mount()
  await settle()
  harness.unmount()
  count.resolve(countResult(0))
  await mounting
  assert.equal(harness.intervals.size, 0, '异步挂载返回后不能重新启动已卸载组件的定时器')
  assert.equal(harness.listeners.size, 0, '异步挂载返回后不能重新添加监听')
})

await check('正常挂载与原版自动刷新', async harness => {
  await harness.mount()
  assert.equal(harness.calls.length, 2, '正常挂载不能立即重复查询历史')
  assert.equal(harness.intervals.size, 1, '启用刷新时仅保留一个轮询定时器')
  assert.equal(harness.listeners.size, 1)
  const callback = [...harness.intervals.values()][0]
  callback()
  await settle()
  assert.equal(harness.calls.length, 4, '一次轮询应查询一组计数与明细')
  assert.equal(harness.calls.at(-1).query.Pagenum, '1')
  harness.state.autoRefresh.value = false
  await settle()
  assert.equal(harness.intervals.size, 0, '关闭自动刷新后应停止轮询')
}, { globalRefreshEnabled: true })

console.log(`采集云历史回归通过（${suites.length} 项）：${suites.join('、')}。`)
