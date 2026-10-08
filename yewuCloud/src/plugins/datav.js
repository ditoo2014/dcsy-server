/** DataV 按需注册（仅注册业务用到的组件） */
import { ScrollBoard } from '@kjgl77/datav-vue3'
import '@kjgl77/datav-vue3/dist/style.css'

export function registerDatav(app) {
  app.component('DvScrollBoard', ScrollBoard)
}
