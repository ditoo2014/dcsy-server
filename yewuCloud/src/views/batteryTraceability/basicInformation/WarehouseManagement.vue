<template>
  <div class="contentCss">
    <el-scrollbar class="treeContentCss allheight">
      <MyElTree :navi-name-list="treeList" @click-tree="chooseTargetgroup" />
    </el-scrollbar>
    <div class="dataContentCss">
      <PageBanner
        :list="bannerList"
        @click-button="clickBannerBtn"
        @input-change="inputBannerChange"
        @input-keyup-enter="inputBannerKeyupEnter"
      />
      <el-scrollbar class="scrollbarCss">
        <el-table
          :key="tableKey"
          height="630"
          :data="tableData"
          fit
          highlight-current-row
        >
          <el-table-column label="序号" width="50" align="center">
            <template #default="scope">
              <span>{{ scope.row.rowNumber }}</span>
            </template>
          </el-table-column>
          <el-table-column label="编码" width="200" align="center">
            <template #default="scope">
              <span>{{ scope.row.warehouseCode }}</span>
            </template>
          </el-table-column>
          <el-table-column label="名称" width="170" align="center">
            <template #default="scope">
              <span>{{ scope.row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column label="库房地址" width="285" align="center">
            <template #default="scope">
              <span>{{ scope.row.address }}</span>
            </template>
          </el-table-column>
          <el-table-column label="负责人" width="100" align="center">
            <template #default="scope">
              <span>{{ scope.row.managerName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="电话" width="110" align="center">
            <template #default="scope">
              <span>{{ scope.row.managerPhone }}</span>
            </template>
          </el-table-column>
          <el-table-column label="所属公司" width="200" align="center">
            <template #default="scope">
              <span>{{ scope.row.corpName }}</span>
            </template>
          </el-table-column>
          <el-table-column
            label="操作"
            align="center"
            class-name="small-padding fixed-width"
            width="220"
          >
            <template #default="scope">
              <el-button type="primary" size="small" @click="modify(scope.row)">
                修改
              </el-button>
              <el-button
                type="danger"
                size="small"
                @click="remove(scope.row)"
              >删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <Pagination
          v-show="total>0"
          v-model:page="page"
          v-model:limit="limit"
          :total="total"
          @pagination="getMsgList"
        />
      </el-scrollbar>
    </div>

    <!-- 右侧弹窗 -->
    <ElDrawerWrap v-model="showDrawer" :title="textMap[dialogStatus]" :wrapper-closable="false" width="500">
      <el-form
        v-if="dialogStatus === 'add' || dialogStatus === 'modify'"
        ref="dataForm"
        :rules="rules"
        :model="form"
        size="small"
        label-width="120px"
      >
        <el-row>
          <el-col v-if="dialogStatus === 'add'" :span="20">
            <el-form-item label="所属公司">
              <span>{{ form.corpName }}</span>
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="编码" prop="warehouseCode">
              <el-input v-model="form.warehouseCode" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="名称" prop="name">
              <el-input v-model="form.name" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="库房地址" prop="addr">
              <el-input v-model="form.addr" disabled>
                <template #append>
                  <el-button :icon="$legacyIcon('el-icon-search')" @click="chooseMap" />
                </template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="负责人" prop="managerName">
              <el-input v-model="form.managerName" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="电话" prop="managerPhone">
              <el-input v-model="form.managerPhone" />
            </el-form-item>
          </el-col>
          <el-col v-if="dialogStatus === 'modify'" :span="20">
            <el-form-item label="所属公司">
              <MyElTree :navi-name-list="modifyTreeList" @click-tree="changeUserGroup" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <el-form
        v-else-if="dialogStatus === 'upload'"
        ref="dataForm"
        :model="form"
        size="small"
        label-width="140px"
      >
        <el-row>
          <el-col :span="24">
            <el-form-item label="上传excel">
              <UploadFile :limit="1" :file-name-list="form.fileList" @select-file="selectFile" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-button @click="uploadTemp">下载模板</el-button>
          </el-col>
        </el-row>
      </el-form>
      <div class="fixedBottomCss">
        <el-button @click="showDrawer = false">取消</el-button>
        <el-button type="primary" @click="setData()">确定</el-button>
      </div>
    </ElDrawerWrap>

    <!-- 选择地图弹出层 -->
    <el-dialog v-model="dialogMap" title="请选择详细地址" width="70%">
      <div v-if="dialogMap" class="mapContentCss">
        <SelectMapArea
          :map-msg="mapMsg"
          @map="mapinfo"
        />
      </div>
    </el-dialog>

    <ExportExcel ref="ExportExcelRef" />
  </div>
</template>

<script>
import { Message } from '@/utils/elementShim'
import Pagination from '@/components/Pagination' // 分页组件
import PageBanner from '@/components/ccpcComp/Banner.vue'
import MyElTree from '@/components/sharedComponents/elTree/MyElTree.vue' // 左侧导航
import UploadFile from '@/components/sharedComponents/uploadFile/UploadFile.vue'
import SelectMapArea from '@/components/sharedComponents/selectMap/SelectMapArea.vue'
import ExportExcel from '@/components/sharedComponents/exportExcel/ExportExcel.vue'
import ElDrawerWrap from '@/components/elDrawer/index.vue'
import { corpList, findSelf } from '@/api/user.js'
import { checkPhone } from '@/utils/validate.js'
import { meAlert, objectSort, exportBlobFile } from '@/utils/utils.js'
import {
  buildQueryableCorpKeys,
  findDefaultCorp,
  resolveCorpQueryId
} from '@/utils/corpScope.js'
import { processDrawList } from '@/utils/ccpcData.js'
import { findEquipmentTypeList } from '@/api/basicData.js'
import { findWarehouseList, saveWarehouse, deleteWarehouse, modifyWarehouse } from '@/api/warehouse.js'
import { warehouseExport, warehouseImport, warehouseImportTemplate } from '@/api/batteryTraceability.js'
import { chinaRegions } from '@/components/sharedComponents/mapComplex/index.js'
import lodash from 'lodash'
import totalImage from '@/assets/imgs/total.png'

export default {
  // KeepAlive include uses the existing lower-case route name.
  // eslint-disable-next-line vue/component-definition-name-casing
  name: 'WarehouseManagement',
  components: {
    Pagination,
    PageBanner,
    MyElTree,
    UploadFile,
    SelectMapArea,
    ExportExcel,
    ElDrawerWrap
  },
  data() {
    const validatePhone = (rule, value, callback) => {
      if (checkPhone(value)) {
        callback()
      } else {
        callback(new Error('请输入正确的电话号码'))
      }
    }
    return {
      userInfo: {},
      queryableCorpKeys: [],

      bannerList: [
        { style: 'infoNum', icon: totalImage, title: '库房总量', num: '0', unit: '个' },
        { style: 'input', placeholder: '请输入库房名称', input: '' },
        { style: 'button', buttonName: '查询', type: 'primary' },
        { style: 'button', buttonName: '重置', type: '' },
        { style: 'button', buttonName: '导出', type: '' },
        { style: 'button', buttonName: '导入', type: '' },
        { style: 'button', buttonName: '新增', type: 'primary' }
      ],

      searchTree: '', // 查询tree节点
      allGroupList: [], // 所以分组列表
      groupList: [], // 分组列表

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

      tableKey: 1, // 参数值变化，表格对应改变
      total: 0, // 数据总条数
      limit: 10, // 每次加载数据条数
      page: 1, // 页码

      warehouseSearchName: '', // 查询设备

      showDrawer: false, // 抽屉是否显示
      textMap: { // 表单弹窗标题
        add: '添加库房',
        modify: '修改库房',
        upload: '导入excel'
      },
      dialogStatus: '', // 表单状态值为'add'或'modify' ，与 textMap 对应
      rules: {
        name: [{ required: true, trigger: 'blur', message: '库房名称不能为空' }],
        warehouseCode: [{ required: true, trigger: 'blur', message: '库房编码不能为空' }],
        managerPhone: [{ required: true, trigger: 'blur', validator: validatePhone }],
        addr: [{ required: true, trigger: 'blur', message: '地理位置不能为空' }]
      },

      allTableData: [], // 全部数据
      tableData: [],
      selectedTableData: [], // 选中数据
      form: {
        name: '', // 库房名称
        address: '', // 库房地址
        managerName: '', // 库房管理员
        managerUserId: '', // 库房管理员id
        managerPhone: '', // 管理员电话
        warehouseCode: '', // 编码
        corpId: '', // 部门id
        createUserId: '' // 创建用户id
      },

      adminList: [], // 管理员列表
      processDrawList, // 工艺图模板列表

      equipTypeList: [], // 库房类型列表
      serNumList: [ // 编码规则列表
        { key: '6', name: '规则6' },
        { key: '5', name: '规则5' },
        { key: '4', name: '规则4' },
        { key: '3', name: '规则3' },
        { key: '2', name: '规则2' },
        { key: '1', name: '规则1' }
      ],
      siteList: [], // 站址列表
      apiKeys: [], // 站址实际使用的apiKey列表

      reportIntervalUnitOptions: [ // 填报间隔类型列表
        { key: '小时', name: '小时' },
        { key: '日', name: '日' },
        { key: '月', name: '月' },
        { key: '年', name: '年' }
      ],

      chinaRegions, // 省份及其下辖的市

      mapMsg: {},
      dialogMap: false // 选择地理位置的窗口是否显示
    }
  },
  computed: {
    dataReportIntervalMax() {
      return (val) => {
        let max = 10000
        switch (val) {
          case '日': max = 28; break
          case '月': max = 11; break
        }
        return max
      }
    },
    inspectionReportIntervalMax() {
      return (val) => {
        let max = 10000
        switch (val) {
          case '日': max = 28; break
          case '月': max = 11; break
        }
        return max
      }
    }
  },
  watch: {
    form: {
      handler(newValue, oldValue) {},
      deep: true // 设为true时监听对象属性值变化
    }
  },
  mounted() {
    this.init()
  },
  methods: {
    async init() {
      try {
        const [allTableData, corpRes, userRes] = await Promise.all([
          this.getAllDataList(),
          corpList(), // 列表（用户部门树）
          findSelf()
        ])
        const currentUser = userRes.return && userRes.return.user
        if (!currentUser) throw new Error('invalid current user response')
        this.userInfo = currentUser
        await this.getTreeList(allTableData, corpRes)
        await this.getDataList()
      } catch (error) {
        console.error(error)
        this.total = 0
        this.tableData = []
        this.setTotalBanner()
      }
    },
    async getAllDataList() { // 获取全部数据列表
      const res = await findWarehouseList({
        pageNum: 1,
        pageSize: 1000000
      })
      let allTableData = []
      if (res.return) {
        const list = res.return.data
        if (list) {
          allTableData = list
        }
      }
      return allTableData
    },
    async getTreeList(allTableData, res) {
      const list = res.return.data
      const { treeList, chengduCompany } = this.setTreeExpand({
        list,
        allTableData
      })

      this.treeList = objectSort(treeList, 'level')
      this.treeList.splice(3, 0, chengduCompany)
      this.treeList.forEach(item => { item.selected = false })
      this.queryableCorpKeys = buildQueryableCorpKeys(this.treeList, this.userInfo)
      const defaultCorp = findDefaultCorp(this.treeList, this.userInfo, this.queryableCorpKeys)
      if (defaultCorp) {
        defaultCorp.selected = true
        this.targetGroupNowForm = defaultCorp
      }
      this.modifyTreeList = lodash.cloneDeep(this.treeList)
    },
    setTreeExpand(params) {
      const { list, allTableData } = params
      let chengduCompany = {} // 成都市分公司信息
      const treeList = []
      if (list) {
        for (const key in this.chinaRegions) { // 设置省份对于部门id
          const regionItem = this.chinaRegions[key]
          regionItem.corpIds = []
          list.forEach(item => {
            if (item.corpName.indexOf(key.slice(0, 2)) !== -1) {
              regionItem.corpIds.push(item.id)
            } else {
              regionItem.children.forEach(child => {
                if (item.corpName.indexOf(child.slice(0, 2)) !== -1) {
                  regionItem.corpIds.push(item.id)
                }
              })
            }
          })
        }
        for (const key in this.chinaRegions) { // 设置省份对于部门id
          const regionItem = this.chinaRegions[key]
          regionItem.dataNum = 0
          regionItem.corpIds.forEach(corpId => {
            allTableData.forEach(dataItem => {
              if (corpId === dataItem.corpId) {
                regionItem.dataNum++
              }
            })
          })
        }
        list.forEach(item => {
          item.name = item.corpName
          item.expand = false
          if (item.name === '中国铁塔股份有限公司') {
            item.expand = true
          }
          for (const key in this.chinaRegions) { // 设置省份对于部门id
            const regionItem = this.chinaRegions[key]
            regionItem.corpIds.forEach(corpId => {
              if (item.id === corpId) {
                if (regionItem.dataNum > 0) {
                  item.expand = true
                }
              }
            })
          }
          if (item.name === '成都市分公司') {
            chengduCompany = item
          } else {
            treeList.push(item)
          }
        })
      }
      return {
        treeList,
        chengduCompany
      }
    },
    resetData() { // 重置数据
      this.warehouseSearchName = ''
      this.bannerList.forEach(item => {
        if (item.placeholder === '请输入库房名称') {
          item.input = this.warehouseSearchName
        }
      })
      this.search()
    },
    async getDataList() {
      const tableData = []
      try {
        const res = await findWarehouseList({
          pageNum: this.page,
          pageSize: this.limit,
          name: this.warehouseSearchName,
          corpId: resolveCorpQueryId(
            this.targetGroupNowForm.id,
            this.queryableCorpKeys,
            this.userInfo
          )
        })
        const result = res && res.return
        const list = result && Array.isArray(result.data) ? result.data : []
        this.total = Number(result && result.total) || 0
        list.forEach((item, index) => {
          item.rowNumber = (this.page - 1) * this.limit + index + 1
          this.treeList.forEach(treeItem => {
            if (item.corpId === treeItem.id) {
              item.corpName = treeItem.name
            }
          })
          const detailMsg = item.warehouseDetails || {}
          item.addr = detailMsg.addr // 详细地址
          item.cityName = detailMsg.cityName // 城市名称
          item.latitude = detailMsg.latitude // 纬度
          item.longitude = detailMsg.longitude // 经度
          item.provinceName = detailMsg.provinceName // 省名称
          tableData.push(item)
        })
      } catch (error) {
        console.error(error)
        this.total = 0
      }
      this.tableData = tableData
      this.setTotalBanner()
    },
    setTotalBanner() {
      this.bannerList.forEach(item => {
        if (item.style === 'infoNum') {
          item.num = this.total
        }
      })
    },
    getMsgList(e) { // 页码改变时触发
      this.page = e.page
      this.limit = e.limit
      this.getDataList() // 设置显示的表格数据
    },
    clickBannerBtn(item) { // 点击banner的按钮
      if (item.buttonName === '新增') {
        this.add()
      } else if (item.buttonName === '查询') {
        this.search()
      } else if (item.buttonName === '重置') {
        this.resetData()
      } else if (item.buttonName === '导入') {
        this.upLoadDataFile()
      } else if (item.buttonName === '导出') {
        this.exportExcel()
      }
    },
    inputBannerChange(item) { // banner输入框内容改变
      if (item.placeholder === '请输入库房名称') {
        this.warehouseSearchName = item.input
      }
      this.search()
    },
    inputBannerKeyupEnter() { // banner输入框回车
      this.search()
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
      this.search()
    },
    add() { // 点击添加按钮
      this.dialogStatus = 'add'
      this.resetForm()
    },
    modify(row) { // 点击修改按钮
      this.dialogStatus = 'modify'
      this.resetForm(row)
    },
    async resetForm(row) { // 重置表单
      if (this.dialogStatus === 'add') {
        if (this.targetGroupNowForm.level < 2) {
          Message({
            message: `请选择市级公司`,
            type: 'warning',
            duration: 3 * 1000
          })
          return
        }
        this.form = {
          name: '', // 库房名称
          address: '', // 库房地址
          managerName: '', // 库房管理员
          managerUserId: 2, // 库房管理员id
          managerPhone: '', // 管理员电话
          warehouseCode: '', // 编码
          corpId: this.targetGroupNowForm.id, // 所属部门
          corpName: this.targetGroupNowForm.name, // 所属部门名称
          createUserId: 1, // 创建用户id
          addr: '', // 库房地址
          provinceName: '', // 所属省名称
          cityName: '', // 所属城市名称
          longitude: '', // 经度
          latitude: '' // 纬度
        }
      } else if (this.dialogStatus === 'modify') {
        const modifyTreeList = lodash.cloneDeep(this.modifyTreeList)
        modifyTreeList.forEach(item => {
          item.selected = false
          if (item.id === row.corpId) {
            item.selected = true
          }
        })
        this.modifyTreeList = modifyTreeList
        this.form = {
          id: row.id,
          name: row.name, // 库房名称
          address: row.address, // 库房地址
          managerName: row.managerName, // 库房管理员
          managerUserId: row.managerUserId, // 库房管理员id
          managerPhone: row.managerPhone, // 管理员电话
          warehouseCode: row.warehouseCode, // 编码
          corpId: row.corpId, // 部门id
          corpName: row.corpName, // 所属部门名称
          createUserId: row.createUserId, // 创建用户id
          addr: row.addr, // 详细地址
          provinceName: row.provinceName, // 所属省名称
          cityName: row.cityName, // 所属城市名称
          longitude: row.longitude, // 经度
          latitude: row.latitude // 纬度
        }
      } else if (this.dialogStatus === 'upload') {
        this.form = {
          fileList: []
        }
      }
      this.showDrawer = true // 表单可见
    },
    async getEquipTypeList() { // 获取库房类型列表
      const equipTypeList = []
      const res = await findEquipmentTypeList()
      let list = []
      if (res.list) {
        list = res.list
      }
      list.forEach(item => {
        equipTypeList.push({
          key: item.id, name: item.equipmentTypeName
        })
      })
      this.equipTypeList = equipTypeList
      return this.equipTypeList.length !== 0
    },
    changeUserGroup(e) { // 修改部门
      this.form.corpId = e.id
    },
    upLoadDataFile() {
      this.dialogStatus = 'upload'
      this.resetForm()
    },
    chooseMap() { // 选择地理位置
      if (this.form.longitude !== '' && this.form.latitude !== '') {
        this.mapMsg.msg = {
          lng: this.form.longitude, // 经度
          lat: this.form.latitude, // 纬度
          provinceName: this.form.provinceName, // 省份名称
          cityName: this.form.cityName, // 城市名称
          resultName: this.form.addr // 详细地址
        }
      }
      this.dialogMap = true
    },
    mapinfo(evtValue) { // 确定选择位置
      this.form.addr = evtValue.resultName // 地理位置
      this.form.provinceName = evtValue.provinceName // 所属省名称
      this.form.cityName = evtValue.cityName // 所属城市名称
      this.form.longitude = evtValue.lng // 经度
      this.form.latitude = evtValue.lat // 纬度
      this.dialogMap = false
    },
    setData() { // 确认添加或修改
      this.$refs['dataForm'].validate(async valid => {
        if (valid) {
          if (this.dialogStatus === 'add') {
            saveWarehouse(this.form).then(() => {
              this.$notify({
                title: '添加请求发送成功',
                message: `成功添加`,
                type: 'success',
                duration: 3 * 1000
              })
              this.getMsgList({
                page: this.page,
                limit: this.limit
              })
              this.showDrawer = false
            })
          } else if (this.dialogStatus === 'modify') {
            modifyWarehouse(this.form).then(res => {
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
            })
          } else if (this.dialogStatus === 'upload') {
            if (!this.form.fileList.length) {
              Message({ message: '请选择要导入的Excel文件', type: 'warning' })
              return
            }
            const importResponse = await warehouseImport({ fileList: this.form.fileList })
            const contentType = importResponse.headers?.['content-type'] || ''
            if (contentType.includes('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')) {
              exportBlobFile(importResponse)
              Message({
                message: '导入校验失败，请根据自动下载文件中的红色错误信息修改后重新导入',
                type: 'error',
                duration: 5 * 1000
              })
              return
            }
            const result = await this.parseBlobJson(importResponse.data)
            if (Number(result.code) !== 200) {
              Message({ message: result.msg || '导入失败', type: 'error' })
              return
            }
            this.$notify({
              title: '导入成功',
              message: `成功导入${result.return?.successCount || 0}条`,
              type: 'success',
              duration: 3 * 1000
            })
            this.getMsgList({
              page: this.page,
              limit: this.limit
            })
            this.showDrawer = false
          }
        }
      })
    },
    async remove(row) {
      const removeName = row.name
      const sure = await meAlert(this, {
        title: '确认删除',
        content: `此操作将永久删除${removeName}，是否继续？`
      })
      if (sure) {
        await deleteWarehouse({
          id: row.id // 库房ObjectId
        })
        Message({
          message: `${removeName}删除成功`,
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
      this.page = 1
      this.getDataList()
    },
    selectFile(data) { // 选择文件
      let key = ''
      data.fileList.forEach((item, index) => {
        if (item.size > 5 * 1024 * 1000) key = index
      })
      if (key !== '') {
        this.$notify({
          title: '上传失败',
          message: `上传文件不能大于5M`,
          type: 'warning',
          duration: 3 * 1000
        })
        this.form.fileList = []
      } else {
        this.form.fileList = data.fileList
      }
    },
    async exportExcel() { // 导出excel
      const res = await warehouseExport({
        name: this.warehouseSearchName,
        pageNum: this.page,
        pageSize: this.limit,
        corpId: resolveCorpQueryId(
          this.targetGroupNowForm.id,
          this.queryableCorpKeys,
          this.userInfo
        )
      })
      const contentType = res.headers?.['content-type'] || ''
      if (!contentType.includes('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')) {
        const result = await this.parseBlobJson(res.data)
        Message({ message: result.msg || '导出失败', type: 'error' })
        return
      }
      exportBlobFile(res)
    },
    async parseBlobJson(blob) {
      try {
        const text = await blob.text()
        return JSON.parse(text)
      } catch {
        return {}
      }
    },
    async uploadTemp() {
      const res = await warehouseImportTemplate()
      if (res.data?.type?.includes('json')) {
        const result = await this.parseBlobJson(res.data)
        Message.error(result.msg || '模板下载失败')
        return
      }
      exportBlobFile(res)
    }
  }
}
</script>

<style lang="stylus" scoped>
searchContentHeight = 50px
.treeContentCss {
  float left
  width 15%
  height 100%
  padding 5px
  border-right 1px solid #EAEDF7
  background-color white
  .treeSearchCss {
    height searchContentHeight
  }
}
.dataContentCss {
  float right
  width 85%
  height 100%
  .scrollbarCss {
    height calc(100% - 50px)
    padding 10px 20px 0 10px
  }
}

.mapContentCss {
  width 100%
  height 60vh
}
</style>
