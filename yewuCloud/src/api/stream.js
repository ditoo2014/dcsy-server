import { JsonPost } from '@/api/api.js'

export function getStreamList(params) { // 数据池列表
  return JsonPost(`tolink/stream/findpage`, {
    page: params.page, // 分页页码
    size: params.limit, // 返回数据条数
    search: params.search
  })
}

export function streamCreate(params) { // 数据池添加
  return JsonPost(`tolink/stream/create`, {
    name: params.name.replace(/\s*/g, ''), // 数据池名称
    admin: params.admin, // 管理员名称
    streamCode: params.streamCode // 数据池编号（可以用于绑定设备id）(string)
  })
}

export function streamUpdate(params) { // 数据池修改
  const apiKey = params.apiKey
  const query = `?ApiKey=${apiKey}`
  const data = {
    admin: params.admin,
    apiKey: params.apiKey,
    streamCode: params.streamCode // 数据池编号（可以用于绑定设备id）(string
  }
  if (params.name) data.name = params.name.replace(/\s*/g, '') // 数据池名称
  return JsonPost(`tolink/stream/update${query}`, data)
}

export function streamDelete(params) { // 数据池删除
  const Name = params.Name.replace(/\s*/g, '') // 数据池名称
  return JsonPost(`tolink/stream/delete?Name=${Name}`, {})
}

export function getStreamLabel(params) { // 获取数据池的标签
  const Name = params.Name.replace(/\s*/g, '') // 流名称
  let Function = params.Function // 功能指令 可取值：'intact'代表获取详细列表、'list'代表单个查找
  if (!Function) {
    Function = 'intact'
  }
  const query = `Name=${Name}&Function=${Function}`
  return JsonPost(`tolink/stream/getlabel?${query}`, {})
}

export function setStreamLabel(params) { // 数据池的标签增删改
  const Name = params.Name.replace(/\s*/g, '') // 流名称
  const Function = params.Function /* Function的值为'create'时为创建属性，为'delete'时为删除属性，为'update'时为修改属性 */

  let data = []
  if (Function === 'create') {
    params.list.forEach(item => {
      const formula = item.proceFormula // 处理公式
      const unformula = item.inverTreatFormula // 逆处理公式
      if (!(Function === 'delete')) {
        if (!(formula.indexOf(' ') === -1 && unformula.indexOf(' ') === -1)) {
          console.log('处理公式和逆处理公式均不能有空格')
          return
        }
      }
      data.push({
        streamId: item.streamId, // 数据池id
        name: item.name.replace(/\s*/g, ''), // 标签名称
        gatewayName: item.gatewayName.replace(/\s*/g, ''), // 网关名称
        gatewayId: item.gatewayId, // 网关ID
        dataName: item.dataName.replace(/\s*/g, ''), // 数据名称
        dataId: item.dataId, // 数据ID
        dataType: item.dataType, // 上报类型(string)
        formula,
        unformula
      })
    })
  } else if (Function === 'delete') {
    data = [{
      labelId: params.labelId // 标签ID
    }]
  } else if (Function === 'update') {
    const formula = params.proceFormula // 处理公式
    const unformula = params.inverTreatFormula // 逆处理公式
    if (!(Function === 'delete')) {
      if (!(formula.indexOf(' ') === -1 && unformula.indexOf(' ') === -1)) {
        console.log('处理公式和逆处理公式均不能有空格')
        return
      }
    }
    data = [{
      labelId: params.labelId, // 标签ID
      streamId: params.streamId, // 数据池id
      name: params.name.replace(/\s*/g, ''), // 标签名称
      gatewayName: params.gatewayName.replace(/\s*/g, ''), // 网关名称
      gatewayId: params.gatewayId, // 网关ID
      dataName: params.dataName.replace(/\s*/g, ''), // 数据名称
      dataId: params.dataId, // 数据ID
      dataType: params.dataType, // 上报类型(string)
      formula,
      unformula
    }]
  }
  return JsonPost(`tolink/stream/label?Name=${Name}&Function=${Function}`, data)
}

export function streamImportexcel(params) { // 数据池表格导入
  const Name = params.Name // 数据池名称
  const data = new FormData()
  data.append('file', params.fileList[0])
  return JsonPost(`go/stream/importexcel?Name=${Name}`, data)
}

export function streamExportExcel(params) { // 数据池表格导出
  const Name = params.Name // 数据池名称
  return JsonPost(`go/stream/exportexcel?Name=${Name}`)
}
