import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'
import * as Vue from 'vue'

// 从项目根目录运行：node scripts/check-collection-render.mjs
// 使用真实 SFC 编译与 Vue 响应式渲染，检查迁移后的模板绑定，不访问后端。
const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const require = createRequire(import.meta.url)
const componentCache = new Map()
let apiCalls = 0
const chart = { setOption() {}, resize() {}, dispose() {} }
const echarts = { use() {}, init: () => chart, graphic: { LinearGradient: class {} }}
const isolatedVue = {
  ...Vue,
  // DOM、主题监听及定时器不属于本检查范围，避免在 Node 中启动浏览器副作用。
  onMounted() {},
  onUnmounted() {},
  onBeforeUnmount() {}
}

function evaluateModule(source, filename) {
  const result = transformSync(source, { format: 'cjs', loader: 'js', sourcefile: filename })
  const module = { exports: {}}
  const localRequire = specifier => {
    if (specifier === 'vue') return isolatedVue
    if (/^echarts(?:-collection)?\//.test(specifier)) return echarts
    if (specifier.endsWith('/api/gateway-realtime')) {
      const rejectRequest = () => {
        apiCalls += 1
        throw new Error('渲染回归检查禁止调用历史数据接口')
      }
      return { fetchGatewayHistoryCount: rejectRequest, fetchGatewayHistoryPaged: rejectRequest }
    }
    if (specifier.endsWith('.vue')) return loadComponent(path.resolve(path.dirname(filename), specifier))
    return require(specifier)
  }
  new Function('require', 'module', 'exports', result.code)(localRequire, module, module.exports)
  return module.exports
}

function loadComponent(filename) {
  if (componentCache.has(filename)) return componentCache.get(filename)
  const { descriptor, errors } = parse(fs.readFileSync(filename, 'utf8'), { filename })
  assert.deepEqual(errors, [], `${filename} 解析失败`)
  const id = path.basename(filename, '.vue')
  const script = compileScript(descriptor, { id, genDefaultAs: '__component' })
  const template = compileTemplate({
    source: descriptor.template.content,
    filename,
    id,
    compilerOptions: { bindingMetadata: script.bindings }
  })
  assert.deepEqual(template.errors, [], `${filename} 模板编译失败`)
  const source = `${script.content}\n${template.code}\n__component.render = render;\nexport default __component;`
  const component = evaluateModule(source, filename).default
  componentCache.set(filename, component)
  return component
}

// 用内存节点承接 Vue 渲染，保留真实事件和更新过程，无需安装 DOM 模拟依赖。
const node = (type, text = '') => ({ type, text, children: [], props: {}, parent: null })
function remove(child) {
  if (!child.parent) return
  const siblings = child.parent.children
  siblings.splice(siblings.indexOf(child), 1)
  child.parent = null
}
function insert(child, parent, anchor = null) {
  remove(child)
  const index = anchor ? parent.children.indexOf(anchor) : -1
  parent.children.splice(index < 0 ? parent.children.length : index, 0, child)
  child.parent = parent
}
const renderer = Vue.createRenderer({
  createElement: type => node(type),
  createText: text => node('#text', text),
  createComment: text => node('#comment', text),
  insert,
  remove,
  setText: (target, text) => { target.text = text },
  setElementText: (target, text) => { target.text = text; target.children = [] },
  patchProp: (target, key, _oldValue, value) => { target.props[key] = value },
  parentNode: target => target.parent,
  nextSibling: target => target.parent?.children[target.parent.children.indexOf(target) + 1] || null,
  setScopeId() {},
  insertStaticContent(content, parent, anchor) {
    const target = node('#static', content)
    insert(target, parent, anchor)
    return [target, target]
  }
})

const controls = new Map()
const tableRows = Symbol('tableRows')
function textOf(vnode) {
  if (typeof vnode === 'string') return vnode
  if (Array.isArray(vnode)) return vnode.map(textOf).join('')
  return typeof vnode?.children === 'string' ? vnode.children : textOf(vnode?.children || [])
}
function elementStub(name) {
  return Vue.defineComponent({
    name,
    inheritAttrs: false,
    setup(_props, { attrs, slots }) {
      const entry = { attrs, label: '' }
      if (!controls.has(name)) controls.set(name, new Set())
      controls.get(name).add(entry)
      Vue.onUnmounted(() => controls.get(name).delete(entry))
      if (name === 'ElTable') Vue.provide(tableRows, Vue.computed(() => attrs.data || []))
      const rows = name === 'ElTableColumn' ? Vue.inject(tableRows) : null
      return () => {
        // 主动渲染展开列的一条数据，检查历史组件是否在真实模板中正确绑定。
        const children = name === 'ElTableColumn'
          ? rows.value.slice(0, 1).flatMap((row, $index) => slots.default?.({ row, $index }) || [])
          : [...(slots.header?.() || []), ...(slots.default?.() || [])]
        entry.label = textOf(children).trim()
        return Vue.h('section', { 'data-component': name }, children)
      }
    }
  })
}

const panelPath = path.join(projectRoot, 'src/collection/components/gateway/RealtimePointListPanel.vue')
const Panel = loadComponent(panelPath)
const sampleRow = {
  pointKey: 'point-1',
  pointName: '测试电压',
  displayName: '测试电压',
  displayValue: '2.649',
  valueTime: '2026/09/21 09:00:00',
  orderIndex: 1,
  dataId: '1',
  status: '0'
}
const state = Vue.reactive({ rows: [sampleRow], gatewayId: '', token: '', refreshEnabled: false })
let refreshEvents = 0
const warnings = []
const runtimeErrors = []
const app = renderer.createApp({
  render: () => Vue.h(Panel, { ...state, onToggleRefresh: () => { refreshEvents += 1 } })
})
app.config.warnHandler = message => warnings.push(message)
app.config.errorHandler = error => runtimeErrors.push(error.stack || String(error))
for (const [filename, exportName] of [
  ['element-components.js', 'registerElementComponents'],
  ['element-table.js', 'registerElementTable']
]) {
  const fullPath = path.join(projectRoot, 'src/plugins', filename)
  evaluateModule(fs.readFileSync(fullPath, 'utf8'), fullPath)[exportName](app)
}
for (const name of ['ElCard', 'ElEmpty', 'ElAlert', 'ElDivider']) {
  assert.ok(app.component(name), `全局组件漏注册：${name}`)
}
// 先检查生产注册集，再替换渲染壳；不能通过注册测试专用组件掩盖生产遗漏。
for (const name of Object.keys(app._context.components)) {
  app._context.components[name] = elementStub(name)
}
app._context.directives.loading = {}

function findControl(name, predicate = () => true) {
  const result = [...(controls.get(name) || [])].find(predicate)
  assert.ok(result, `未渲染控件：${name}`)
  return result
}
function invoke(entry, event, value) {
  assert.equal(typeof entry.attrs[event], 'function', `模板事件未绑定：${event}`)
  return entry.attrs[event](value)
}
const renderedRows = () => findControl('ElTable').attrs.data
const button = label => findControl('ElButton', entry => entry.label === label)

const root = node('root')
const renderedText = target => target.text + target.children.map(renderedText).join('')
app.mount(root)
await Vue.nextTick()
assert.equal(renderedRows()?.length, 1, '传入一条点位后，实际表格没有收到数据')
assert.equal(renderedRows()[0].pointName, sampleRow.pointName)
assert.ok(findControl('ElCard').label.includes('网关数据列表'), '卡片标题插槽未渲染')
assert.match(renderedText(root), /测试电压\s+历史曲线/, '展开列未渲染真实历史组件')
const historySwitch = findControl('ElSwitch')
assert.equal(historySwitch.attrs.disabled, true, '历史组件未读取全局刷新状态')
const datePicker = findControl('ElDatePicker')
assert.ok(Array.isArray(datePicker.attrs.modelValue), '历史时间范围未绑定')
invoke(datePicker, 'onUpdate:modelValue', [new Date(2026, 8, 20), new Date(2026, 8, 21)])
invoke(datePicker, 'onChange')
await Vue.nextTick()
assert.equal(datePicker.attrs.modelValue[1].getHours(), 23, '历史时间查询未补全结束日期')
assert.equal(datePicker.attrs.modelValue[1].getMinutes(), 59)
assert.equal(datePicker.attrs.modelValue[1].getSeconds(), 59)
const historyPager = findControl('ElPagination', entry => entry.attrs['page-sizes']?.includes(30))
assert.equal(historyPager.attrs.total, 0, '未查询真实历史时应显示 0 条，不能使用模拟历史点')
invoke(historyPager, 'onUpdate:pageSize', 60)
await Vue.nextTick()
assert.equal(historyPager.attrs['page-size'], 60, '历史每页条数未绑定')

const input = findControl('ElInput')
invoke(input, 'onUpdate:modelValue', '不存在的数据点')
invoke(button('查询'), 'onClick')
await Vue.nextTick()
assert.equal(renderedRows().length, 0, '查询按钮未执行筛选')
invoke(button('重置'), 'onClick')
await Vue.nextTick()
assert.equal(renderedRows().length, 1, '重置按钮未恢复数据')
assert.equal(input.attrs.modelValue, '', '重置按钮未清空查询条件')
invoke(button('开启刷新'), 'onClick')
assert.equal(refreshEvents, 1, '刷新按钮未向父组件发送事件')

state.rows = Array.from({ length: 25 }, (_, index) => ({
  ...sampleRow, pointKey: `point-${index + 1}`, pointName: `测试点位${index + 1}`, orderIndex: index + 1
}))
await Vue.nextTick()
assert.equal(renderedRows().length, 20, '默认分页未限制为 20 条')
const pager = findControl('ElPagination', entry => entry.attrs['page-sizes']?.includes(10))
invoke(pager, 'onUpdate:currentPage', 2)
await Vue.nextTick()
assert.equal(renderedRows().length, 5, '翻页未更新实际表格数据')
assert.equal(renderedRows()[0].orderIndex, 21)
invoke(pager, 'onSizeChange', 10)
await Vue.nextTick()
assert.equal(renderedRows().length, 10, '修改每页条数未更新数据')
assert.equal(pager.attrs['current-page'], 1, '修改每页条数后未返回第一页')

app.unmount()
assert.equal(apiCalls, 0, '回归检查意外调用后端接口')
assert.deepEqual(runtimeErrors, [], '渲染过程中出现运行时错误')
assert.deepEqual(warnings, [], '渲染过程中出现 Vue 警告')
console.log('采集云渲染回归通过：组件注册、卡片标题、点位显示、查询/重置/刷新、分页和历史模板绑定均正常。')
