const getters = {
  sidebar: state => state.app.sidebar,
  permission_routes: state => state.permission.routes,
  errorLogs: state => state.errorLog.logs,
  device: state => state.app.device,
  token: state => state.user.token,
  roleType: state => state.user.roleType, // 用户所属角色类型
  roles: state => state.user.roles, // 用户所属角色
  avatar: state => state.user.avatar, // 用户头像图片地址
  codeName: state => state.user.codeName, // 账号的企业代码名称或厂家代码名称
  manufactor: state => state.user.manufactor // 当登录账号为经销商创建厂家时，需要使用
}
export default getters
