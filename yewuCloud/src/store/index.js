import { createPinia, defineStore } from 'pinia'
import appModule from './modules/app'
import errorLogModule from './modules/errorLog'
import permissionModule from './modules/permission'
import settingsModule from './modules/settings'
import tagsViewModule from './modules/tagsView'
import userModule from './modules/user'
import getterDefinitions from './getters'

/**
 * DCSY统一使用Pinia保存状态。
 * 兼容层暂时保留原页面的dispatch/commit形式，升级状态框架时不改变页面DOM和样式。
 */
const modules = {
  app: appModule,
  errorLog: errorLogModule,
  permission: permissionModule,
  settings: settingsModule,
  tagsView: tagsViewModule,
  user: userModule
}

const cloneState = source => {
  if (typeof structuredClone === 'function') return structuredClone(source)
  return JSON.parse(JSON.stringify(source))
}

const useDcsyStore = defineStore('dcsy', {
  state: () => Object.fromEntries(
    Object.entries(modules).map(([name, module]) => [name, cloneState(module.state)])
  )
})

export const pinia = createPinia()
let piniaStore

function getStore() {
  if (!piniaStore) piniaStore = useDcsyStore(pinia)
  return piniaStore
}

function splitType(type) {
  const separator = type.indexOf('/')
  return separator < 0 ? ['', type] : [type.slice(0, separator), type.slice(separator + 1)]
}

const store = {
  install(app) {
    app.use(pinia)
    app.config.globalProperties.$store = store
  },
  get state() {
    return getStore().$state
  },
  commit(type, payload) {
    const [moduleName, mutationName] = splitType(type)
    const mutation = modules[moduleName]?.mutations?.[mutationName]
    if (!mutation) throw new Error(`未找到状态变更: ${type}`)
    mutation(getStore()[moduleName], payload)
  },
  dispatch(type, payload) {
    const [moduleName, actionName] = splitType(type)
    const action = modules[moduleName]?.actions?.[actionName]
    if (!action) return Promise.reject(new Error(`未找到状态操作: ${type}`))
    const context = {
      state: getStore()[moduleName],
      rootState: getStore().$state,
      getters: store.getters,
      rootGetters: store.getters,
      commit: (name, value, options) => store.commit(
        options?.root || name.includes('/') ? name : `${moduleName}/${name}`,
        value
      ),
      dispatch: (name, value, options) => store.dispatch(
        options?.root || name.includes('/') ? name : `${moduleName}/${name}`,
        value
      )
    }
    return Promise.resolve(action(context, payload))
  },
  getters: {}
}

Object.entries(getterDefinitions).forEach(([name, getter]) => {
  Object.defineProperty(store.getters, name, {
    enumerable: true,
    get: () => getter(getStore().$state)
  })
})

export default store
