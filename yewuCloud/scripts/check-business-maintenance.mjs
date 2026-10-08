// 运维页离线回归：只执行真实查询整理逻辑与组件方法，不连接后端、不改数据库。
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { parse, compileTemplate } from '@vue/compiler-sfc'
import { parse as parseJavaScript } from '@babel/parser'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const helperPath = path.join(root, 'src/views/lldcsy/smzqgl/maintenance-data.js')
const helper = await import(`data:text/javascript;base64,${Buffer.from(fs.readFileSync(helperPath, 'utf8')).toString('base64')}`)
const { readMaintenanceFilters, loadMaintenanceTable } = helper
let passed = 0
async function check(name, run) {
  await run()
  passed++
  console.log(`通过：${name}`)
}
const response = (rows, total = rows.length) => ({ code: 200, return: { data: rows, total } })
const filters = { equipmentCode: '', equipmentTypeName: '', minimumCount: 0, startTime: 0, endTime: 2000000000 }
const records = Array.from({ length: 12 }, (_, i) => ({
  id: i + 1, equipmentId: i < 5 ? 1 : 2,
  maintainTitle: `运维 ${i + 1}`, maintainContent: '真实记录',
  maintainDetails: { create_time: '2026-09-21 10:25:30', imageUrl: 'photo.jpg' }
}))
const equipment = [
  { id: 1, equipmentCode: '电池A', equipmentTypeName: '钠电', positionStatus: 4, belongToSiteId: 9 },
  { id: 2, equipmentCode: '电池B', equipmentTypeName: '锂硫', positionStatus: 6, belongToSiteId: 9 }
]
function apiFixture(rows = records) {
  const calls = { equipment: [], site: [], records: [] }
  return {
    calls,
    async findRecords(params) {
      calls.records.push(params)
      let matching = rows
      if (params.equipmentCode) {
        matching = matching.filter(row => {
          const item = equipment.find(value => value.id === row.equipmentId) || {}
          return String(item.equipmentCode || '').includes(params.equipmentCode) ||
            String(item.equipmentName || '').includes(params.equipmentCode)
        })
      }
      if (params.equipmentTypeName) {
        matching = matching.filter(row => {
          const item = equipment.find(value => value.id === row.equipmentId) || {}
          return String(item.equipmentTypeName || '').includes(params.equipmentTypeName)
        })
      }
      if (params.minimumCount > 0) {
        const counts = new Map()
        matching.forEach(row => counts.set(row.equipmentId, (counts.get(row.equipmentId) || 0) + 1))
        matching = matching.filter(row => counts.get(row.equipmentId) >= params.minimumCount)
      }
      return response(matching.slice((params.pageNum - 1) * params.pageSize, params.pageNum * params.pageSize), matching.length)
    },
    async findEquipment({ id }) { calls.equipment.push(id); return response(equipment.filter(item => item.id === id)) },
    async findSite({ id }) { calls.site.push(id); return response([{ id, siteSerialNumber: '站点-9' }]) }
  }
}

function component(relativePath, overrides = {}) {
  const source = fs.readFileSync(path.join(root, relativePath), 'utf8')
  const { descriptor, errors } = parse(source)
  assert.deepEqual(errors, [])
  assert.deepEqual(compileTemplate({ source: descriptor.template.content, filename: relativePath, id: 'maintenance-test' }).errors, [])
  const script = descriptor.script.content
  const ast = parseJavaScript(script, { sourceType: 'module' })
  const dependencies = {
    Banner: {}, UploadFile: {}, Ywxq: {}, processDrawList: [], totalImage: 'total.png',
    ...helper, ...overrides
  }
  for (const node of ast.program.body.filter(node => node.type === 'ImportDeclaration')) {
    for (const specifier of node.specifiers) {
      const name = specifier.local.name
      if (!(name in dependencies)) dependencies[name] = node.source.value.endsWith('.png') ? node.source.value : (() => {})
    }
  }
  let body = script
  for (const node of [...ast.program.body].reverse()) {
    if (node.type === 'ImportDeclaration') body = body.slice(0, node.start) + body.slice(node.end)
  }
  body = body.replace('export default', 'return')
  const options = new Function(...Object.keys(dependencies), body)(...Object.values(dependencies))
  const context = {}
  Object.assign(context, options.data?.call(context) || {})
  for (const [name, method] of Object.entries(options.methods || {})) context[name] = method.bind(context)
  return { options, context, source, descriptor }
}

await check('合并电池、类型和次数筛选，5 次仍按原版计入', async () => {
  const result = await loadMaintenanceTable(apiFixture(), { ...filters, equipmentCode: '电池A', equipmentTypeName: '钠', minimumCount: 5 })
  assert.equal(result.rows.length, 5)
  assert.equal(result.equipmentCount, 1)
  assert.equal(result.rows[0].positionStatus, 4)
  assert.equal(result.rows[0].dcid, '电池A')
})
await check('关联资料按实体去重，保留真实站点、图片和状态', async () => {
  const api = apiFixture()
  const result = await loadMaintenanceTable(api, filters)
  assert.equal(result.rows.length, 12)
  assert.equal(result.equipmentCount, 2)
  assert.deepEqual(api.calls.equipment.sort(), [1, 2])
  assert.deepEqual(api.calls.site, [9])
  assert.equal(result.rows[0].zdid, '站点-9')
  assert.equal(result.rows[0].imgUrl, 'photo.jpg')
})
await check('设备或站点补充查询失败时仍保留运维主记录', async () => {
  const api = apiFixture()
  api.findEquipment = async ({ id }) => {
    if (id === 2) throw new Error('设备资料服务暂不可用')
    return response(equipment.filter(item => item.id === id))
  }
  const result = await loadMaintenanceTable(api, filters)
  assert.equal(result.rows.length, 12)
  assert.equal(result.rows.filter(row => row.equipmentId === 2).length, 7)
  assert.equal(result.rows.find(row => row.equipmentId === 2).dcid, '-')
})
await check('优先使用运维记录内的设备资料，补充接口只查询缺失字段', async () => {
  const api = apiFixture([
    { ...records[0], equipmentCode: '记录编码', equipmentName: '记录名称', equipmentTypeName: '记录类型',
      siteSerialNumber: '记录站点', positionStatus: 3, createTime: 1700000000,
      maintainDetails: { create_time: '旧时间' }, maintainPhotos: ['one.png', 'two.png'] }
  ])
  const result = await loadMaintenanceTable(api, filters)
  assert.equal(result.rows[0].dcid, '记录编码')
  assert.equal(result.rows[0].equipmentTypeName, '记录类型')
  assert.equal(result.rows[0].zdid, '记录站点')
  assert.equal(result.rows[0].positionStatus, 3)
  assert.equal(result.rows[0].ywsj, new Date(1700000000 * 1000).toLocaleString('zh-CN').replace(/\//g, '-'))
  assert.deepEqual(result.rows[0].imgUrls, ['one.png', 'two.png'])
  assert.deepEqual(api.calls.equipment, [])
  assert.deepEqual(api.calls.site, [])
})
await check('按页读完 501 条，不再只取首页', async () => {
  const rows = Array.from({ length: 501 }, (_, index) => ({ ...records[0], id: index + 1 }))
  const api = apiFixture(rows)
  assert.equal((await loadMaintenanceTable(api, filters)).rows.length, 501)
  assert.equal(api.calls.records.length, 2)
})
await check('分页记录没有 id 时仍按接口总数完整读取', async () => {
  const rows = Array.from({ length: 501 }, () => ({
    equipmentId: 1, maintainContent: '无编号记录', maintainDetails: {}
  }))
  const api = apiFixture(rows)
  const result = await loadMaintenanceTable(api, filters)
  assert.equal(result.rows.length, 501)
  assert.equal(api.calls.records.length, 2)
})
await check('保留日期范围的时分秒，并支持反向选择的时间', () => {
  const start = new Date(2026, 8, 21, 10, 25, 30)
  const end = new Date(2026, 8, 21, 12, 35, 40)
  const result = readMaintenanceFilters([{ name: '选择时间段', timeDateArr: [end, start] }])
  assert.equal(result.startTime, start.getTime() / 1000)
  assert.equal(result.endTime, end.getTime() / 1000)
})
await check('空返回显示零，关联信息缺失不伪造状态', async () => {
  const api = apiFixture()
  api.findRecords = async () => ({ code: 200, return: null })
  assert.deepEqual(await loadMaintenanceTable(api, filters), { rows: [], equipmentCount: 0 })
  const missing = apiFixture([{ ...records[0], equipmentId: 99, maintainDetails: null }])
  const result = await loadMaintenanceTable(missing, filters)
  assert.equal(result.rows[0].positionStatus, null)
  assert.equal(result.rows[0].imgUrl, '')
  assert.equal(result.rows[0].dcid, '-')
})
await check('业务错误和重复分页明确失败，不按无数据或重复记录处理', async () => {
  const api = apiFixture()
  api.findRecords = async () => ({ code: 400, msg: '查询失败' })
  await assert.rejects(loadMaintenanceTable(api, filters), /查询失败/)
  api.findRecords = async () => response([records[0]], 2)
  await assert.rejects(loadMaintenanceTable(api, filters), /重复的分页数据/)
})
await check('查询按钮、输入回车和重置全部走统一查询', async () => {
  const { context } = component('src/views/lldcsy/smzqgl/ywgl.vue')
  let count = 0
  context.getEquipmentmitrcd = async () => { count++ }
  context.bannerList[2].input = '电池A'
  context.bannerList[3].input = '钠电'
  context.bannerList[4].value = '5'
  context.bannerList[5].timeDateArr = [new Date(), new Date()]
  await context.clickBannerBtn({ buttonName: '查询' })
  await context.inputBannerKeyupEnter()
  await context.resetData()
  assert.equal(count, 3)
  assert.equal(context.bannerList[2].input, '')
  assert.equal(context.bannerList[3].input, '')
  assert.equal(context.bannerList[4].value, '')
  assert.deepEqual(context.bannerList[5].timeDateArr, [])
})
await check('总次数取全部匹配记录，分页不改总数且页容量变化返回首页', async () => {
  const { context } = component('src/views/lldcsy/smzqgl/ywgl.vue', { loadMaintenanceTable: async () => ({ rows: records, equipmentCount: 2 }) })
  await context.search()
  assert.equal(context.bannerList[0].num, '12')
  assert.equal(context.tableData.length, 10)
  context.handleCurrentChange(2)
  assert.equal(context.tableData.length, 2)
  assert.equal(context.bannerList[0].num, '12')
  context.handleSizeChange(20)
  assert.equal(context.currentPage, 1)
  assert.equal(context.tableData.length, 12)
})
await check('网络失败清理加载状态与旧数据，并保留中文错误提示', async () => {
  const notices = []
  const { context } = component('src/views/lldcsy/smzqgl/ywgl.vue', {
    loadMaintenanceTable: async () => { throw new Error('网络暂时不可用') }, Message: value => notices.push(value)
  })
  context.tableData = records
  await context.search()
  assert.equal(context.showLoading, false)
  assert.equal(context.queryError, '网络暂时不可用')
  assert.equal(context.total, 0)
  assert.deepEqual(context.tableData, [])
  assert.equal(notices.length, 1)
})
await check('新增运维必须有图片，多个文件只保留合法图片', async () => {
  const notices = []
  const { context } = component('src/views/lldcsy/smzqgl/ywgl.vue', {
    Message: value => notices.push(value),
    fileIsImg: () => ({ sureIsImg: true })
  })
  context.selectFile({ fileList: [
    { name: 'a.png', size: 100 },
    { name: 'b.png', size: 6 * 1024 * 1024 },
    { name: 'c.png', size: 100 }
  ] })
  assert.deepEqual(context.form.fileList.map(file => file.name), ['a.png', 'c.png'])
  context.form.dcid = '电池A'
  context.form.gzms = '故障描述'
  context.form.gzxq = '故障详情'
  context.showDrawer = true
  context.form.fileList = []
  await context.addNewMaintain()
  assert.equal(context.showDrawer, true)
  assert.match(notices.at(-1).message, /图片/)
})
await check('新增运维按编码精确匹配，并保存故障详情和全部图片', async () => {
  const queries = []
  const created = []
  const { context } = component('src/views/lldcsy/smzqgl/ywgl.vue', {
    Message: () => {},
    findSelf: async () => ({ return: { user: { id: 8, username: '测试用户' } } }),
    findEquipmentListExactMatch: async query => {
      queries.push(query)
      if (query.equipmentCode) return response([{ id: 11, equipmentCode: 'BAT-11', equipmentName: '旧名称' }])
      return response([])
    },
    findEquipmentcirlog: async () => response([{ operateType: 3 }]),
    uploadEquipmentmitrcdImage: async ({ file }) => ({ code: 200, data: { url: `/go/images/file/${file.name}` } }),
    addEquipmentmitrcd: async value => { created.push(value); return { code: 200 } },
    getCurrentDateTimeString: () => '2026-09-21 10:00:00'
  })
  context.getEquipmentmitrcd = async () => {}
  context.showDrawer = true
  context.form = {
    dcid: 'BAT-11', ssgs: '测试公司', gzms: '更换模块', gzxq: '更换模块详情',
    fileList: [{ name: 'a.png' }, { name: 'b.png' }]
  }
  await context.addNewMaintain()
  assert.equal(queries[0].equipmentCode, 'BAT-11')
  assert.equal(created.length, 1)
  assert.equal(created[0].maintainContent, '更换模块详情')
  assert.deepEqual(created[0].maintainPhotos, ['/go/images/file/a.png', '/go/images/file/b.png'])
  assert.equal(created[0].maintainDetails.fault_detail, '更换模块详情')
  assert.equal(context.showDrawer, false)
})
await check('乱序返回及离开页面后的请求均不能覆盖最新结果', async () => {
  const pending = []
  const { context, options } = component('src/views/lldcsy/smzqgl/ywgl.vue', {
    loadMaintenanceTable: () => new Promise(resolve => pending.push(resolve))
  })
  const first = context.search()
  const second = context.search()
  pending[1]({ rows: [records[1]], equipmentCount: 1 })
  await second
  pending[0]({ rows: records, equipmentCount: 2 })
  await first
  assert.equal(context.total, 1)
  assert.equal(context.tableData[0].id, 2)
  const third = context.search()
  options.beforeUnmount.call(context)
  pending[2]({ rows: [], equipmentCount: 0 })
  await third
  assert.equal(context.total, 1)
})
await check('详情按真实状态切换原有图片，缺失或未知不显示上站', () => {
  const { options } = component('src/views/lldcsy/smzqgl/comp/ywxq.vue')
  const renderImage = positionStatus => options.computed.dcdqztUrl.call({ formData: { positionStatus } })
  assert.match(renderImage(4), /退网下站/)
  assert.match(renderImage(6), /检测淘汰/)
  assert.equal(renderImage(undefined), '')
  assert.equal(renderImage(0), '')
})
await check('导出透传完整筛选和精确时间，并直接下载响应文件流', async () => {
  const requests = []
  const downloads = []
  const mime = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
  const file = { data: new Blob(['测试文件'], { type: mime }), headers: { 'content-type': mime } }
  const { context } = component('src/views/lldcsy/smzqgl/ywgl.vue', {
    equipmentmitrcdExport: async value => { requests.push(value); return file },
    exportBlobFile: value => downloads.push(value)
  })
  context.bannerList[2].input = '电池A'
  context.bannerList[3].input = '钠电'
  context.bannerList[4].value = '5'
  const dates = [new Date(2026, 8, 20, 10, 20, 30), new Date(2026, 8, 21, 12, 0, 1)]
  context.bannerList[5].timeDateArr = dates
  await context.exportExcel()
  assert.equal(requests[0].equipmentCode, '电池A')
  assert.equal(requests[0].equipmentTypeName, '钠电')
  assert.equal(requests[0].minimumCount, 5)
  assert.equal(requests[0].startTime, dates[0].getTime() / 1000)
  assert.equal(requests[0].endTime, dates[1].getTime() / 1000)
  assert.deepEqual(downloads, [file])
})
await check('导出错误响应不另存成损坏表格，网络错误使用中文提示', async () => {
  const notices = []
  const { context } = component('src/views/lldcsy/smzqgl/ywgl.vue', {
    equipmentmitrcdExport: async () => ({ data: new Blob([JSON.stringify({ code: 400, msg: '暂无可导出的记录' })], { type: 'application/json' }) }),
    exportBlobFile: () => assert.fail('不应下载错误 JSON'),
    Message: value => notices.push(value)
  })
  await context.exportExcel()
  assert.equal(notices[0].message, '暂无可导出的记录')
  assert.match(helper.maintenanceErrorMessage(new Error('Network Error')), /请检查网络/)
})

await check('列表请求向后端透传编码、类型、次数和时间筛选', async () => {
  const api = apiFixture([])
  await loadMaintenanceTable(api, { ...filters, equipmentCode: 'BAT-1', equipmentTypeName: '钠电池', minimumCount: 5 })
  assert.deepEqual(api.calls.records[0], {
    pageNum: 1, pageSize: 500, startTime: filters.startTime, endTime: filters.endTime,
    equipmentCode: 'BAT-1', equipmentTypeName: '钠电池', minimumCount: 5
  })
})

const basicDataSource = fs.readFileSync(path.join(root, 'src/api/basicData.js'), 'utf8')
await check('设备运维查询 API 请求体保留后端筛选字段', () => {
  assert.match(basicDataSource, /equipmentCode:\s*param\.equipmentCode/)
  assert.match(basicDataSource, /equipmentTypeName:\s*param\.equipmentTypeName/)
  assert.match(basicDataSource, /minimumCount:\s*param\.minimumCount/)
})

await check('旧标题占位符回退正文，空状态会补查且保留后端元信息', async () => {
  const api = apiFixture([{ ...records[0], equipmentCode: '后端编码', equipmentName: '后端名称',
    equipmentTypeName: '后端类型', siteSerialNumber: '后端站点', positionStatus: null,
    maintainTitle: '-', maintainContent: '旧运维正文' }])
  const result = await loadMaintenanceTable(api, filters)
  assert.deepEqual(api.calls.equipment, [1])
  assert.equal(result.rows[0].positionStatus, 4)
  assert.equal(result.rows[0].dcid, '后端编码')
  assert.equal(result.rows[0].gzms, '旧运维正文')
  assert.equal(result.rows[0].ms, '旧运维正文')
})

function createFixture(overrides = {}) {
  const notices = []
  const created = []
  const fixture = component('src/views/lldcsy/smzqgl/ywgl.vue', {
    Message: value => notices.push(value),
    findSelf: async () => ({ return: { user: { id: 8, username: '测试用户' } } }),
    findEquipmentListExactMatch: async () => response([{ id: 11, equipmentCode: 'BAT-11', positionStatus: 3 }]),
    findEquipmentcirlog: async () => assert.fail('状态已存在，不应查询流转'),
    uploadEquipmentmitrcdImage: async () => ({ code: 200, data: { url: '/go/images/file/test.png' } }),
    addEquipmentmitrcd: async value => { created.push(value); return { code: 200 } },
    ...overrides
  })
  fixture.context.add()
  fixture.context.form = { dcid: 'BAT-11', gzms: '描述', gzxq: '详情', fileList: [{ name: 'test.png' }] }
  fixture.context.getEquipmentmitrcd = async () => {}
  return { ...fixture, notices, created }
}

await check('新增遇到重复精确编码时拒绝提交', async () => {
  const fixture = createFixture({
    findEquipmentListExactMatch: async () => response([
      { id: 11, equipmentCode: 'BAT-11' }, { id: 12, equipmentCode: 'BAT-11' }
    ])
  })
  await fixture.context.addNewMaintain()
  assert.equal(fixture.created.length, 0)
  assert.match(fixture.notices[0].message, /多个设备/)
  assert.equal(fixture.context.submitting, false)
})

await check('名称回退仅匹配无编码设备，不匹配已具有其他编码的同名设备', async () => {
  const fixture = createFixture({
    findEquipmentListExactMatch: async query => response(query.equipmentCode ? [] : [
      { id: 11, equipmentCode: 'OTHER', equipmentName: 'BAT-11', positionStatus: 3 },
      { id: 12, equipmentCode: '-', equipmentName: 'BAT-11', positionStatus: 3 }
    ])
  })
  await fixture.context.addNewMaintain()
  assert.equal(fixture.created.length, 1)
  assert.equal(fixture.created[0].equipmentId, 12)
})

await check('新增和导入切换重置表单文件，不把 Excel 带入图片提交', () => {
  const { context } = component('src/views/lldcsy/smzqgl/ywgl.vue')
  context.upLoadDataFile()
  context.form.fileList = [{ name: 'maintenance.xlsx' }]
  context.clickBannerBtn({ buttonName: '新增' })
  assert.equal(context.dialogStatus, 'add')
  assert.deepEqual(context.form, { dcid: '', ssgs: '', gzms: '', gzxq: '', fileList: [], files: [] })
  context.form.fileList = [{ name: 'maintenance.png' }]
  context.upLoadDataFile()
  assert.equal(context.dialogStatus, 'upload')
  assert.deepEqual(context.form.fileList, [])
})

await check('新增连点只提交一次，提交期间切换入口不覆盖表单', async () => {
  let resolveUser
  let userRequests = 0
  const fixture = createFixture({
    findSelf: () => {
      userRequests++
      return new Promise(resolve => { resolveUser = resolve })
    }
  })
  const first = fixture.context.addNewMaintain()
  await fixture.context.addNewMaintain()
  fixture.context.upLoadDataFile()
  assert.equal(fixture.context.dialogStatus, 'add')
  assert.equal(fixture.context.form.dcid, 'BAT-11')
  assert.equal(userRequests, 1)
  resolveUser({ return: { user: { id: 8, username: '测试用户' } } })
  await first
  assert.equal(fixture.created.length, 1)
  assert.equal(fixture.context.submitting, false)
})

await check('导入连点只提交一次并展示实际成功重复失败数', async () => {
  let completeImport
  let importCount = 0
  const { context } = component('src/views/lldcsy/smzqgl/ywgl.vue', {
    equipmentmitrcdImport: () => {
      importCount++
      return new Promise(resolve => { completeImport = resolve })
    }
  })
  const notifications = []
  context.$notify = value => notifications.push(value)
  context.search = async () => {}
  context.upLoadDataFile()
  context.form.fileList = [{ name: 'maintenance.xlsx' }]
  const first = context.saveFile()
  await context.saveFile()
  context.add()
  assert.equal(context.dialogStatus, 'upload')
  assert.equal(importCount, 1)
  completeImport({ code: 200, return: { successCount: 2, dumplicateCount: 3, failCount: 1 } })
  await first
  assert.match(notifications[0].message, /成功 2 条，重复 3 条，失败 1 条/)
  assert.equal(context.submitting, false)
})

await check('新增遇到英文网络异常使用中文提示且释放提交锁', async () => {
  const fixture = createFixture({ findSelf: async () => { throw new Error('Network Error') } })
  await fixture.context.addNewMaintain()
  assert.match(fixture.notices[0].message, /运维记录添加失败，请检查网络/)
  assert.equal(fixture.context.submitting, false)
})

console.log(`运维页面离线回归完成：${passed} 项通过。`)
