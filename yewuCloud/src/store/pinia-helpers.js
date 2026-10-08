import store from './index'

/** Options API页面使用的Pinia getter映射辅助函数。 */
export function mapGetters(names) {
  return Object.fromEntries(names.map(name => [name, function() {
    return store.getters[name]
  }]))
}

/** 兼容原有mapState对象写法，底层数据来自Pinia。 */
export function mapState(mappers) {
  return Object.fromEntries(Object.entries(mappers).map(([name, mapper]) => [name, function() {
    return mapper(store.state)
  }]))
}
