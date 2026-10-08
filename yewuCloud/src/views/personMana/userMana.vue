<template>
  <div>
    <el-scrollbar class="treeContentCss allheight">
      <MyElTree :navi-name-list="treeList" @click-tree="chooseTargetgroup" />
    </el-scrollbar>
    <div class="app-container">
      <div class="titleCss">{{ targetGroupNowForm.name }}</div>
      <div class="filter-container">
        <el-input v-model="searchMsg" placeholder="请输入姓名、账号或手机号" style="width: 200px;" class="filter-item" @keyup.enter="search" />
        <el-button class="filter-item" style="margin-left: 5px;" type="primary" size="small" @click="search">
          查询
        </el-button>
        <el-button class="filter-item" style="margin-left: 5px;" size="small" @click="resetData">
          重置
        </el-button>
          &nbsp;
        <el-button class="filter-item" style="margin-left: 10px;" type="primary" :icon="$legacyIcon('el-icon-circle-plus-outline')" size="small" @click="add">
          添加用户
        </el-button>
        <el-button class="filter-item" style="margin-left: 10px;" type="primary" :icon="$legacyIcon('el-icon-edit')" size="small" @click="modifyTaoup">
          修改部门名称
        </el-button>
        <el-button class="filter-item" style="margin-left: 10px;" type="primary" :icon="$legacyIcon('el-icon-circle-plus-outline')" size="small" @click="addTaoup">
          新建子部门
        </el-button>
        <el-button
          v-if="targetGroupNowForm.name !== ''"
          class="filter-item"
          style="margin-left: 10px;"
          type="danger"
          :icon="$legacyIcon('el-icon-delete')"
          size="small"
          @click="removeTaoup"
        >
          删除当前部门
        </el-button>
      </div>

      <el-scrollbar class="dataContentCss allheight">
        <!-- border:显示边框,fit:列的宽度自撑开,highlight-current-row:对当前行高亮
        sort-change:当表格排序发生变化时触发,key:当值发生变化时
        selection-change:当复选框的选中状态发生改变时-->
        <el-table
          :key="tableKey"
          height="500"
          :data="tableData"
          border
          fit
          highlight-current-row
          @sort-change="sortChange"
        >
          <el-table-column label="姓名" align="center">
            <template #default="scope">
              <span>{{ scope.row.trueName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="账号" align="center">
            <template #default="scope">
              <span>{{ scope.row.username }}</span>
            </template>
          </el-table-column>
          <el-table-column label="手机号码" align="center">
            <template #default="scope">
              <span>{{ scope.row.phone }}</span>
            </template>
          </el-table-column>
          <el-table-column label="角色" align="center">
            <template #default="scope">
              <span>{{ scope.row.roleNames }}</span>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" align="center" width="160">
            <template #default="scope">
              <span>{{ scope.row.time }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" align="center" class-name="small-padding fixed-width" width="400">
            <template #default="scope">
              <el-button type="primary" size="small" @click="modify(scope.row)">
                修改
              </el-button>
              <el-button type="danger" size="small" @click="remove(scope.row)">
                删除
              </el-button>
              <el-button v-if="userInfo.isadmin === 1" size="small" @click="clickRestPassword(scope.row)">
                重置密码
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <!-- total:总条目数, page:页码,limit:一页-->
        <Pagination
          v-show="total>0"
          v-model:page="page"
          v-model:limit="limit"
          :total="total"
          @pagination="getMsgList"
        />
      </el-scrollbar>

      <!-- 添加修改弹窗 -->
      <elDrawer v-model="showDrawer" :title="textMap[dialogStatus]" width="400">
        <el-form
          v-if="dialogStatus === 'add' || dialogStatus === 'modify'"
          ref="dataForm"
          :rules="rules"
          :model="form"
          size="small"
          label-width="110px"
          :class="{ 'password-policy-form': dialogStatus === 'add' }"
        >
          <el-row>
            <el-col :span="20">
              <el-form-item label="姓名" prop="truename">
                <el-input v-model="form.truename" />
              </el-form-item>
            </el-col>
            <el-col :span="20">
              <el-form-item label="角色">
                <el-select v-model="form.roleIds" filterable multiple>
                  <el-option v-for="item in roleList" :key="item.key" :label="item.name" :value="item.key" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="20">
              <el-form-item label="账号" prop="username">
                <el-input v-model="form.username" :disabled="dialogStatus !== 'add'" />
              </el-form-item>
            </el-col>
            <el-col v-if="dialogStatus === 'add'" :span="20">
              <el-form-item label="密码" prop="password">
                <el-input v-model="form.password" type="password" show-password autocomplete="new-password" @input="clearPasswordPolicyServerError('password')" />
              </el-form-item>
            </el-col>
            <el-col v-if="dialogStatus === 'add'" :span="20">
              <el-form-item label="确认密码" prop="surePass">
                <el-input v-model="form.surePass" type="password" show-password autocomplete="new-password" />
              </el-form-item>
            </el-col>
            <el-col :span="20">
              <el-form-item label="手机号" prop="phone">
                <el-input v-model="form.phone" />
              </el-form-item>
            </el-col>
            <el-col v-if="dialogStatus === 'modify'" :span="20">
              <el-form-item label="部门">
                <MyElTree :navi-name-list="modifyTreeList" @click-tree="changeUserGroup" />
              </el-form-item>
            </el-col>
            <el-col :span="20">
              <el-form-item label="邮箱" prop="email">
                <el-input v-model="form.email" />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
        <el-form
          v-if="dialogStatus === 'newTaoup' || dialogStatus === 'modifyTaoupName'"
          ref="dataForm"
          :model="targetGroupForm"
          size="small"
          label-width="90px"
        >
          <el-row>
            <el-col :span="20">
              <el-form-item label="部门名称" prop="targetGroupName">
                <el-input v-model="targetGroupForm.name" />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
        <el-form
          v-if="dialogStatus === 'restPassword'"
          ref="dataForm"
          :rules="rules"
          :model="form"
          size="small"
          label-width="110px"
          class="password-policy-form"
        >
          <el-row>
            <el-col :span="20">
              <el-form-item label="用户名">
                <span>{{ form.username }}</span>
              </el-form-item>
            </el-col>
            <el-col :span="20">
              <el-form-item label="姓名">
                <span>{{ form.trueName }}</span>
              </el-form-item>
            </el-col>
            <el-col :span="20">
              <el-form-item label="新密码" prop="newPassword">
                <el-input v-model="form.newPassword" type="password" show-password autocomplete="new-password" @input="clearPasswordPolicyServerError('newPassword')" />
              </el-form-item>
            </el-col>
            <el-col :span="20">
              <el-form-item label="确认新密码" prop="sureNewPassword">
                <el-input v-model="form.sureNewPassword" type="password" show-password autocomplete="new-password" />
              </el-form-item>
            </el-col>
            <el-col :span="20">
              <el-form-item label="管理员密码" prop="managerPassword">
                <el-input v-model="form.managerPassword" type="password" show-password autocomplete="current-password" />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
        <div class="fixedBottomCss">
          <el-button @click="showDrawer = false">
            取消
          </el-button>
          <el-button v-loading="setMsgLoading" type="primary" @click="setData">
            确定
          </el-button>
        </div>
      </elDrawer>
    </div>

  </div>
</template>

<script>
import { Message } from '@/utils/element-shim'
import Pagination from '@/components/Pagination' // 分页组件
import MyElTree from '@/components/a_myComp/elTree/MyElTree.vue' // 左侧导航
import { checkPhone } from '@/utils/validate.js'
import { parseUserInfo } from '@/utils/safe-json.js'
import {
  createPasswordPolicyValidator,
  PASSWORD_POLICY_ERROR_CODE,
  PASSWORD_POLICY_MESSAGE
} from '@/utils/password-policy.js'
import { meAlert, formatDate, getRSApassword, objectSort } from '@/utils/utils.js'
import {
  getUserList,
  getRoleList,
  getTolinkRoleList,
  addUser,
  updateUser,
  deleteUser,
  corpList,
  addCorp,
  updateCorp,
  deleteCorp,
  addUserControlEquipment,
  deleteUserControlEquipment,
  addUserConcernEquipment,
  deleteUserConcernEquipment,
  userBindRoleList,
  userTolinkBindRoleList,
  userBindingRole,
  userUnBindRole,
  resetPassword
} from '@/api/user.js'
import lodash from 'lodash'

export default {
  components: {
    Pagination,
    MyElTree
  },
  data() {
    const validateUsername = (rule, value, callback) => {
      if (value.length < 6) {
        callback(new Error('账号长度不能小于6位'))
      } else {
        callback()
      }
    }
    const validatePass = createPasswordPolicyValidator(
      () => this.form,
      value => this.getPasswordPolicyServerMessage('password', value)
    )
    const validateSurePass = (rule, value, callback) => {
      if (value !== this.form.password) {
        callback(new Error('确认密码错误'))
      } else {
        callback()
      }
    }
    const validatePhone = (rule, value, callback) => {
      if (checkPhone(value)) {
        callback()
      } else {
        callback(new Error('请输入正确的电话号码'))
      }
    }
    const validateNewPassword = createPasswordPolicyValidator(
      () => this.form,
      value => this.getPasswordPolicyServerMessage('newPassword', value)
    )
    const validateSureNewPassword = (rule, value, callback) => {
      if (this.form.newPassword === value) {
        callback()
      } else {
        callback(new Error('与新密码不一致'))
      }
    }
    return {
      userInfo: parseUserInfo({}),

      tableKey: 1, // 参数值变化，表格对应改变

      total: 0, // 数据总条数
      limit: 10, // 每次加载数据条数
      page: 1, // 页码

      searchMsg: '', // 查询内容

      showDrawer: false, // 抽屉是否显示
      passwordPolicyServerValidation: {
        fieldName: '',
        value: '',
        message: ''
      },
      textMap: { // 表单弹窗标题
        add: '添加用户',
        modify: '修改用户',
        newTaoup: '新建子部门',
        modifyTaoupName: '修改部门名称',
        restPassword: '重置密码'
      },
      dialogStatus: '', // 表单状态值为'add'或'modify' ，与 textMap 对应

      dialogEquipAuth: false, // 设备授权是否显示

      rules: {
        truename: [{ required: true, trigger: 'blur', message: '姓名不能为空' }],
        username: [{ required: true, trigger: 'blur', validator: validateUsername }],
        password: [{ required: true, trigger: 'blur', validator: validatePass }],
        surePass: [{ required: true, trigger: 'blur', validator: validateSurePass }],
        phone: [{ required: true, trigger: 'blur', validator: validatePhone }],
        email: [{ required: true, trigger: 'blur', message: '邮箱不能为空' }],
        newPassword: [{ required: true, trigger: 'blur', validator: validateNewPassword }],
        sureNewPassword: [{ required: true, trigger: 'blur', validator: validateSureNewPassword }],
        managerPassword: [{ required: true, trigger: 'blur', message: '管理员密码不能为空' }]
      },

      tableData: [],
      form: {
        username: '', // 用户名 String*
        password: '', // 用户密码 String*(公钥加密密码)
        truename: '', // 真实姓名 String
        phone: '', // 电话号码 String
        email: '', // 邮件地址 String
        parentId: '', // 创建人ID int
        paleName: '', // 平台名称 String
        paleType: '', // 平台类别 String
        baned: 0, // 是否禁用 int （0：否，1：是）

        roleIds: [], // 角色id数组
        corpIds: [] // 部门id数组
      },

      targetGroupNowForm: { // 当前部门导航树信息
        level: 0, // 目录级别
        parentId: 0, // 父id
        id: 1, // 自己的id
        name: '' // 名称
      },
      treeList: [ // 部门列表
        // { level: 0, parentId: 0, id: 1, name: '企业' }
      ],
      targetGroupForm: { // 预添加或修改导航树信号
        level: 0, // 目录级别
        parentId: 0, // 父id
        id: 1, // 自己的id
        name: '企业' // 名称
      },
      modifyTreeList: [], // 用户修改部门列表

      roleList: [], // 角色列表

      userBindRoleDataList: [], // 用户绑定角色列表

      userRow: {}, // 被选中的 user
      concernOrControl: '关注', // 关注或控制操作
      addEquipAuths: [], // 添加的用户设备权限
      removeEquipAuths: [], // 删除的用户设备权限

      setMsgLoading: false // 是否在保存数据中
    }
  },
  watch: {
    form: {
      handler(newValue, oldValue) {

      },
      deep: true // 设为true时监听对象属性值变化
    }
  },
  mounted() {
    this.init()
  },
  methods: {
    async init() {
      await this.getRole()
      await this.getTreeList()
      this.getDataList()
    },
    async getTreeList() {
      const treeList = []
      const res = await corpList() // 列表（用户部门树）
      const list = res.return.data
      let cdMsg = {} // 成都市分公司信息
      if (list) {
        list.forEach(item => {
          item.name = item.corpName
          if (item.name !== '中国铁塔股份有限公司' && item.name !== '四川省分公司') {
            item.expand = false // 不展开
          }
          if (item.name === '成都市分公司') {
            cdMsg = item
          } else {
            treeList.push(item)
          }
        })
      }
      this.treeList = objectSort(treeList, 'level')
      this.treeList.splice(3, 0, cdMsg)
      let userGroupId = null
      if (this.userInfo.corpIds.length !== 0) {
        userGroupId = this.userInfo.corpIds[0]
      }
      if (userGroupId) {
        treeList.forEach(item => {
          if (item.id === userGroupId) {
            item.selected = true
            this.targetGroupNowForm = item
          }
        })
      } else {
        const fristTree = treeList[0]
        if (fristTree) {
          fristTree.selected = true
          this.targetGroupNowForm = fristTree
        }
      }

      this.modifyTreeList = lodash.cloneDeep(this.treeList)
    },
    resetData() { // 重置数据
      this.searchMsg = ''
      this.search()
    },
    async getDataList() {
      let corpId = this.targetGroupNowForm.id
      if (this.userInfo.username === 'ditoo') {
        corpId = null
      }
      const tableData = []
      const res = await getUserList({
        page: this.page,
        limit: this.limit,
        userSearchName: this.searchMsg,
        userGroupId: corpId
      })
      const list = res.return.data
      if (list) {
        this.total = res.return.total
        list.forEach(item => {
          const date = formatDate(item.createTime)
          item.time = date.year + '-' + date.month + '-' + date.day + ' ' + date.hour + ':' + date.minute + ':' + date.second
          item.truename = item.trueName
          item.roleIds = []
          item.roleNames = ''
          tableData.push(item)
        })
      }
      this.tableData = tableData
      this.setUserBindRole()
    },
    async setUserBindRole() {
      const userBindRoleRes = await userBindRoleList()
      const list = userBindRoleRes.return.data
      if (list) {
        list.forEach(item => {
          this.roleList.forEach(roleItem => {
            if (item.roleId === roleItem.id) {
              item.roleName = roleItem.roleName
            }
          })
        })
      }
      const tolinkUserBindRoleRes = await userTolinkBindRoleList()
      const tolinkList = tolinkUserBindRoleRes.return.data
      if (tolinkList) {
        tolinkList.forEach(item => {
          this.roleList.forEach(roleItem => {
            if (item.roleId === roleItem.id) {
              item.roleName = roleItem.roleName
            }
          })
        })
      }
      this.userBindRoleDataList = list.concat(tolinkList)
      this.tableData.forEach(item => {
        this.userBindRoleDataList.forEach(userBindRoleItem => {
          if (item.id === userBindRoleItem.userId) {
            item.roleIds.push(userBindRoleItem.roleId)
            item.roleNames += userBindRoleItem.roleName + ';'
          }
        })
      })
      this.tableKey++
    },
    chooseTargetgroup(e) { // 点击左侧导航
      this.targetGroupNowForm = {
        level: e.level, // 目录级别
        parentId: e.parentId, // 父id
        id: e.id, // 自己的id
        name: e.title // 名称
      }
      if (e.titleColor === '') {
        this.targetGroupNowForm.name = ''
      }
      this.getDataList()
    },
    addTaoup() { // 新建子部门
      this.dialogStatus = 'newTaoup'
      this.resetForm(this.targetGroupNowForm)
      this.showDrawer = true // 表单可见
    },
    modifyTaoup() { // 修改部门名称
      this.dialogStatus = 'modifyTaoupName'
      this.resetForm(this.targetGroupNowForm)
      this.showDrawer = true // 表单可见
    },
    async removeTaoup() { // 删除当前部门
      const sure = await meAlert(this, {
        title: '确认删除',
        content: `此操作将永久删除${this.targetGroupNowForm.name}，是否继续？`
      })
      if (sure) {
        await deleteCorp({
          id: this.targetGroupNowForm.id
        })
        Message({
          message: `${this.targetGroupNowForm.name}删除成功`,
          type: 'success',
          duration: 3 * 1000
        })
        this.getTreeList()
      }
    },
    sortChange(data) { // 监听触发排序事件
      const { prop, order } = data
      if (prop === 'orderNum') {
        if (order === 'ascending') {
          console.log('倒序')
        } else if (order === 'descending') {
          console.log('顺序')
        }
      }
    },
    getMsgList(e) { // 页码改变时触发
      this.page = e.page
      this.limit = e.limit
      this.getDataList()
    },
    async add() { // 点击添加按钮
      this.dialogStatus = 'add'
      this.clearPasswordPolicyServerError()
      this.resetForm()
      this.showDrawer = true // 表单可见
    },
    async modify(row) {
      this.dialogStatus = 'modify'
      this.resetForm(row)
      this.showDrawer = true // 表单可见
    },
    clickRestPassword(row) { // 点击重置密码
      this.clearPasswordPolicyServerError()
      this.form = {
        username: row.username,
        trueName: row.trueName,
        newPassword: '', // 新密码
        sureNewPassword: '', // 确认新密码
        managerPassword: '', // 管理员密码
        userEntityId: row.id // 被重置用户的id
      }
      this.dialogStatus = 'restPassword'
      this.showDrawer = true
    },
    async resetForm(row) { // 重置表单
      this.setMsgLoading = false
      if (this.dialogStatus === 'add') {
        const notHasRole = await this.getRole()
        if (notHasRole) { // 不存在角色
          Message({
            message: '不存在角色，请到角色管理界面添加',
            type: 'error',
            duration: 3 * 1000
          })
          return
        }
        const roleIds = []
        if (this.roleList.length !== 0) {
          roleIds.push(this.roleList[0].key)
        }

        this.form = {
          username: '', // 用户名 String*
          password: '', // 用户密码 String*(公钥加密密码)
          truename: '', // 真实姓名 String
          phone: '', // 电话号码 String
          email: '', // 邮件地址 String
          parentId: this.userInfo.id, // 创建人ID int
          paleName: '', // 平台名称 String
          paleType: '', // 平台类别 String
          baned: 0, // 是否禁用 int （0：否，1：是）

          roleIds, // 角色id数组
          corpIds: [this.targetGroupNowForm.id] // 部门id数组
        }
      } else if (this.dialogStatus === 'modify') {
        await this.getRole()
        const modifyTreeList = lodash.cloneDeep(this.modifyTreeList)
        modifyTreeList.forEach(item => {
          item.selected = false
          if (item.id === row.corpIds[0]) {
            item.selected = true
          }
        })
        this.modifyTreeList = modifyTreeList
        this.form = {
          id: row.id,
          username: row.username, // 用户名 String*
          password: row.password, // 用户密码 String*(公钥加密密码)
          truename: row.truename, // 真实姓名 String
          phone: row.phone, // 电话号码 String
          email: row.email, // 邮件地址 String
          parentId: row.parentId, // 创建人ID int
          paleName: row.paleName, // 平台名称 String
          paleType: row.paleType, // 平台类别 String
          baned: row.baned, // 是否禁用 int （0：否，1：是）

          lastRoleIds: row.roleIds, // 修改钱角色id数组
          roleIds: row.roleIds, // 角色id数组
          corpIds: row.corpIds // 部门id数组
        }
      } else if (this.dialogStatus === 'newTaoup') { // 新建部门
        this.targetGroupForm = {
          level: row.level + 1, // 目录级别
          parentId: row.id, // 父id
          name: '' // 名称
        }
        if (row.name === '') {
          this.targetGroupForm = {
            level: 0, // 目录级别
            parentId: 0, // 父id
            name: '' // 名称
          }
        }
      } else if (this.dialogStatus === 'modifyTaoupName') { // 修改部门
        this.targetGroupForm = row
      }
    },
    async getRole() { // 获取角色列表
      const roleList = []
      const res = await getRoleList({
        page: 1,
        limit: 1000000
      })
      const list = res.return.data
      if (list) {
        list.forEach(item => {
          item.key = item.id
          item.name = item.roleName
          roleList.push(item)
        })
      }
      const tolinkRes = await getTolinkRoleList({ // 查询业务云角色列表
        page: 1,
        limit: 1000000
      })
      const tolinkList = tolinkRes.return.data
      if (tolinkList) {
        tolinkList.forEach(item => {
          item.key = item.id
          item.name = item.roleName
          roleList.push(item)
        })
      }
      this.roleList = roleList
      return this.roleList.length === 0
    },
    changeUserGroup(e) { // 修改部门
      this.form.corpIds = [e.id]
    },
    setData() { // 确认添加或修改
      this.$refs['dataForm'].validate((valid) => {
        if (valid) {
          this.setMsgLoading = true
          if (this.dialogStatus === 'add') {
            const form = lodash.cloneDeep(this.form)
            const submittedPassword = form.password
            form.username = form.username.replace(' ', '')
            form.password = getRSApassword(this.form.password, this)
            addUser(form).then(() => {
              this.$notify({
                title: '添加请求发送成功',
                message: `成功添加`,
                type: 'success',
                duration: 3 * 1000
              })
              getUserList({
                page: 1,
                limit: 10000
              }).then(userListRes => { // 进行绑定角色
                const list = userListRes.return.data
                if (list) {
                  let userId = null
                  list.forEach(item => {
                    if (item.username === form.username) {
                      userId = item.id
                    }
                  })
                  if (userId) {
                    form.roleIds.forEach(roleId => {
                      userBindingRole({ // 绑定角色
                        userId: userId,
                        roleId
                      })
                    })
                  }
                }
                this.getMsgList({
                  page: this.page,
                  limit: this.limit
                })
                this.showDrawer = false
                this.setMsgLoading = false
              })
            }).catch(error => {
              this.handlePasswordPolicyError(error, 'password', submittedPassword)
              this.setMsgLoading = false
            })
          } else if (this.dialogStatus === 'modify') {
            updateUser(this.form).then(() => {
              getUserList({
                page: 1,
                limit: 10000
              }).then(userListRes => {
                const list = userListRes.return.data
                if (list) {
                  let userData = {} // 当前操作用户的信息
                  list.forEach(item => {
                    if (item.username === this.form.username) {
                      userData = item
                    }
                  })

                  this.form.lastRoleIds.forEach(lastRoleId => { // 处理解绑角色
                    let lastHasRole = false
                    this.form.roleIds.forEach(roleId => {
                      if (lastRoleId === roleId) {
                        lastHasRole = true
                      }
                    })
                    if (!lastHasRole) {
                      this.userBindRoleDataList.forEach(userBindRoleItem => {
                        if (lastRoleId === userBindRoleItem.roleId && userData.id === userBindRoleItem.userId) {
                          userUnBindRole({ // 解绑角色
                            id: userBindRoleItem.id
                          })
                        }
                      })
                    }
                  })
                  this.form.roleIds.forEach(roleId => { // 处理绑定角色
                    let hasRole = false
                    this.form.lastRoleIds.forEach(lastRoleId => {
                      if (roleId === lastRoleId) {
                        hasRole = true
                      }
                    })
                    if (!hasRole) {
                      userBindingRole({ // 绑定角色
                        userId: userData.id,
                        roleId
                      })
                    }
                  })
                }
                setTimeout(() => {
                  this.$notify({
                    title: '修改请求发送成功',
                    message: `成功修改`,
                    type: 'success',
                    duration: 3 * 1000
                  })
                  this.getMsgList({
                    page: this.page,
                    limit: this.limit
                  })
                  this.showDrawer = false
                  this.setMsgLoading = false
                }, 2000)
              })
            })
          } else if (this.dialogStatus === 'newTaoup') {
            const targetGroupForm = lodash.cloneDeep(this.targetGroupForm)
            targetGroupForm.corpName = targetGroupForm.name
            addCorp(targetGroupForm).then(() => {
              this.$notify({
                title: '新建部门请求发送成功',
                message: `成功创建`,
                type: 'success',
                duration: 3 * 1000
              })
              this.getTreeList()
              this.showDrawer = false
            })
          } else if (this.dialogStatus === 'modifyTaoupName') {
            const targetGroupForm = lodash.cloneDeep(this.targetGroupForm)
            targetGroupForm.corpName = targetGroupForm.name
            updateCorp(targetGroupForm).then(() => {
              this.$notify({
                title: '修改企业名称请求发送成功',
                message: `成功修改`,
                type: 'success',
                duration: 3 * 1000
              })
              this.getTreeList()
              this.showDrawer = false
            })
          } else if (this.dialogStatus === 'restPassword') {
            this.addOrModifyloading = false
            const form = JSON.parse(JSON.stringify(this.form))
            const submittedPassword = form.newPassword
            form.newPassword = getRSApassword(this.form.newPassword, this) // 新密码
            form.managerPassword = getRSApassword(this.form.managerPassword, this) // 管理员密码
            resetPassword(form).then(res => {
              if (res.code === 200) {
                this.$notify({
                  title: '请求发送成功',
                  message: `${this.form.username}的密码已重置`,
                  type: 'success',
                  duration: 3 * 1000
                })
                this.showDrawer = false
                this.setMsgLoading = false
              }
            }).catch(error => {
              this.handlePasswordPolicyError(error, 'newPassword', submittedPassword)
              this.setMsgLoading = false
            })
          }
        }
      })
    },
    getPasswordPolicyServerMessage(fieldName, value) {
      const validation = this.passwordPolicyServerValidation
      if (validation.fieldName === fieldName && validation.value === value) {
        return validation.message
      }
      return ''
    },
    clearPasswordPolicyServerError(fieldName) {
      const validation = this.passwordPolicyServerValidation
      if (!validation.message || (fieldName && validation.fieldName !== fieldName)) {
        return
      }
      const validatedField = validation.fieldName
      this.passwordPolicyServerValidation = {
        fieldName: '',
        value: '',
        message: ''
      }
      if (this.$refs.dataForm && validatedField) {
        this.$refs.dataForm.clearValidate(validatedField)
      }
    },
    handlePasswordPolicyError(error, fieldName, rejectedValue) {
      if (error && error.errorCode === PASSWORD_POLICY_ERROR_CODE) {
        this.passwordPolicyServerValidation = {
          fieldName,
          value: rejectedValue,
          message: PASSWORD_POLICY_MESSAGE
        }
        this.$nextTick(() => {
          if (this.$refs.dataForm) {
            this.$refs.dataForm.validateField(fieldName, () => {})
          }
        })
      }
    },
    async remove(row) {
      const sure = await meAlert(this, {
        title: '确认删除',
        content: `此操作将永久删除${row.username}，是否继续？`
      })
      if (sure) {
        for (let i = 0; i < row.roleIds.length; i++) {
          let userBindRoleId = '' // 用户角色绑定id
          this.userBindRoleDataList.forEach(userBindRoleItem => {
            if (row.roleIds[i] === userBindRoleItem.roleId && row.id === userBindRoleItem.userId) {
              userBindRoleId = userBindRoleItem.id
            }
          })
          if (userBindRoleId !== '') {
            await userUnBindRole({ // 解绑角色
              id: userBindRoleId
            })
          }
        }

        await deleteUser({
          id: row.id
        })
        Message({
          message: `${row.username}删除成功`,
          type: 'success',
          duration: 3 * 1000
        })
        this.getMsgList({
          page: this.page,
          limit: this.limit
        })
      }
    },
    search() { // 查询
      this.getDataList()
    },
    equipConcernGrantAuth(row) { // 设备关注授权
      this.concernOrControl = '关注'
      this.userRow = row
      this.dialogEquipAuth = true
    },
    equipControlGrantAuth(row) { // 设备控制授权
      this.concernOrControl = '控制'
      this.userRow = row
      this.dialogEquipAuth = true
    },
    equipAuthSelectChange(e) {
      const addEquipAuths = []
      e.nowVals.forEach(nowVal => {
        let hasAdd = true // 对应设备权限是否添加
        e.multipleSelection.forEach(item => {
          if (nowVal.id === item.id) {
            hasAdd = false
          }
        })
        if (hasAdd) {
          addEquipAuths.push(nowVal.id)
        }
      })

      const removeEquipAuths = []
      e.multipleSelection.forEach(item => {
        let hasRemove = true // 对应设备权限是否删除
        e.nowVals.forEach(nowVal => {
          if (item.id === nowVal.id) {
            hasRemove = false
          }
        })
        if (hasRemove) {
          removeEquipAuths.push(item.id)
        }
      })

      this.addEquipAuths = addEquipAuths // 添加的用户设备权限
      this.removeEquipAuths = removeEquipAuths // 删除的用户设备权限
    },
    async sureChange() { // 确定修改设备权限
      let addRes = {
        code: 0
      }
      if (this.addEquipAuths.length !== 0) {
        if (this.concernOrControl === '关注') {
          addRes = await addUserConcernEquipment({
            grantedUser: this.userRow.username,
            equipmentIds: this.addEquipAuths
          })
        } else if (this.concernOrControl === '控制') {
          addRes = await addUserControlEquipment({
            grantedUser: this.userRow.username,
            equipmentIds: this.addEquipAuths
          })
        }
      }
      let removeRes = {
        code: 0
      }
      if (this.removeEquipAuths.length !== 0) {
        if (this.concernOrControl === '关注') {
          removeRes = await deleteUserConcernEquipment({
            grantedUser: this.userRow.username,
            equipmentIds: this.removeEquipAuths
          })
        } else if (this.concernOrControl === '控制') {
          removeRes = await deleteUserControlEquipment({
            grantedUser: this.userRow.username,
            equipmentIds: this.removeEquipAuths
          })
        }
      }
      if ((addRes.code === 0 || addRes.code === 200) && (removeRes.code === 0 || removeRes.code === 200)) {
        this.dialogEquipAuth = false
        Message({
          message: `操作成功`,
          type: 'success',
          duration: 3 * 1000
        })
      } else {
        if (addRes.code !== 0) {
          Message({
            message: `添加设备权限失败`,
            type: 'error',
            duration: 3 * 1000
          })
        }
        if (removeRes.code !== 0) {
          Message({
            message: `删除设备权限失败`,
            type: 'error',
            duration: 3 * 1000
          })
        }
      }
    }
  }
}
</script>

<style lang="stylus" scoped>
.treeContentCss {
  margin 1% 1% 0 1%
  padding 1% 0 0 1%
  width 18%
  background-color #F3F7FA
  float left
}
.titleCss {
  margin-bottom 8px
  font-size 25px
}
.dataContentCss {
  width 80%
  float right
}
.password-policy-form {
  :deep(.el-form-item__content) {
    align-items flex-start
  }
  :deep(.el-form-item__error) {
    position static
    width 100%
    padding-top 6px
    color #F56C6C
    line-height 18px
    text-align left
    white-space normal
  }
}
</style>
