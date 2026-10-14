import assert from 'node:assert/strict'
import fs from 'node:fs'
import { fileURLToPath } from 'node:url'
import { parse, compileStyleAsync } from '@vue/compiler-sfc'
import postcss from 'postcss'

// 离线检查真实组件样式和共用边界，不访问接口；浏览器尺寸矩阵另行验证。
const read = name => fs.readFileSync(new URL(`../${name}`, import.meta.url), 'utf8')
let checks = 0

function check(name, verify) {
  verify()
  checks++
  console.log(`通过：${name}`)
}

async function componentCss(name) {
  const filename = fileURLToPath(new URL(`../${name}`, import.meta.url))
  const { descriptor, errors } = parse(read(name), { filename })
  assert.deepEqual(errors, [])
  const results = await Promise.all(descriptor.styles.map(style => compileStyleAsync({
    source: style.content,
    filename,
    id: 'data-v-responsive-test',
    scoped: style.scoped,
    preprocessLang: style.lang
  })))
  results.forEach(result => assert.deepEqual(result.errors, []))
  return postcss.parse(results.map(result => result.code).join('\n'))
}

function declarations(css, selector) {
  const result = {}
  css.walkRules(rule => {
    if (!rule.selector.includes(selector)) return
    rule.nodes.filter(node => node.type === 'decl').forEach(node => {
      result[node.prop] = node.value
    })
  })
  return result
}

function baseDeclarations(css, selector) {
  const result = {}
  css.walkRules(rule => {
    // 窄容器会覆写布局；基础桌面断言只读取不在 @container/@media 中的规则。
    if (!rule.selector.includes(selector) || rule.parent.type !== 'root') return
    rule.nodes.filter(node => node.type === 'decl').forEach(node => {
      result[node.prop] = node.value
    })
  })
  return result
}

const detailSource = read('src/views/lldcsy/dcjkgl/dcDetail.vue')
const detailCss = await componentCss('src/views/lldcsy/dcjkgl/dcDetail.vue')
const responsiveCss = postcss.parse(read('src/styles/responsive.css'))
const passwordCss = await componentCss('src/views/personMana/modifyPas.vue')

check('电池头部隔离浮动并随实际文字高度增长，避免小数像素侵入地址行', () => {
  const style = baseDeclarations(detailCss, '.zdDetailsCss')
  assert.equal(style.display, 'flow-root')
  assert.equal(style['min-height'], '2.2vw')
  assert.equal(style.height, undefined)
})

check('长地址保持单行省略，并保留完整地址的悬停提示', () => {
  const style = declarations(detailCss, '.zdAddrCss')
  assert.equal(style['white-space'], 'nowrap')
  assert.equal(style.overflow, 'hidden')
  assert.equal(style['text-overflow'], 'ellipsis')
  assert.match(detailSource, /class="zdAddrCss" :title="formData\.addr/)
})

check('统一适配样式在入口加载，不随业务或采集主题切换停用', () => {
  assert.match(read('src/main.js'), /import '\.\/styles\/responsive\.css'/)
  assert.doesNotMatch(read('src/styles/pageStyles.js'), /responsive\.css/)
})

check('弹窗限制可见宽高，正文可滚动、标题及提交区域不压缩', () => {
  // 选择精确根规则，避免把全屏和正文规则当作普通弹窗。
  const root = responsiveCss.nodes.find(node => node.type === 'rule' && node.selector === '.el-dialog')
  assert.ok(root)
  const style = Object.fromEntries(root.nodes.filter(node => node.type === 'decl').map(node => [node.prop, node.value]))
  assert.equal(style.display, 'flex')
  assert.equal(style['flex-direction'], 'column')
  assert.equal(style['max-width'], 'calc(100% - 32px)')
  assert.match(style['max-height'], /100dvh/)
  assert.equal(declarations(responsiveCss, '.el-dialog__body').overflow, 'auto')
  assert.equal(declarations(responsiveCss, '.el-dialog__footer')['flex-shrink'], '0')
})

check('固定宽抽屉和消息框不超过视口，超长正文仍可滚动', () => {
  assert.equal(declarations(responsiveCss, '.el-drawer.rtl')['max-width'], '100%')
  assert.equal(declarations(responsiveCss, '.el-drawer__body').overflow, 'auto')
  assert.equal(declarations(responsiveCss, '.el-message-box')['max-width'], 'calc(100vw - 32px)')
})

check('分页宽度受容器限制，按钮不删除、不压缩，窄空间允许横向滚动', () => {
  assert.equal(declarations(responsiveCss, '.el-pagination')['max-width'], '100%')
  assert.equal(declarations(responsiveCss, '.el-pagination')['overflow-x'], 'auto')
  assert.equal(declarations(responsiveCss, '.el-pagination > *')['flex-shrink'], '0')
})

check('独立改密页面可收缩和滚动，保留原表单及安全校验', () => {
  assert.equal(declarations(passwordCss, '.modfiyPasCss').overflow, 'auto')
  assert.equal(declarations(passwordCss, '.formContentCss')['max-width'], 'calc(100% - 24px)')
  assert.match(read('src/views/personMana/modifyPas.vue'), /createPasswordPolicyValidator/)
})

console.log(`统一适配样式回归：${checks} 项通过。`)
