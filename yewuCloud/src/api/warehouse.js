import { JsonPost } from '@/api/api.js'

export function saveWarehouse(params) { // 添加库房
  return JsonPost(`business/warehouse/add`, {
    name: params.name, // 库房名称
    address: params.addr, // 库房地址
    managerName: params.managerName, // 库房管理员
    managerUserId: params.managerUserId, // 库房管理员id
    managerPhone: params.managerPhone, // 管理员电话
    warehouseCode: params.warehouseCode, // 编码
    corpId: params.corpId, // 部门id
    createUserId: params.createUserId, // 创建用户id
    warehouseDetails: { // 库房具体地理位置等其他信息
      longitude: params.longitude, // 经度
      latitude: params.latitude, // 纬度
      provinceName: params.provinceName, // 省份
      cityName: params.cityName, // 城市
      addr: params.addr // 详细地址
    }
  })
}

export function findWarehouseList(params) { // 查询库房列表
  const data = {}
  if (params) {
    if (params.name && params.name !== '') data.name = params.name
    if ((params.corpId && params.corpId !== '') || params.corpId === 0) data['corpId'] = params.corpId // 部门id
  }
  const pageNum = params.pageNum
  const pageSize = params.pageSize
  const query = `?pageNum=${pageNum}&pageSize=${pageSize}`
  return JsonPost(`business/warehouse/find${query}`, data)
}

export function deleteWarehouse(params) { // 删除库房
  return JsonPost(`business/warehouse/delete`, {
    id: params.id
  })
}

export function modifyWarehouse(params) { // 修改库房
  return JsonPost(`business/warehouse/update`, {
    id: params.id,
    name: params.name, // 库房名称
    address: params.addr, // 库房地址
    managerName: params.managerName, // 库房管理员
    managerUserId: params.managerUserId, // 库房管理员id
    managerPhone: params.managerPhone, // 管理员电话
    warehouseCode: params.warehouseCode, // 编码
    corpId: params.corpId, // 部门id
    createUserId: params.createUserId, // 创建用户id
    warehouseDetails: { // 库房具体地理位置等其他信息
      longitude: params.longitude, // 经度
      latitude: params.latitude, // 纬度
      provinceName: params.provinceName, // 省份
      cityName: params.cityName, // 城市
      addr: params.addr // 详细地址
    }
  })
}
