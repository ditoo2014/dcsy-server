/* eslint-env es2021 */
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { registerHooks } from 'node:module'

// 使用项目指定的 Node.js 23.11.1 运行，无需浏览器或额外测试依赖。
// 本脚本检查样式生命周期；MainLayout 的主题恢复仍需在浏览器中验证。
const helperUrl = new URL('../src/styles/pageStyles.js', import.meta.url).href
const cssFixtures = new Map([
  ['normalize.css/normalize.css?inline', '/* normalize */'],
  ['element-plus/dist/index.css?inline', '/* element */'],
  ['./index.styl?inline', '/* business */'],
  ['./app.styl?inline', '/* application */']
])

// 只替换 Vite 的样式导入，直接执行项目 helper，不复制被测逻辑。
const hooks = registerHooks({
  resolve(specifier, context, nextResolve) {
    if (context.parentURL === helperUrl && cssFixtures.has(specifier)) {
      const source = `export default ${JSON.stringify(cssFixtures.get(specifier))}`
      return {
        url: `data:text/javascript,${encodeURIComponent(source)}`,
        shortCircuit: true
      }
    }
    return nextResolve(specifier, context)
  },
  load(url, context, nextLoad) {
    if (url === helperUrl) {
      return { format: 'module', source: readFileSync(new URL(url), 'utf8'), shortCircuit: true }
    }
    return nextLoad(url, context)
  }
})

const nodes = []
const pageClasses = new Set()
const documentFixture = {
  head: {
    querySelector(selector) {
      assert.equal(selector, 'style, link[rel="stylesheet"]')
      return nodes[0] || null
    },
    insertBefore(node, before) {
      const index = before ? nodes.indexOf(before) : nodes.length
      assert.ok(index >= 0, '样式插入位置必须仍在 head 中')
      nodes.splice(index, 0, node)
    }
  },
  body: {
    classList: {
      contains: name => pageClasses.has(name),
      toggle(name, active) {
        if (active) pageClasses.add(name)
        else pageClasses.delete(name)
      }
    }
  },
  querySelector(selector) {
    const match = /^style\[data-page-style="([^"]+)"\]$/.exec(selector)
    assert.ok(match, `未实现的测试选择器：${selector}`)
    return nodes.find(node => node.dataset.pageStyle === match[1]) || null
  },
  createElement(tag) {
    assert.equal(tag, 'style')
    return {
      dataset: {},
      get nextSibling() {
        return nodes[nodes.indexOf(this) + 1] || null
      },
      remove() {
        const index = nodes.indexOf(this)
        if (index >= 0) nodes.splice(index, 1)
      }
    }
  }
}

// 预先存在的组件样式，模拟开发服务或生产入口已插入的样式表。
const componentStyle = documentFixture.createElement('style')
componentStyle.dataset.pageStyle = 'component-fixture'
nodes.push(componentStyle)
const originalDocument = Object.getOwnPropertyDescriptor(globalThis, 'document')
Object.defineProperty(globalThis, 'document', { configurable: true, value: documentFixture })

function check(name, verify) {
  verify()
  console.log(`通过：${name}`)
}

const styleOrder = () => nodes.map(node => node.dataset.pageStyle)
const baseOrder = ['normalize', 'element', 'business', 'component-fixture']

try {
  const { setCollectionPageActive, mountCollectionPageStyles } = await import(helperUrl)

  check('基础样式顺序及业务默认状态', () => {
    assert.deepEqual(styleOrder(), baseOrder)
    assert.equal(nodes[0].media, 'all')
    assert.equal(nodes[2].media, 'all')
    assert.equal(nodes[2].textContent, '/* business */\n/* application */')
  })

  setCollectionPageActive(true)
  check('进入采集云停用 normalize 和业务样式，保留组件基础样式', () => {
    assert.equal(nodes[0].media, 'not all')
    assert.equal(nodes[2].media, 'not all')
    assert.notEqual(nodes[1].media, 'not all')
    assert.ok(pageClasses.has('collection-cloud-active'))
  })

  const removeCollectionStyle = mountCollectionPageStyles('/* original collection */')
  check('采集云全局样式位于组件样式之前', () => {
    assert.deepEqual(styleOrder(), ['normalize', 'element', 'business', 'collection', 'component-fixture'])
    assert.equal(nodes[3].textContent, '/* original collection */')
  })

  removeCollectionStyle()
  check('布局卸载移除采集云全局样式', () => {
    assert.deepEqual(styleOrder(), baseOrder)
  })

  setCollectionPageActive(false)
  check('退出采集云恢复业务样式并清除页面标记', () => {
    assert.equal(nodes[0].media, 'all')
    assert.equal(nodes[2].media, 'all')
    assert.ok(!pageClasses.has('collection-cloud-active'))
  })

  setCollectionPageActive(true)
  const removeReenteredStyle = mountCollectionPageStyles('/* original collection */')
  check('再次进入保持样式顺序且不重复创建基础节点', () => {
    assert.deepEqual(styleOrder(), ['normalize', 'element', 'business', 'collection', 'component-fixture'])
    assert.equal(nodes[0].media, 'not all')
    assert.equal(nodes[2].media, 'not all')
  })
  removeReenteredStyle()
  setCollectionPageActive(false)
  assert.deepEqual(styleOrder(), baseOrder)
} finally {
  hooks.deregister()
  if (originalDocument) Object.defineProperty(globalThis, 'document', originalDocument)
  else delete globalThis.document
}
