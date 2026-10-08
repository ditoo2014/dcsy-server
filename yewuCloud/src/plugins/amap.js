/** 高德地图 @vuemap/vue-amap 按需注册 */
import { ElAmap } from '@vuemap/vue-amap'
import '@vuemap/vue-amap/dist/style.css'

export function registerAmap(app) {
  app.component('ElAmap', ElAmap)
}
