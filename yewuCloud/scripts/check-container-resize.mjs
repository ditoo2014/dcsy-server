// 容器适配回归：覆盖父布局变化、浏览器缩放、隐藏恢复和缓存页清理。
/* global globalThis */
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { parse, compileTemplate } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'

const source = await readFile(new URL('../src/utils/container-resize.js', import.meta.url), 'utf8')
const { createContainerResizeWatcher } = await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`)
const frames = new Map()
const listeners = new Set()
const observers = []
let nextFrame = 0
globalThis.window = {
  devicePixelRatio: 1,
  addEventListener(type, callback) {
    assert.equal(type, 'resize')
    listeners.add(callback)
  },
  removeEventListener(type, callback) {
    assert.equal(type, 'resize')
    listeners.delete(callback)
  }
}
globalThis.requestAnimationFrame = callback => {
  frames.set(++nextFrame, callback)
  return nextFrame
}
globalThis.cancelAnimationFrame = frame => frames.delete(frame)
globalThis.ResizeObserver = class {
  constructor(callback) {
    this.callback = callback
    observers.push(this)
  }
  observe(container) {
    this.container = container
  }
  disconnect() {
    this.container = null
  }
}
function flushFrames() {
  const pending = [...frames.values()]
  frames.clear()
  pending.forEach(callback => callback())
}

const container = { clientWidth: 960, clientHeight: 420 }
let resizeCount = 0
let callbackReady = true
const watcher = createContainerResizeWatcher(() => container, element => {
  assert.equal(element, container)
  if (!callbackReady) return false
  resizeCount += 1
  return true
})

watcher.start()
watcher.start()
assert.equal(observers.length, 1)
assert.equal(observers[0].container, container)
assert.equal(listeners.size, 1)
flushFrames()
assert.equal(resizeCount, 1)
console.log('通过：重复激活不会重复注册，首次显示完成调整')

container.clientWidth = 720
observers[0].callback()
observers[0].callback()
listeners.forEach(callback => callback())
watcher.schedule()
assert.equal(frames.size, 1)
flushFrames()
assert.equal(resizeCount, 2)
watcher.schedule()
flushFrames()
assert.equal(resizeCount, 2)
console.log('通过：父容器变化会调整，多种通知同帧合并，相同尺寸去重')

window.devicePixelRatio = 1.25
listeners.forEach(callback => callback())
flushFrames()
assert.equal(resizeCount, 3)
console.log('通过：浏览器缩放改变像素比时重新调整')

container.clientWidth = 0
observers[0].callback()
flushFrames()
assert.equal(resizeCount, 3)
container.clientWidth = 720
observers[0].callback()
flushFrames()
assert.equal(resizeCount, 4)
console.log('通过：隐藏不调整，再次显示恢复相同尺寸也会调整')

container.clientHeight = 480
callbackReady = false
observers[0].callback()
flushFrames()
assert.equal(resizeCount, 4)
callbackReady = true
watcher.schedule()
flushFrames()
assert.equal(resizeCount, 5)
console.log('通过：图表尚未就绪不会记录成功，后续就绪允许重试')

watcher.schedule()
assert.equal(frames.size, 1)
watcher.stop()
assert.equal(frames.size, 0)
assert.equal(listeners.size, 0)
assert.equal(observers[0].container, null)
observers[0].callback()
watcher.schedule()
assert.equal(frames.size, 0)
console.log('通过：停用清理监听和待执行帧，晚到通知不再调整')

watcher.start()
flushFrames()
assert.equal(resizeCount, 6)
watcher.stop()
console.log('通过：缓存页重新激活会重新测量')

delete globalThis.ResizeObserver
watcher.start()
flushFrames()
container.clientHeight = 500
listeners.forEach(callback => callback())
flushFrames()
assert.equal(resizeCount, 8)
watcher.stop()
assert.equal(listeners.size, 0)
console.log('通过：无 ResizeObserver 时保留窗口调整兼容能力')

const expectedError = new Error('模拟图表调整失败')
const errorWatcher = createContainerResizeWatcher(() => container, () => { throw expectedError })
errorWatcher.start()
assert.throws(flushFrames, error => error === expectedError)
errorWatcher.stop()
console.log('通过：异常不会被工具静默吞掉')

// 使用真实地图组件的生命周期；只替换地图 SDK、定时器，不访问地图服务。
globalThis.ResizeObserver = observers[0].constructor
for (const file of ['mapComplex/DetailMap.vue', 'mapDriving/MapDriving.vue']) {
  const filename = new URL(`../src/components/a_myComp/${file}`, import.meta.url)
  const { descriptor, errors } = parse(await readFile(filename, 'utf8'), { filename: filename.pathname })
  assert.deepEqual(errors, [])
  assert.deepEqual(compileTemplate({ source: descriptor.template.content, filename: filename.pathname, id: 'map-size-check' }).errors, [])
  let finishLoading
  const sdkLoading = new Promise(resolve => { finishLoading = resolve })
  const dependencies = {
    vue: { markRaw: value => value },
    '@/utils/amap-loader': { loadBusinessMap: () => sdkLoading },
    '@/utils/container-resize.js': { createContainerResizeWatcher }
  }
  const module = { exports: {}}
  const code = transformSync(descriptor.script.content, { format: 'cjs' }).code
  const timers = new Map()
  let timerId = 0
  const setTimer = callback => { timers.set(++timerId, callback); return timerId }
  new Function('module', 'exports', 'require', 'setTimeout', 'clearTimeout', code)(
    module, module.exports, name => dependencies[name], setTimer, id => timers.delete(id)
  )
  const options = module.exports.default
  const mapElement = { clientWidth: 800, clientHeight: 400 }
  const state = {
    ...options.data(),
    $el: mapElement,
    $refs: { mapContainer: mapElement },
    $nextTick: callback => callback(),
    $message: { error: message => assert.fail(message) }
  }
  for (const [name, method] of Object.entries(options.methods)) state[name] = method.bind(state)
  options.mounted.call(state)
  flushFrames()
  const observer = observers.at(-1)
  assert.equal(observer.container, mapElement)
  let mapResizes = 0
  let mapDestroys = 0
  const map = { resize: () => { mapResizes++ }, destroy: () => { mapDestroys++ } }
  state[file.startsWith('mapDriving') ? 'map' : 'mapKey'] = map
  state.__mapSizeWatcher.schedule()
  flushFrames()
  assert.equal(mapResizes, 1)
  mapElement.clientWidth = 640
  observer.callback()
  flushFrames()
  assert.equal(mapResizes, 2)
  mapElement.clientHeight = 0
  observer.callback()
  flushFrames()
  assert.equal(mapResizes, 2)
  mapElement.clientHeight = 400
  observer.callback()
  flushFrames()
  assert.equal(mapResizes, 3)
  options.deactivated.call(state)
  assert.equal(listeners.size, 0)
  assert.equal(observer.container, null)
  options.activated.call(state)
  flushFrames()
  assert.equal(mapResizes, 4)
  state.__mapSizeWatcher.schedule()
  options.beforeUnmount.call(state)
  assert.equal(frames.size, 0)
  assert.equal(listeners.size, 0)
  assert.equal(timers.size, 0)
  assert.equal(mapDestroys, 1)
  finishLoading({})
  await Promise.resolve()
  await Promise.resolve()
  assert.equal(state[file.startsWith('mapDriving') ? 'map' : 'mapKey'], null)
  console.log(`通过：${file} 容器变化、隐藏恢复、缓存页与卸载清理`)
}
