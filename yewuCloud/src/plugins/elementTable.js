/** 表格组件全局注册，保证 ElTable / ElTableColumn 配套使用 */
import { ElTable, ElTableColumn } from 'element-plus'

export function registerElementTable(app) {
  app.component(ElTable.name, ElTable)
  app.component(ElTableColumn.name, ElTableColumn)
}
