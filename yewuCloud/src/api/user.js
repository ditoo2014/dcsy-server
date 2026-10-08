import { formDataPost, JsonPost } from '@/api/api.js'
import { objectSort } from '@/utils/utils.js'

export function sendLoginSmsCode(params) { // 获取登录短信验证码
  const formData = new FormData()
  formData.append('username', params.username)
  formData.append('password', params.password)
  return formDataPost(`auth/smscode`, formData)
}

export function login(params) { // 登录
  return JsonPost(`auth/login`, {
    username: params.username,
    password: params.password,
    smsCodeId: params.smsCodeId,
    smsCode: params.smsCode
  })
}

export function findSelf() { // 查询用户详细信息
  return JsonPost(`auth/info`, {})
}

export function getUserList(params) { // 查询用户列表
  const page = params.page
  const limit = params.limit
  const corpId = params.userGroupId
  const userSearchName = params.userSearchName
  let query = `?pageNum=${page}&pageSize=${limit}`
  if (corpId) query += `&corpId=${corpId}`
  if (userSearchName) query += `&query=${userSearchName}`
  const filterFunc = params.filterFunc ? params.filterFunc : 'BinC' // ConE:通过设备关注用户查询, BinC:通过站址绑定部门查询
  query += `&filterFunc=${filterFunc}`
  return JsonPost(`auth/findpage${query}`, {})
}

export function addUser(params) { // 添加用户
  return JsonPost(`auth/register`, {
    username: params.username, // 用户名 String*
    password: params.password, // 用户密码 String*(公钥加密密码)
    truename: params.truename, // 真实姓名 String
    phone: params.phone, // 电话号码 String
    email: params.email, // 邮件地址 String
    parentId: params.parentId, // 创建人ID int
    paleName: params.paleName, // 平台名称 String
    paleType: params.paleType, // 平台类别 String
    baned: params.baned, // 是否禁用 int （0：否，1：是）
    corpIds: params.corpIds // 用户部门 int[]数组
  })
}

export function updateUser(params) { // 修改用户
  return JsonPost(`auth/update`, {
    id: params.id,
    username: params.username, // 用户名 String*
    truename: params.truename, // 真实姓名 String
    phone: params.phone, // 电话号码 String
    email: params.email, // 邮件地址 String
    parentId: params.parentId, // 创建人ID int
    paleName: params.paleName, // 平台名称 String
    paleType: params.paleType, // 平台类别 String
    baned: params.baned, // 是否禁用 int （0：否，1：是）
    corpIds: params.corpIds // 用户部门 int[]数组
  })
}

export function deleteUser(params) { // 删除用户
  return JsonPost(`auth/delete`, {
    id: params.id
  })
}

export function getRoleList() { // 查询角色列表
  return JsonPost(`auth/business/role/list`, {})
}

export function getTolinkRoleList() { // 查询采集云角色列表
  return JsonPost(`auth/tolink/role/list`, {})
}

export function addRole(params) { // 添加角色
  return JsonPost(`auth/business/role/add`, {
    roleName: params.roleName,
    roleDesc: params.roleDesc, // 描述
    platId: 1 // 平台id （采集云为2，业务云为1）
  })
}

export function updateRole(params) { // 修改角色
  return JsonPost(`auth/business/role/update`, {
    id: params.id,
    roleName: params.roleName,
    roleDesc: params.roleDesc, // 描述
    platId: 1 // 平台id （采集云为2，业务云为1）
  })
}

export function deleteRole(params) { // 删除角色
  return JsonPost(`auth/business/role/delete`, {
    id: params.id
  })
}

function fetchPermissionList() { // 全部权限列表
  return JsonPost(`auth/business/permission/list`, {})
}

export function permissionList(params) {
  return fetchPermissionList().then(res => {
    if (!params || !res.return || !res.return.data) {
      return res
    }
    if ('action' in params) {
      return {
        ...res,
        page: {
          list: res.return.data.filter(p => p.type === 2)
        }
      }
    }
    return res
  })
}

export function rolePermiList(params) { // 查询角色绑定的权限
  return JsonPost(`auth/business/rolepermission/list`, {
    roleId: params.roleId // 角色id
  })
}

export function batchaddRolePermi(param) { // 批量绑定角色权限
  const list = []
  param.list.forEach(item => {
    list.push({
      roleId: item.roleId, // 角色id
      permissionId: item.permissionId // 权限的id
    })
  })
  return JsonPost(`auth/business/rolepermission/batchadd`, list)
}

export function batchdeleteRolePermi(param) { // 批量解绑角色权限
  const list = []
  param.list.forEach(item => {
    list.push({
      roleId: item.roleId, // 角色id
      permissionId: item.permissionId // 权限的id
    })
  })
  return JsonPost(`auth/business/rolepermission/batchdeletelist`, list)
}

export async function corpList() { // 查询部门列表
  const res = await JsonPost(`auth/business/corp/list`, {})
  const list = res && res.return && res.return.data
  if (list) {
    res.return.data = objectSort(list, 'id')
  }
  return res
}

export function addCorp(params) { // 添加部门
  return JsonPost(`auth/business/corp/add`, {
    level: params.level,
    parentId: params.parentId,
    corpName: params.corpName,
    platId: 0
  })
}

export function updateCorp(params) { // 修改部门
  return JsonPost(`auth/business/corp/update`, {
    id: params.id,
    level: params.level,
    parentId: params.parentId,
    corpName: params.corpName,
    platId: 0
  })
}

export function deleteCorp(params) { // 删除部门
  return JsonPost(`auth/business/corp/delete`, {
    id: params.id
  })
}

export function userBindRoleList() { // 用户绑定角色查询
  return JsonPost(`auth/business/userrole/list`, {})
}

export function userTolinkBindRoleList() { // 采集云用户绑定角色查询
  return JsonPost(`auth/tolink/userrole/list`, {})
}

export function userBindingRole(params) { // 用户绑定角色
  return JsonPost(`auth/business/userrole/add`, {
    userId: params.userId,
    roleId: params.roleId
  })
}

export function userUnBindRole(params) { // 用户解绑角色
  return JsonPost(`auth/business/userrole/delete`, {
    id: params.id // 绑定的id
  })
}

export function getUserPermissions() { // 查询当前用户权限
  return JsonPost(`auth/business/user/permissions`, {})
}

export function modifyPassword(params) { // 修改密码
  return JsonPost(`auth/changepassword`, {
    password: params.password, // 旧密码
    newPassword: params.newPassword // 新密码
  })
}

export function resetPassword(params) { // 重置密码
  return JsonPost(`auth/resetuserpassword`, {
    newPassword: params.newPassword, // 新密码
    managerPassword: params.managerPassword, // 管理员密码
    usersEntity: {
      id: params.userEntityId // 被修改人id
    }
  })
}

/** @deprecated 使用 updateRole */
export const modifyRole = updateRole

function toSecondTimestamp(value) {
  if (!value) return ''
  if (value instanceof Date) {
    return Math.floor(value.getTime() / 1000)
  }
  if (typeof value === 'number') {
    return Math.floor(value > 9999999999 ? value / 1000 : value)
  }
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? '' : Math.floor(parsed / 1000)
}

export function syslog(params = {}) { // 业务云系统操作日志
  const page = params.page || params.pageNum || 1
  const limit = params.limit || params.pageSize || 10
  const dateRange = params.dateRange || []
  const startTime = params.startTime || toSecondTimestamp(dateRange[0])
  const endTime = params.endTime || toSecondTimestamp(dateRange[1])
  let query = `?pageNum=${page}&pageSize=${limit}`
  if (startTime) query += `&startTime=${startTime}`
  if (endTime) query += `&endTime=${endTime}`

  const data = {
    platId: 1,
    username: params.username ? String(params.username).trim() : '',
    trueName: params.trueName ? String(params.trueName).trim() : ''
  }

  return JsonPost(`auth/business/syslog/list${query}`, data)
}

export async function addUserConcernEquipment(params) { // 用户关注设备授权
  const res = await JsonPost(`auth/business/userconcernequipment/add`, {
    grantedUser: params.grantedUser,
    equipmentIds: params.equipmentIds
  })
  return { ...res, ...(res.return || {}) }
}

export async function deleteUserConcernEquipment(params) { // 取消用户关注设备
  const res = await JsonPost(`auth/business/userconcernequipment/delete`, {
    grantedUser: params.grantedUser,
    equipmentIds: params.equipmentIds
  })
  return { ...res, ...(res.return || {}) }
}

export async function findUserConcernEquipment(params) { // 查询用户关注设备
  const res = await JsonPost(`auth/business/userconcernequipment/find`, {
    grantedUser: params.grantedUser
  })
  return { ...res, ...(res.return || {}) }
}

export async function addUserControlEquipment(params) { // 用户控制设备授权
  const res = await JsonPost(`auth/business/usercontrolequipment/add`, {
    grantedUser: params.grantedUser,
    equipmentIds: params.equipmentIds
  })
  return { ...res, ...(res.return || {}) }
}

export async function deleteUserControlEquipment(params) { // 取消用户控制设备
  const res = await JsonPost(`auth/business/usercontrolequipment/delete`, {
    grantedUser: params.grantedUser,
    equipmentIds: params.equipmentIds
  })
  return { ...res, ...(res.return || {}) }
}

export async function findUserControlEquipment(params) { // 查询用户控制设备
  const res = await JsonPost(`auth/business/usercontrolequipment/find`, {
    grantedUser: params.grantedUser
  })
  return { ...res, ...(res.return || {}) }
}

/**
 * 查询角色已绑定的功能权限（告警中心等旧页面兼容格式）
 */
export async function rolePermis(params) {
  const res = await rolePermiList({ roleId: params.roleId })
  const bound = (res.return && res.return.data) || []
  const allRes = await permissionList()
  const allPermis = (allRes.return && allRes.return.data) || []
  const list = bound.map(item => {
    const perm = allPermis.find(p => p.id === item.permissionId) || {}
    return {
      permission_name: perm.permissionName || '',
      parent_id: perm.parentId,
      type: perm.type,
      permissionId: item.permissionId
    }
  }).filter(item => params.type == null || item.type === params.type)
  return { list }
}
