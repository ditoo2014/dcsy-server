import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { transformSync } from 'esbuild'
import { parse, compileTemplate } from '@vue/compiler-sfc'

// 内存执行实际状态仓库、导航守卫及请求拦截器，不访问真实账号、短信或生产服务。
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const read = name => fs.readFileSync(path.join(root, name), 'utf8')
function evaluate(source, dependencies = {}, globals = {}) {
  const module = { exports: {}}
  const code = transformSync(source, { format: 'cjs', define: { 'import.meta.env': '{}' }}).code
  const require = name => {
    if (Object.hasOwn(dependencies, name)) return dependencies[name]
    throw new Error(`未声明的测试依赖：${name}`)
  }
  new Function('module', 'exports', 'require', ...Object.keys(globals), code)(module, module.exports, require, ...Object.values(globals))
  return module.exports
}
const passwordState = evaluate(read('src/utils/password-change-state.js'))
const restricted = reason => ({ code: 200, return: { user: { id: 1, isadmin: 1 }, passwordChangeRequired: true, passwordChangeReason: reason }})
const expired = { code: 403, msg: '请先修改密码', return: { ...restricted('quarter_expired').return, errorCode: passwordState.PASSWORD_CHANGE_REQUIRED }}
const normal = admin => ({ code: 200, return: { user: { id: 1, isadmin: admin }, passwordChangeRequired: false, passwordExpiresAt: '2027-01-01T00:00:00+08:00' }})
let checks = 0
async function check(name, run) {
  await run()
  checks++
  console.log(`通过：${name}`)
}
const flush = () => new Promise(resolve => setImmediate(resolve))

await check('登录状态与专用失败响应分开，正常登录不会被当作403', () => {
  assert.equal(passwordState.isPasswordChangeRequired(restricted('password_unset')), true)
  assert.equal(passwordState.isPasswordChangeFailure(restricted('password_unset')), false)
  assert.equal(passwordState.isPasswordChangeFailure(expired), true)
  assert.equal(passwordState.isPasswordChangeFailure({ code: 403, msg: '无权限' }), false)
  assert.match(passwordState.passwordChangeMessage('future_time'), /更新时间异常/)
  assert.equal(passwordState.requestBearerToken({ headers: { authorization: 'Bearer 示例令牌' }}), '示例令牌')
})

function userHarness() {
  let token = '测试令牌'
  let info = normal(0)
  let permissionFailure
  let infoHandler
  let permissionHandler
  let clearFailure
  const calls = { permissions: 0, reset: 0, access: 0, collectionLogout: 0, tags: 0 }
  const storage = new Map()
  const auth = {
    getToken: () => token,
    setToken: value => { token = value },
    removeToken: () => { token = '' },
    setUserInfoStr: value => storage.set('userInfoStr', value),
    setUserId: value => storage.set('userId', value),
    setPageList: value => storage.set('pagelist', value)
  }
  for (const name of ['Username', 'UserInfoStr', 'UserId', 'RoleId', 'PageList', 'CjyToken']) {
    auth[`remove${name}`] = () => storage.delete(name)
  }
  const module = evaluate(read('src/store/modules/user.js'), {
    '@/api/user.js': {
      login: async() => ({ ...info, return: { ...info.return, token: '新令牌' }}),
      findSelf: async() => infoHandler ? infoHandler() : info,
      getUserPermissions: async() => {
        calls.permissions++
        if (permissionHandler) return permissionHandler()
        if (permissionFailure) throw permissionFailure
        return { code: 200, return: { data: [] }}
      }
    },
    '@/utils/auth.js': auth,
    '@/router': { resetRouter: () => calls.reset++, asyncRoutes: [{ path: '/业务' }] },
    '@/utils/utils.js': { unique: value => value, objectSort: value => value },
    '@/utils/build-permission-routes.js': { buildRoutesFromPermissions: () => [{ path: '/业务' }] },
    '@/utils/element-shim': { Message: () => {} },
    '@/utils/session-logout': { notifyBackendLogout() {} },
    '@/config.js': { default: {}},
    '@/constants/platform': { PLATFORM: {}},
    '@/utils/password-change-state.js': passwordState,
    '@/collection/stores/auth.js': { useAuthStore: () => ({ clearAccess: () => calls.access++, logout: () => calls.collectionLogout++ }) },
    '@/collection/stores/tagsView.js': { useTagsViewStore: () => ({ clearVisitedViews: () => calls.tags++ }) }
  }, { localStorage: { removeItem: key => storage.delete(key) }}).default
  const state = structuredClone(module.state)
  const context = {
    state,
    commit: (name, payload) => module.mutations[name](state, payload),
    dispatch: async(name, payload, options) => {
      if (options?.root) {
        if (clearFailure) throw clearFailure
        return
      }
      return module.actions[name](context, payload)
    }
  }
  return {
    state, calls, context, token: () => token,
    setToken: value => { token = value },
    setInfo: value => { info = value },
    setInfoHandler: value => { infoHandler = value },
    setPermissionHandler: value => { permissionHandler = value },
    setClearFailure: value => { clearFailure = value },
    setPermissionFailure: value => { permissionFailure = value }
  }
}

await check('普通用户与超级管理员受限时均不加载权限，令牌保留供改密', async() => {
  for (const admin of [0, 1]) {
    const h = userHarness()
    const info = restricted('password_unset')
    info.return.user.isadmin = admin
    h.setInfo(info)
    const result = await h.context.dispatch('getInfo')
    assert.equal(result.passwordChangeRequired, true)
    assert.deepEqual(result.roles, [])
    assert.equal(h.calls.permissions, 0)
    assert.equal(h.token(), '测试令牌')
    assert.equal(h.calls.reset, 1)
    assert.equal(h.calls.access, 1)
    assert.equal(h.calls.tags, 1)
  }
})

await check('正常用户权限流程保留，管理员权限请求跨季度失败不再兜底放行', async() => {
  const h = userHarness()
  const result = await h.context.dispatch('getInfo')
  assert.deepEqual(result.roles, ['manager'])
  assert.equal(h.calls.permissions, 1)
  h.setInfo(normal(1))
  h.setPermissionFailure(Object.assign(new Error('季度到期'), { errorCode: passwordState.PASSWORD_CHANGE_REQUIRED, responseData: expired }))
  const denied = await h.context.dispatch('getInfo')
  assert.equal(denied.passwordChangeRequired, true)
  assert.deepEqual(denied.routers, [])
})

await check('重新登录与退出多轮复位密码状态及采集权限，不残留旧菜单', async() => {
  const h = userHarness()
  for (const reason of ['quarter_expired', 'admin_reset', 'password_unset']) {
    h.setInfo(restricted(reason))
    await h.context.dispatch('login', {})
    assert.equal(h.state.passwordChangeReason, reason)
    assert.equal(h.token(), '新令牌')
    await h.context.dispatch('sessionTimeout')
    assert.equal(h.token(), '')
    assert.equal(h.state.passwordChangeRequired, false)
    assert.equal(h.state.passwordChangeReason, '')
    assert.deepEqual(h.state.roles, [])
  }
  h.setInfo(normal(0))
  await h.context.dispatch('login', {})
  assert.equal((await h.context.dispatch('getInfo')).roles.length, 1)
  assert.equal(h.calls.collectionLogout, 3)
})

await check('旧用户info与权限403晚返回不限制新账号，清理异常正确结束登录请求', async() => {
  for (const responseType of ['info', 'permission']) {
    const h = userHarness()
    let fail
    const pending = new Promise((resolve, reject) => { fail = reject })
    if (responseType === 'info') h.setInfoHandler(() => pending)
    else h.setPermissionHandler(() => pending)
    const request = h.context.dispatch('getInfo')
    await flush()
    h.setToken('新用户令牌')
    fail(Object.assign(new Error('旧用户到期'), { errorCode: passwordState.PASSWORD_CHANGE_REQUIRED, responseData: expired }))
    await assert.rejects(request, { errorCode: 'SESSION_CHANGED' })
    assert.equal(h.state.passwordChangeRequired, false)
    assert.equal(h.calls.reset, 0)
    assert.equal(h.token(), '新用户令牌')
  }
  const h = userHarness()
  h.setClearFailure(new Error('模拟本地清理错误'))
  await assert.rejects(h.context.dispatch('login', {}), /模拟本地清理错误/)
  assert.equal(h.token(), '测试令牌')
})

await check('并发403仅清理跳转一次，忽略旧令牌，下一轮仍能强制改密', async() => {
  let token = '第一轮令牌'
  let release
  let clears = 0
  const redirects = []
  const password = evaluate(read('src/utils/password-change.js'), {
    '@/store': { dispatch: () => { clears++; return new Promise(resolve => { release = resolve }) } },
    '@/router': { currentRoute: { value: { path: '/collection' }}, replace: async to => redirects.push(to) },
    '@/utils/auth.js': { getToken: () => token },
    '@/utils/password-change-state.js': passwordState
  })
  const first = password.requirePasswordChange(expired, token)
  assert.equal(first, password.requirePasswordChange(expired, token))
  await flush()
  assert.equal(clears, 1)
  release()
  await first
  assert.deepEqual(redirects, [{ path: '/modifyPas' }])
  token = '第二轮令牌'
  await password.requirePasswordChange(expired, '第一轮令牌')
  assert.equal(clears, 1)
  const second = password.requirePasswordChange(expired, token)
  await flush()
  release()
  await second
  assert.equal(clears, 2)
  assert.equal(redirects.length, 2)
})

function interceptorHarness(file) {
  let handlers
  let expires = 0
  const redirects = []
  const client = { interceptors: { request: { use() {} }, response: { use: (...args) => { handlers = args } }}}
  const api = evaluate(read(file), {
    axios: { create: () => client },
    '@/utils/element-shim': { Message() {} },
    '@/utils/session-expiration': { expireSession: () => expires++ },
    '@/utils/password-change.js': { requirePasswordChange: (...args) => { redirects.push(args); return Promise.resolve() } },
    '@/utils/password-change-state.js': passwordState,
    '@/utils/auth.js': { getToken: () => '当前用户令牌' },
    '../utils/mainLoginRedirect': { redirectToMainLogin: () => expires++ }
  }, { Blob })
  if (api.attachAuthFailureRedirectInterceptor) api.attachAuthFailureRedirectInterceptor(client)
  return { handlers, redirects, expires: () => expires }
}

await check('业务与采集403拦截保留令牌，成功info状态正常返回，不触发登录过期', async() => {
  for (const file of ['src/utils/request.js', 'src/collection/api/authFailureRedirect.js']) {
    const h = interceptorHarness(file)
    const response = { data: restricted('quarter_expired'), config: { headers: { Authorization: 'Bearer 测试令牌' }}}
    const success = await h.handlers[0](response)
    assert.ok(success)
    assert.equal(h.redirects.length, 0)
    await assert.rejects(h.handlers[1]({ config: response.config, response: { status: 403, data: expired }}), error => {
      assert.equal(error.errorCode, passwordState.PASSWORD_CHANGE_REQUIRED)
      assert.equal(error.responseData, expired)
      return true
    })
    assert.equal(h.expires(), 0)
    assert.equal(h.redirects.length, 1)
    assert.equal(h.redirects[0][1], '测试令牌')
  }
  const h = interceptorHarness('src/utils/request.js')
  const excel = new Blob(['示例Excel内容'])
  excel.text = () => assert.fail('正常下载不能被作为JSON读取')
  const download = { data: excel, config: { responseType: 'blob' }, headers: { 'content-type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }}
  assert.equal(await h.handlers[0](download), download)
  await assert.rejects(h.handlers[1]({ config: { headers: { Authorization: 'Bearer 下载令牌' }}, response: { status: 403, data: new Blob([JSON.stringify(expired)]) }}), { errorCode: passwordState.PASSWORD_CHANGE_REQUIRED })
  assert.equal(h.redirects[0][1], '下载令牌')
  for (const file of ['src/utils/request.js', 'src/collection/api/authFailureRedirect.js']) {
    const oldRequest = interceptorHarness(file)
    await assert.rejects(oldRequest.handlers[1]({ config: { headers: { Authorization: 'Bearer 旧用户令牌' }}, response: { status: 401, data: { code: 401 }}}))
    assert.equal(oldRequest.expires(), 0, '旧401不能退出新用户会话')
  }
})

await check('刷新及跨季度导航优先检查改密，采集权限和业务角色均不能绕过', async() => {
  let guard
  let required = true
  let hasRoles = true
  let collectionCalls = 0
  let businessCalls = 0
  let token = '测试令牌'
  let checkStatus
  let resets = 0
  const routes = { beforeEach: callback => { guard = callback }, afterEach() {} }
  evaluate(read('src/permission.js'), {
    './router': routes,
    './store': {
      getters: { get roles() { return hasRoles ? ['manager'] : [] } }, state: { permission: { addRoutes: [] }},
      dispatch: async action => {
        if (action === 'user/checkPasswordChange') return checkStatus ? checkStatus() : { passwordChangeRequired: required }
        if (action === 'user/resetToken') resets++
        businessCalls++
        return { roles: ['manager'], routers: [] }
      }
    },
    '@/utils/element-shim': { Message: { error() {} }},
    nprogress: { configure() {}, start() {}, done() {} },
    'nprogress/nprogress.css': {},
    '@/utils/auth.js': { getToken: () => token },
    '@/utils/get-page-title': () => '',
    '@/config.js': { default: {}},
    '@/constants/platform': { PLATFORM: {}},
    '@/utils/resolve-accessible-route.js': { normalizeLegacyRoutePath: value => value, isRouteAccessible: () => true },
    '@/utils/banner-route.js': { resolveBannerIdFromPath: () => '' },
    '@/collection/utils/authAccess.js': { ensureTolinkAccess: async() => collectionCalls++ },
    '@/collection/stores/auth.js': { useAuthStore: () => ({ hasAnyPermission: () => true }) },
    '@/utils/dynamic-route-state.js': { isDynamicRoutesReady: () => false, resetDynamicRoutesReady() {}, markDynamicRoutesReady() {} },
    '@/utils/password-change-state.js': passwordState
  }, { window: {}, document: {}, localStorage: { setItem() {} }})
  const navigate = async path => {
    const targets = []
    await guard({ path, fullPath: path, query: {}, meta: {}, matched: [{ meta: { collectionCloud: path.startsWith('/collection') }}] }, {}, target => targets.push(target))
    assert.equal(targets.length, 1)
    return targets[0]
  }
  for (hasRoles of [true, false]) {
    for (const route of ['/collection', '/collection/ttdc/网关', '/home', '/jkztjc']) {
      assert.deepEqual(await navigate(route), { path: '/modifyPas', replace: true })
    }
    assert.equal(await navigate('/modifyPas'), undefined)
  }
  assert.equal(collectionCalls, 0)
  assert.equal(businessCalls, 0)
  required = false
  hasRoles = true
  assert.equal(await navigate('/home'), undefined)
  assert.equal(await navigate('/collection'), undefined)
  assert.equal(collectionCalls, 1)
  for (const error of [new Error('连接失败'), { response: { status: 503 }, message: '数据库暂时不可用' }]) {
    checkStatus = async() => { throw error }
    assert.equal(await navigate('/home'), false)
    assert.equal(resets, 0)
    assert.equal(token, '测试令牌')
  }
  // 同一浏览器在等待旧info时重新登录，旧403及旧401都不能修改新会话。
  for (const error of [{ errorCode: passwordState.PASSWORD_CHANGE_REQUIRED, responseData: expired }, { response: { status: 401 }}]) {
    checkStatus = async() => { token = '新用户令牌'; throw error }
    token = '测试令牌'
    assert.equal(await navigate('/home'), false)
    assert.equal(resets, 0)
    assert.equal(token, '新用户令牌')
  }
  checkStatus = undefined
  token = ''
  assert.equal(await navigate('/modifyPas'), '/login?redirect=/modifyPas')
})

await check('改密页面沿用原表单，禁止相同密码，成功后清会话，受限返回执行退出', async() => {
  const file = 'src/views/personMana/modifyPas.vue'
  const { descriptor, errors } = parse(read(file), { filename: file })
  assert.deepEqual(errors, [])
  assert.deepEqual(compileTemplate({ source: descriptor.template.content, filename: file, id: 'password-test' }).errors, [])
  const exits = []
  let policyChecks = 0
  const submitted = []
  const options = evaluate(descriptor.script.content, {
    '@/utils/utils.js': { goBack: () => exits.push('返回'), getRSApassword: value => `加密:${value}` },
    '@/utils/auth.js': { getUsername: () => '测试账号', getToken: () => '测试令牌' },
    '@/api/user.js': { modifyPassword: async form => { submitted.push(form); return { code: 200 } } },
    '@/utils/session-expiration': { expireSession: async options => exits.push(options) },
    '@/utils/password-change-state.js': passwordState,
    '@/utils/password-policy.js': { createPasswordPolicyValidator: () => (rule, value, callback) => { policyChecks++; callback() } }
  }).default
  const vm = { $store: { state: { user: { passwordChangeRequired: true, passwordChangeReason: 'quarter_expired' }}}, $notify() {}, $refs: { dataForm: { validate: callback => callback(true) }}}
  Object.assign(vm, options.data.call(vm))
  for (const [name, fn] of Object.entries(options.methods)) vm[name] = fn.bind(vm)
  vm.forcePasswordChange = options.computed.forcePasswordChange.call(vm)
  vm.form.password = '旧密码'
  let validationError
  vm.rules.newPassword[0].validator({}, '旧密码', error => { validationError = error })
  assert.match(validationError.message, /不能与旧密码相同/)
  assert.equal(policyChecks, 0)
  vm.form.newPassword = '新密码'
  vm.rules.newPassword[0].validator({}, '新密码', assert.ifError)
  assert.equal(policyChecks, 1)
  vm.setData()
  await flush()
  assert.equal(submitted[0].password, '加密:旧密码')
  assert.equal(submitted[0].newPassword, '加密:新密码')
  assert.deepEqual(exits, [{ redirectPath: '/' }])
  assert.equal(vm.submitting, false)
  await vm.backPage()
  assert.deepEqual(exits.at(-1), { notifyBackend: true, redirectPath: '/' })
  vm.forcePasswordChange = false
  await vm.backPage()
  assert.equal(exits.at(-1), '返回')
})

console.log(`季度强制改密专项：${checks} 项通过。`)
