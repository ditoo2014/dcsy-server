import assert from 'node:assert/strict'
import fs from 'node:fs'
import { fileURLToPath } from 'node:url'
import { parse, compileStyleAsync, compileTemplate } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'
import postcss from 'postcss'

// 离线检查真实组件编译结果，防止固定高度、容器断点和标签避让再次失效。
// 运行：node scripts/check-battery-detail-layout.mjs；不访问接口、数据库或网络。
const read = name => fs.readFileSync(new URL(`../${name}`, import.meta.url), 'utf8')
let checks = 0
function check(name, verify) {
  verify()
  checks++
  console.log(`通过：${name}`)
}

async function component(name) {
  const filename = fileURLToPath(new URL(`../${name}`, import.meta.url))
  const { descriptor, errors } = parse(read(name), { filename })
  assert.deepEqual(errors, [])
  assert.deepEqual(compileTemplate({
    source: descriptor.template.content, filename, id: 'battery-layout-check'
  }).errors, [])
  const styles = await Promise.all(descriptor.styles.map(style => compileStyleAsync({
    source: style.content, filename, id: 'data-v-battery-layout-check',
    scoped: style.scoped, preprocessLang: style.lang
  })))
  styles.forEach(style => assert.deepEqual(style.errors, []))
  return { descriptor, css: postcss.parse(styles.map(style => style.code).join('\n')) }
}

function selectors(rule) {
  return rule.selector.split(',').map(selector => selector.replace(/\[data-v-[^\]]+\]/g, '').trim())
}
function declarations(css, selector) {
  const result = {}
  css.walkRules(rule => {
    if (rule.parent.type !== 'root' || !selectors(rule).includes(selector)) return
    rule.nodes.filter(node => node.type === 'decl').forEach(node => { result[node.prop] = node.value })
  })
  return result
}
function containerRule(css, name, selector, condition = '') {
  let result = null
  css.walkAtRules('container', container => {
    if (!container.params.startsWith(`${name} (`)) return
    if (condition && container.params !== `${name} (${condition})`) return
    container.walkRules(rule => {
      if (selectors(rule).includes(selector)) result = rule
    })
  })
  return result
}

const realtime = await component('src/views/lldcsy/dcjkgl/comp/RealTimeData.vue')
const history = await component('src/views/lldcsy/dcjkgl/comp/HistrData.vue')
const header = await component('src/views/lldcsy/dcjkgl/dcDetail.vue')

check('实时信息卡没有固定高度，字段换行后撑高整行和卡片', () => {
  const card = declarations(realtime.css, '.sssjCss .dcDetailsMsgContentCss .dcDetailsMsgCss')
  assert.equal(card.height, undefined)
  assert.equal(card['min-height'], '11vw')
  const field = declarations(realtime.css, '.sssjCss .dcDetailsMsgContentCss .dcDetailsMsgCss .dcDetialsGzMsgCss .dcGzMsgContentCss')
  assert.equal(field.display, 'flex')
  assert.equal(field['flex-wrap'], 'wrap')
  assert.equal(field.height, undefined)
  assert.equal(field['line-height'], 'max(20px, 1.6vw)')
})

check('实时图表最小260像素，外框由图表内容撑高', () => {
  assert.equal(declarations(realtime.css, '.sssjCss .chartContentCss').height, undefined)
  assert.equal(declarations(realtime.css, '.sssjCss .chartContentCss .chartCss .multipleYAxesCss')['min-height'], '260px')
})

check('温度和电压卡片保留桌面比例，CSS尺寸下限不被Stylus提前求值', () => {
  assert.equal(declarations(realtime.css, '.sssjCss .dcDetailsMsgContentCss .mbxxDetailsMsgCss').height, 'max(190px, 13.75vw)')
  assert.equal(declarations(realtime.css, '.sssjCss .dcDetailsMsgContentCss .mbxxDetailsMsgCss .mbxxContentCss').height,
    'calc(max(190px, 13.75vw) - max(24px, 2vw) - 0.3vw)')
})

check('实时窄容器断点使用与基础规则相同的选择器，不被优先级覆盖', () => {
  const selector = '.sssjCss .realTimeLayoutCss'
  assert.equal(declarations(realtime.css, selector)['grid-template-columns'], 'minmax(0, 70fr) minmax(320px, 29fr)')
  const rule = containerRule(realtime.css, 'battery-realtime', selector)
  assert.ok(rule)
  assert.equal(rule.nodes.find(node => node.prop === 'grid-template-columns').value, 'minmax(0, 1fr)')
  assert.match(realtime.css.toString(), /@supports not \(container-type: inline-size\)/)
})

check('两组柱状标签启用碰撞隐藏，完整数据和悬停提示保留', () => {
  const module = { exports: {}}
  const code = transformSync(realtime.descriptor.script.content, { format: 'cjs', loader: 'js' }).code
  // 仅执行 data() 配置；接口、图片及业务工具不执行。
  new Function('require', 'module', 'exports', code)(() => ({}), module, module.exports)
  const bar = module.exports.default.data().basicBarData3
  assert.equal(bar.tooltip.trigger, 'axis')
  assert.equal(bar.series.length, 2)
  bar.series.forEach(series => {
    assert.equal(series.label.show, true)
    assert.equal(series.labelLayout.hideOverlap, true)
    assert.equal(series.label.formatter, undefined)
  })
  const echarts = read('src/utils/echarts.js')
  assert.match(echarts, /import \{ LabelLayout \} from 'echarts\/features'/)
  assert.match(echarts, /echarts\.use\([\s\S]*\bLabelLayout\b/)
})

check('历史长时间记录不固定行高，切片外框按内容增长', () => {
  const slices = declarations(history.css, '.hisDataCss .sjxzContentCss')
  assert.equal(slices.height, undefined)
  assert.equal(slices.display, 'grid')
  assert.equal(slices.gap, 'max(8px, 0.5vw) 1%')
  const record = declarations(history.css, '.hisDataCss .sjxzContentCss .sjxzCss .ksxzLsjlContentCss .fdjlContentCss .fdjlCss .fdjlTabCss .fdjlListCss .fdjlItemCss')
  assert.equal(record.height, undefined)
  assert.equal(record['overflow-wrap'], 'anywhere')
  assert.equal(record['min-height'], 'max(24px, 1.2vw)')
})

check('历史图表高度和筛选换行保护保持生效', () => {
  assert.equal(declarations(history.css, '.hisDataCss .bhqxContentCss .multipleYAxesContentCss')['min-height'], '260px')
  assert.equal(declarations(history.css, '.hisDataCss .bhqxContentCss .bhqxHeadContentCss')['flex-wrap'], 'wrap')
  assert.ok(containerRule(history.css, 'battery-history', '.hisDataCss .sjxzContentCss'))
  assert.ok(containerRule(history.css, 'battery-history-slice', '.hisDataCss .sjxzContentCss .sjxzCss .ksxzLsjlContentCss'))
})

check('顶部和标签栏恢复原比例排列，保留地址浮动隔离及完整提示', () => {
  const banner = declarations(header.css, '.dcDetailCss .topBannerCss')
  assert.equal(banner.height, '4vw')
  assert.equal(banner['min-width'], undefined)
  const content = declarations(header.css, '.dcDetailCss .topBannerCss .zdContentCss')
  assert.equal(content.float, 'left')
  assert.equal(content.width, '44%')
  assert.equal(content['text-align'], 'right')
  const station = declarations(header.css, '.dcDetailCss .topBannerCss .zdContentCss .zdDetailsCss')
  assert.equal(station.display, 'flow-root')
  assert.equal(station['min-height'], '2.2vw')
  assert.equal(station.height, undefined)
  assert.equal(station['font-size'], '1vw')
  const address = declarations(header.css, '.dcDetailCss .topBannerCss .zdContentCss .zdAddrCss')
  assert.equal(address['white-space'], 'nowrap')
  assert.equal(address['text-overflow'], 'ellipsis')
  assert.match(header.descriptor.template.content, /class="zdAddrCss" :title="formData\.addr/)
  assert.equal(declarations(header.css, '.dcDetailCss .topBannerCss .zdBasicCss .dqdczCss .dqdczSelCss').width, '10vw')
  assert.equal(declarations(header.css, '.dcDetailCss .dcHeaderCss').height, '4vw')
  assert.equal(declarations(header.css, '.dcDetailCss .dcHeaderCss .tabContentCss').position, 'absolute')
  assert.doesNotMatch(header.descriptor.template.content, /topBannerViewportCss|topBannerScrollCss|zdParametersCss/)
  // 恢复早上布局，不保留后续顶部和标签栏的断点重排。
  header.css.walkRules(rule => {
    if (rule.parent.type === 'root') return
    assert.ok(!/\.topBanner|\.dcHeaderCss/.test(rule.selector), '顶部和标签栏不应被断点重新排版')
  })
})

console.log(`电池详情适配回归：${checks} 项通过。`)
