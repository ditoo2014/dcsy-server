<template>
  <div class="app-container">
    <div class="filter-container">
      <el-button class="filter-item" style="margin-left: 10px;" type="primary" :icon="$legacyIcon('el-icon-edit')" size="small" @click="add">
        添加角色
      </el-button>
    </div>
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
      style="width: 100%;"
    >
      <!-- label:表头名称,prop:表格内容,sortable:设置为"custom"时可以排序,align:对齐方式 -->
      <el-table-column label="角色名称" align="center">
        <template #default="scope">
          <span>{{ scope.row.roleName }}</span>
        </template>
      </el-table-column>
      <el-table-column label="描述" align="center">
        <template #default="scope">
          <span>{{ scope.row.roleDesc }}</span>
        </template>
      </el-table-column>
      <el-table-column label="添加时间" align="center">
        <template #default="scope">
          <span>{{ scope.row.createDate }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" align="center" fixed="right" width="300">
        <template #default="scope">
          <el-button type="primary" size="small" @click="modify(scope.row)">
            修改
          </el-button>
          <el-button type="danger" size="small" @click="remove(scope.row)">
            删除
          </el-button>
          <el-button type="primary" size="small" @click="setPermis(scope.row)">
            设置权限
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

    <!-- 添加修改弹窗 -->
    <elDrawer v-model="showDrawer" :title="textMap[dialogStatus]" :wrapper-closable="false" width="400">
      <el-form ref="dataForm" :rules="rules" :model="form" size="small" label-width="120px">
        <el-row>
          <el-col :span="20">
            <el-form-item label="角色名称" prop="roleName">
              <el-input v-model="form.roleName" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="20">
            <el-form-item label="描述" prop="roleDesc">
              <el-input v-model="form.roleDesc" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <div class="fixedBottomCss">
        <el-button @click="showDrawer = false">
          取消
        </el-button>
        <el-button type="primary" @click="setData()">
          确定
        </el-button>
      </div>
    </elDrawer>

    <!-- 设置权限弹窗 -->
    <elDrawer v-model="showSetPermisDrawer" title="权限设置" :wrapper-closable="false" width="700">
      <div style="height:1200px">
        <el-form size="small" label-width="120px">
          <MeTreeTransfer
            type="page"
            :title="permisPagTitle"
            :left-data="notHasPagPermis"
            :right-data="hasPagPermis"
            @select-data="selectPermis"
          />
          <br><br>
          <MeTreeTransfer
            type="function"
            :title="permisFuncTitle"
            :left-data="notHasFuncPermis"
            :right-data="hasFuncPermis"
            @select-data="selectPermis"
          />
        </el-form>
        <div class="fixedBottomCss">
          <el-button @click="showSetPermisDrawer = false">
            取消
          </el-button>
          <el-button type="primary" @click="surePermis()">
            确定
          </el-button>
        </div>
      </div>
    </elDrawer>

  </div>
</template>

<script>
import { Message } from '@/utils/element-shim'
import Pagination from '@/components/Pagination' // 分页组件
import MeTreeTransfer from '@/components/a_myComp/meTreeTransfer/MeTreeTransfer.vue'
import { meAlert, formatDateString, listdupliRemoval, unique, objectSort } from '@/utils/utils.js'
import { getRoleList, addRole, modifyRole, deleteRole,
  permissionList, rolePermiList, batchaddRolePermi, batchdeleteRolePermi
} from '@/api/user.js'

export default {
  components: {
    Pagination,
    MeTreeTransfer
  },
  data() {
    return {
      tableKey: 1, // 参数值变化，表格对应改变

      total: 0, // 数据总条数
      limit: 10, // 每次加载数据条数
      page: 1, // 页码

      showDrawer: false, // 抽屉是否显示
      textMap: { // 表单弹窗标题
        add: '添加数据流',
        modify: '修改数据流'
      },
      dialogStatus: '', // 表单状态值为'add'或'modify' ，与 textMap 对应

      showSetPermisDrawer: false, // 设置权限是否显示

      rules: {
        roleName: [{ required: true, trigger: 'blur', message: '角色名称不能为空' }],
        roleDesc: [{ required: true, trigger: 'blur', message: '描述不能为空' }]
      },

      tableData: [],
      form: {
        roleName: '', // 角色名称
        roleDesc: '' // 描述
      },

      selPagPermisList: [], // 选择的页面权限
      selFuncPermisList: [], // 选择的功能权限

      removePagPermisList: [], // 移除的页面权限
      removeFuncPermisList: [], // 移除的功能权限

      roleId: '', // 设置权限时的角色id
      allPagPermis: [], // 全部页面权限
      allFuncPermis: [], // 全部功能权限
      permisPagTitle: ['未添加页面', '已添加页面'],
      permisFuncTitle: ['未添加功能', '已添加功能'],
      notHasPagPermis: [], // 未添加的页面权限
      hasPagPermis: [], // 已添加的页面权限
      notHasFuncPermis: [], // 未添加的功能权限
      hasFuncPermis: [], // 已添加的功能权限
      allHasPermisList: [] // 所有的已有权限列表
    }
  },
  watch: {
    form: {
      handler(newValue, oldValue) {

      },
      deep: true // 设为true时监听对象属性值变化
    }
  },
  created() {
    this.init()
  },
  methods: {
    async init() {
      const tableData = []
      const res = await getRoleList({
        page: this.page,
        limit: this.limit
      })
      const list = res.return.data
      if (list) {
        this.total = res.return.total
        list.forEach(item => {
          item.createDate = formatDateString(item.createTime, 'formatDate')
          tableData.push(item)
        })
      }
      this.tableData = tableData
    },
    getMsgList(e) { // 页码改变时触发
      this.page = e.page
      this.limit = e.limit
      this.init()
    },
    add() { // 点击添加按钮
      this.dialogStatus = 'add'
      this.resetForm()
      this.showDrawer = true // 表单可见
      this.$nextTick(() => {
        this.$refs['dataForm'].clearValidate()
      })
    },
    resetForm(row) { // 重置表单
      if (this.dialogStatus === 'add') {
        this.form = {
          roleName: '', // 角色名称
          roleDesc: '' // 描述
        }
      } else if (this.dialogStatus === 'modify') {
        this.form = {
          id: row.id, // 角色标识符
          roleName: row.roleName, // 角色名称
          roleDesc: row.roleDesc // 描述
        }
      }
    },
    setData() { // 确认添加或修改
      this.$refs['dataForm'].validate((valid) => {
        if (valid) {
          if (this.dialogStatus === 'add') {
            addRole(this.form).then(() => {
              this.getMsgList({
                page: this.page,
                limit: this.limit
              })
              this.showDrawer = false
              this.$notify({
                title: '添加请求发送成功',
                message: `成功添加`,
                type: 'success',
                duration: 3 * 1000
              })
            })
          } else if (this.dialogStatus === 'modify') {
            modifyRole(this.form).then(res => {
              if (res.code === 0) {
                this.getMsgList({
                  page: this.page,
                  limit: this.limit
                })
                this.showDrawer = false
                this.$notify({
                  title: '修改请求发送成功',
                  message: `成功修改`,
                  type: 'success',
                  duration: 3 * 1000
                })
              }
            }).catch(err => {
              Message({
                message: '请求出错：' + err,
                type: 'error',
                duration: 3 * 1000
              })
            })
          }
        }
      })
    },
    modify(row) {
      this.dialogStatus = 'modify'
      this.resetForm(row)
      this.showDrawer = true // 表单可见
    },
    async remove(row) {
      const sure = await meAlert(this, {
        title: '确认删除',
        content: `此操作将永久删除${row.roleName}，是否继续？`
      })
      if (sure) {
        await deleteRole({ id: row.id })
        Message({
          message: `${row.roleName}删除成功`,
          type: 'success',
          duration: 3 * 1000
        })
        this.getMsgList({
          page: this.page,
          limit: this.limit
        })
      }
    },
    async setPermis(row) {
      this.roleId = row.id
      await this.setPermisData(row)
      this.showSetPermisDrawer = true
    },
    async getPermisList() { // 获取所有权限
      const res = await permissionList()
      const permisList = objectSort(res.return.data, 'parentId')
      const collectionPermissionList = [] // 采集云权限列表
      permisList.forEach(item => {
        if (item.platId === 0 || item.platId === 1) {
          collectionPermissionList.push({
            id: item.id,
            pid: item.parentId, // 父节点id
            label: item.permissionName, // 权限名称
            level: item.level, // 权限级别
            type: item.type, // 权限类型(1代表页面权限，2代表功能权限)
            children: []
          })
        }
      })

      const allPagPermis = [] // 全部页面权限
      collectionPermissionList.forEach(item => {
        if (item.type === 1) { // 页面权限
          allPagPermis.push(item)
        }
      })
      this.allPagPermis = allPagPermis // 全部页面权限
      const allFuncPermis = [] // 全部功能权限
      collectionPermissionList.forEach(item => {
        // if (item.type === 2) { // 功能权限
        allFuncPermis.push(item)
        // }
      })
      this.allFuncPermis = allFuncPermis // 全部功能权限
    },
    async setPermisData(row) { // 设置权限数据
      await this.getPermisList() // 获取全部页面和功能列表
      const res = await rolePermiList({ // 获取已添加页面权限
        roleId: row.id
      })
      const hasPermisList = res.return.data
      // 页面权限 start
      const hasPagPermis = [] // 已添加的页面权限
      this.allPagPermis.forEach(item => {
        hasPermisList.forEach(hasItem => {
          if (item.id === hasItem.permissionId) {
            hasPagPermis.push(item)
          }
        })
      })
      this.hasPagPermis = hasPagPermis // 已添加的页面权限
      this.selPagPermisList = hasPagPermis

      const notHasPagPermisArray = [] // 未添加的页面权限列表
      for (let i = 0; i < this.allPagPermis.length; i++) { // 未添加页面权限
        const permis = this.allPagPermis[i]
        let hasPagPermisKey = false
        this.hasPagPermis.forEach(pagPermis => {
          if (permis.id === pagPermis.id) {
            hasPagPermisKey = true
          }
        })
        if (!hasPagPermisKey) {
          notHasPagPermisArray.push(permis)
        }
      }
      this.notHasPagPermis = this.completPermisData(notHasPagPermisArray, this.allPagPermis.concat(this.allPagPermis)) // 补全权限数据
      // 页面权限 end

      // 功能权限 start
      const hasFuncPermis = [] // 已添加的功能权限
      this.allFuncPermis.forEach(item => {
        hasPermisList.forEach(hasItem => {
          if (item.id === hasItem.permissionId) {
            hasFuncPermis.push(item)
          }
        })
      })
      this.hasFuncPermis = hasFuncPermis // 已添加的功能权限
      this.selFuncPermisList = hasFuncPermis

      const notHasFuncPermisArray = [] // 未添加的功能权限列表
      for (let i = 0; i < this.allFuncPermis.length; i++) { // 未添加功能权限
        const permis = this.allFuncPermis[i]
        let hasFuncPermisKey = false
        this.hasFuncPermis.forEach(funcPermis => {
          if (funcPermis.id === permis.id) {
            hasFuncPermisKey = true
          }
        })
        if (!hasFuncPermisKey) {
          notHasFuncPermisArray.push(permis)
        }
      }
      this.notHasFuncPermis = this.completPermisData(notHasFuncPermisArray, this.allPagPermis) // 补全权限数据
      this.notHasFuncPermis = this.completPermisData(this.notHasFuncPermis, this.allPagPermis) // 补全权限数据
      // 功能权限 end
    },
    completPermisData(permisList, allPermisList) { // 补全权限数据
      let hasPermisParentArray = []
      for (let i = 0; i < permisList.length; i++) {
        const permis = permisList[i]
        if (permis.pid || permis.pid === 0) {
          let hasPermisParent = false // 用于判断该权限的父级权限是否已存在于权限列表中
          permisList.forEach(allPermis => {
            if (permis.pid === allPermis.id) {
              hasPermisParent = true
            }
          })
          if (!hasPermisParent) { // 如果不存在，则添加
            allPermisList.forEach(allPermis => {
              if (permis.pid === allPermis.id) {
                hasPermisParentArray.push(allPermis)
              }
            })
          }
        }
      }
      hasPermisParentArray = listdupliRemoval(hasPermisParentArray, 'id') // 去重
      permisList = hasPermisParentArray.concat(permisList)
      if (hasPermisParentArray.length !== 0) {
        return this.completPermisData(permisList, allPermisList)
      } else {
        return permisList
      }
    },
    selectPermis(e) {
      if (e.type === 'page') {
        this.selPagPermisList = e.toData
        this.removePagPermisList = e.fromData // 移除的页面权限
      } else if (e.type === 'function') {
        this.selFuncPermisList = e.toData
        this.removeFuncPermisList = e.fromData // 移除的功能权限
      }
    },
    async surePermis() { // 确定权限设置
      let toList = [] // 右侧选择的权限列表
      const allSelectPermisList = this.selPagPermisList.concat(this.selFuncPermisList) // 当前选择的权限
      // 添加权限 start
      allSelectPermisList.forEach(item => {
        toList.push({
          permissionId: item.id,
          roleId: this.roleId
        })
        item.children.forEach(child => {
          toList.push({
            permissionId: child.id,
            roleId: this.roleId
          })
          child.children.forEach(child2 => {
            toList.push({
              permissionId: child2.id,
              roleId: this.roleId
            })
            child2.children.forEach(child3 => {
              toList.push({
                permissionId: child3.id,
                roleId: this.roleId
              })
            })
          })
        })
      })
      toList = unique(toList, 'permissionId') // 去重
      toList = objectSort(toList, 'permissionId') // 排序
      if (toList.length !== 0) {
        await batchaddRolePermi({ list: toList })
      }
      // 添加权限 end

      // 移除权限 start
      let fromData = [] // 选择到左侧的权限列表
      const allRemovePermisList = this.removePagPermisList.concat(this.removeFuncPermisList) // 当前移除的权限
      allRemovePermisList.forEach(item => {
        fromData.push({
          permissionId: item.id,
          roleId: this.roleId
        })
        item.children.forEach(child => {
          fromData.push({
            permissionId: child.id,
            roleId: this.roleId
          })
          child.children.forEach(child2 => {
            fromData.push({
              permissionId: child2.id,
              roleId: this.roleId
            })
            child2.children.forEach(child3 => {
              fromData.push({
                permissionId: child3.id,
                roleId: this.roleId
              })
            })
          })
        })
      })
      fromData = unique(fromData, 'permissionId') // 去重
      fromData = objectSort(fromData, 'permissionId') // 排序
      if (fromData.length !== 0) {
        await batchdeleteRolePermi({ list: fromData })
      }
      // 移除权限 end
      this.showSetPermisDrawer = false
      this.$notify({
        title: '权限设置成功',
        message: `成功设置`,
        type: 'success',
        duration: 3 * 1000
      })
    }
  }
}
</script>
