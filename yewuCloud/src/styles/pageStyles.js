import normalizeCss from 'normalize.css/normalize.css?inline'
import elementCss from 'element-plus/dist/index.css?inline'
import businessCss from './index.styl?inline'
import applicationCss from './app.styl?inline'

const PAGE_CLASS = 'collection-cloud-active'

function createStyle(name, css, before = null) {
  let style = document.querySelector(`style[data-page-style="${name}"]`)
  if (!style) {
    style = document.createElement('style')
    style.dataset.pageStyle = name
    document.head.insertBefore(style, before)
  }
  style.textContent = css
  return style
}

// 开发和打包后均保持基础样式在组件样式之前，避免 Vite 合并 CSS 改变优先级。
const firstStyle = document.head.querySelector('style, link[rel="stylesheet"]')
const normalizeStyle = createStyle('normalize', normalizeCss, firstStyle)
const elementStyle = createStyle('element', elementCss, firstStyle)
const businessStyle = createStyle('business', `${businessCss}\n${applicationCss}`, firstStyle)

export function setCollectionPageActive(active) {
  document.body.classList.toggle(PAGE_CLASS, active)
  // 完整停用业务规则，才能保留原采集云字体、组件字重和浏览器原生滚动条。
  normalizeStyle.media = active ? 'not all' : 'all'
  businessStyle.media = active ? 'not all' : 'all'
}

export function mountCollectionPageStyles(css) {
  // 原项目先加载全局样式、再加载路由组件，进入/再次进入时均保持这一顺序。
  const style = createStyle('collection', css, businessStyle.nextSibling)
  return () => style.remove()
}

setCollectionPageActive(document.body.classList.contains(PAGE_CLASS))

// ?inline 样式不由 Vite 自动插入，热更新时同步更新现有节点且不改变节点顺序。
if (import.meta.hot) {
  import.meta.hot.accept('normalize.css/normalize.css?inline', module => {
    if (module) normalizeStyle.textContent = module.default
  })
  import.meta.hot.accept('element-plus/dist/index.css?inline', module => {
    if (module) elementStyle.textContent = module.default
  })
  let currentBusinessCss = businessCss
  let currentApplicationCss = applicationCss
  import.meta.hot.accept(['./index.styl?inline', './app.styl?inline'], modules => {
    if (modules[0]) currentBusinessCss = modules[0].default
    if (modules[1]) currentApplicationCss = modules[1].default
    businessStyle.textContent = `${currentBusinessCss}\n${currentApplicationCss}`
  })
}
