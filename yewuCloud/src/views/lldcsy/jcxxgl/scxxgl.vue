<template>
  <div class="contentCss">
    <el-scrollbar class="treeContentCss allheight">
      <MyElTree :navi-name-list="treeList" @click-tree="chooseTargetgroup" />
    </el-scrollbar>
    <div class="dataContentCss">
      <Banner
        :list="bannerList"
        @click-button="clickBannerBtn"
        @input-change="inputBannerChange"
        @input-keyup-enter="inputBannerKeyupEnter"
      />
      <el-scrollbar class="scrollbarCss">
        <el-table
          :key="tableKey"
          height="535"
          :data="tableData"
          fit
          highlight-current-row
        >
          <el-table-column label="序号" width="50" align="center">
            <template #default="scope">
              <span>{{ scope.row.xh }}</span>
            </template>
          </el-table-column>
          <el-table-column label="电池ID" width="120" align="center">
            <template #default="scope">
              <span>{{ scope.row.equipmentCode }}</span>
            </template>
          </el-table-column>
          <el-table-column label="电池类型" width="120" align="center">
            <template #default="scope">
              <span>{{ scope.row.equipmentTypeName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="生产厂家" width="180" align="center">
            <template #default="scope">
              <span>{{ scope.row.manufacturer }}</span>
            </template>
          </el-table-column>
          <el-table-column label="生产日期" width="180" align="center">
            <template #default="scope">
              <span>{{ scope.row.manufacturerTime }}</span>
            </template>
          </el-table-column>
          <el-table-column label="批次" width="140" align="center">
            <template #default="scope">
              <span>{{ scope.row.batch }}</span>
            </template>
          </el-table-column>
          <el-table-column label="规格" width="100" align="center">
            <template #default="scope">
              <span>{{ scope.row.specifications }}</span>
            </template>
          </el-table-column>
          <el-table-column label="所属公司" width="140" align="center">
            <template #default="scope">
              <span>{{ scope.row.corpName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="BMS厂家" width="140" align="center">
            <template #default="scope">
              <span>{{ scope.row.bmsManufacturer }}</span>
            </template>
          </el-table-column>
          <el-table-column label="BMS版本" width="140" align="center">
            <template #default="scope">
              <span>{{ scope.row.bmsVersion }}</span>
            </template>
          </el-table-column>
          <el-table-column
            label="操作"
            align="center"
            class-name="small-padding fixed-width"
            width="160"
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
    <elDrawer v-model="showDrawer" :title="textMap[dialogStatus]" :wrapper-closable="false" width="500">
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
            <el-form-item label="电池ID" prop="equipmentCode">
              <el-input v-model="form.equipmentCode" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="电池类型" prop="equipmentTypeName">
              <el-input v-model="form.equipmentTypeName" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="生产厂家" prop="manufacturer">
              <el-input v-model="form.manufacturer" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="生产日期" prop="manufacturerTime">
              <el-date-picker
                v-model="form.manufacturerTime"
                type="datetime"
                placeholder="选择日期时间"
              />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="批次" prop="batch">
              <el-input v-model="form.batch" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="规格" prop="specifications">
              <el-input v-model="form.specifications" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="BMS厂家" prop="bmsManufacturer">
              <el-input v-model="form.bmsManufacturer" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="BMS版本" prop="bmsVersion">
              <el-input v-model="form.bmsVersion" />
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
    </elDrawer>

  </div>
</template>

<script>
import { Message } from '@/utils/element-shim'
import Pagination from '@/components/Pagination' // 分页组件
import Banner from '@/components/ccpcComp/Banner.vue'
import MyElTree from '@/components/a_myComp/elTree/MyElTree.vue' // 左侧导航
import UploadFile from '@/components/a_myComp/uploadFile/UploadFile.vue'
import { corpList } from '@/api/user.js'
import { meAlert, objectSort, formatDateString, exportBlobFile } from '@/utils/utils.js'
import { processDrawList } from '@/utils/ccpcData.js'
import { findEquipmentTypeList } from '@/api/basicData.js'
import { getStreamList } from '@/api/stream.js'
import { findSiteListExactMatch } from '@/api/siteMana.js'
import { findEquipmentListExactMatch, saveEquipment, deleteEquipment, modifyEquipment } from '@/api/equipMana.js'
import { parseUserInfo } from '@/utils/safe-json.js'
import { equipmentExport, equipmentImport, equipmentImportTemplate } from '@/api/lldcsy.js'
import { chinaRegions } from '@/components/a_myComp/mapComplex/index.js'
import lodash from 'lodash'
import totalImage from '@/assets/imgs/总量.png'

export default {
  components: {
    MyElTree,
    Pagination,
    Banner,
    UploadFile
  },
  data() {
    return {
      userInfo: parseUserInfo({}),

      bannerList: [
        { style: 'infoNum', icon: totalImage, title: '电池总量', num: '0', unit: '个' },
        { style: 'input', placeholder: '请输入电池ID', input: '' },
        { style: 'input', placeholder: '请输入电池类型', input: '' },
        { style: 'input', placeholder: '请输入电池生产厂家', input: '' },
        { style: 'input', placeholder: 'BMS厂家', input: '' },
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

      searchEquipmentCode: '', // 查询电池id
      searchEquipmentTypeName: '', // 查询电池类型
      searchManufacturer: '', // 查询电池生产厂家
      searchBmsManufacturer: '', // 查询电池生产厂家

      showDrawer: false, // 抽屉是否显示
      textMap: { // 表单弹窗标题
        add: '添加电池',
        modify: '修改电池',
        upload: '导入excel'
      },
      dialogStatus: '', // 表单状态值为'add'或'modify' ，与 textMap 对应
      rules: {
        equipmentCode: [{ required: true, trigger: 'blur', message: '电池ID不能为空' }],
        equipmentTypeName: [{
          validator: (rule, value, callback) => {
            if (!value || !value.trim()) {
              callback(new Error('电池类型不能为空'))
              return
            }
            callback()
          },
          trigger: 'blur'
        }]
      },

      allTableData: [], // 全部数据
      tableData: [],
      form: {
        equipmentName: '', // 电池名称*(String)
        belongToSiteId: '', // 站址ObjectId
        belongToSiteName: '', // 站址名称(String)
        streamId: '', // 电池对应的数据池id
        apiKey: '', // 电池实际使用的apiKey
        apiKeys: [],
        equipmentTypeId: '', // 电池类型ObjectId
        equipmentTypeName: '', // 电池类型名称(String)
        dataDetailsId: '', // 数据详情ObjectId
        dataDetailSerialNumber: [], // 数据详情序列号(自定义id)以上两个参数可以不传,要传一定都传(add 20201114)
        codeRule: '1', // 编码规则(用户自定义序列号长度)
        engageInternetStatus: 1, // 并网状态
        concernUserIds: [], // 关注用户
        controlUserIds: [], // 可控制监控的用户
        equipmentDetails: {
        },

        equipmentCode: '', // 设备编码
        warehouseId: 1, // 库房id
        warehouseName: '', // 库房名称
        manufacturer: '', // 生产厂家
        manufacturerTime: '', // 生产日期
        batch: '', // 批次
        specifications: '', // 规格
        corpId: 1, // 所属公司
        bmsManufacturer: '', // BMS生产厂家
        bmsVersion: '' // BMS版本
      },

      adminList: [], // 管理员列表
      processDrawList, // 工艺图模板列表

      equipTypeList: [], // 电池类型列表
      serNumList: [ // 编码规则列表
        { key: '6', name: '规则6' },
        { key: '5', name: '规则5' },
        { key: '4', name: '规则4' },
        { key: '3', name: '规则3' },
        { key: '2', name: '规则2' },
        { key: '1', name: '规则1' }
      ],
      siteList: [], // 电池列表
      apiKeys: [], // 电池实际使用的apiKey列表

      tbjgTypeList: [ // 填报间隔类型列表
        { key: '小时', name: '小时' },
        { key: '日', name: '日' },
        { key: '月', name: '月' },
        { key: '年', name: '年' }
      ],

      streamList: [], // 数据池列表

      chinaRegions // 省份及其下辖的市
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
      const allTableData = await this.getAllDataList() // 获取全部数据列表
      await this.getTreeList(allTableData)
      this.getDataList()
    },
    async getAllDataList() { // 获取全部数据列表
      const res = await findEquipmentListExactMatch({
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
    async getTreeList(allTableData) {
      const res = await corpList() // 列表（用户部门树）
      const list = res.return.data
      const { treeList, cdMsg } = this.setTreeExpand({
        list,
        allTableData
      })

      this.treeList = objectSort(treeList, 'level')
      this.treeList.splice(3, 0, cdMsg)
      const fristTree = treeList[0]
      if (fristTree) {
        fristTree.selected = true
        this.targetGroupNowForm = fristTree
      }
      this.modifyTreeList = lodash.cloneDeep(this.treeList)
    },
    setTreeExpand(params) {
      const { list, allTableData } = params
      let cdMsg = {} // 成都市分公司信息
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
            cdMsg = item
          } else {
            treeList.push(item)
          }
        })
      }
      return {
        treeList,
        cdMsg
      }
    },
    async getStreamListMsg() { // 获取数据池列表
      const streamList = []
      const res = await getStreamList({
        page: 1,
        limit: 100000,
        search: ''
      })
      const list = res.return.stream
      if (list) {
        list.forEach(item => {
          item.key = item.id
          streamList.push(item)
        })
      }
      this.streamList = streamList
    },
    resetData() { // 重置数据
      this.searchEquipmentCode = ''
      this.searchEquipmentTypeName = ''
      this.searchManufacturer = ''
      this.searchBmsManufacturer = ''
      this.bannerList.forEach(item => {
        switch (item.placeholder) {
          case '请输入电池ID':
            item.input = this.searchEquipmentCode
            break
          case '请输入电池类型':
            item.input = this.searchEquipmentTypeName
            break
          case '请输入电池生产厂家':
            item.input = this.searchManufacturer
            break
          case 'BMS厂家':
            item.input = this.searchBmsManufacturer
            break
        }
      })
      this.search()
    },
    async getDataList() {
      await this.getStreamListMsg()
      const tableData = []
      const res = await findEquipmentListExactMatch({
        corpId: this.targetGroupNowForm.id,
        pageNum: this.page,
        pageSize: this.limit,
        equipmentCode: this.searchEquipmentCode,
        equipmentTypeName: this.searchEquipmentTypeName,
        manufacturer: this.searchManufacturer, // 生产厂家
        bmsManufacturer: this.searchBmsManufacturer // BMS生产厂家
      })
      if (res.return && res.return.data) {
        const list = res.return.data
        this.total = res.return.total
        list.forEach((item, index) => {
          item.xh = (this.page - 1) * this.limit + index + 1
          if (item.manufacturerTime !== '-') {
            item.manufacturerTime = formatDateString(item.manufacturerTime, 'formatDate')
          }
          item.streamId = ''
          if (item.equipmentDetails.streamId) {
            item.streamId = item.equipmentDetails.streamId
            this.streamList.forEach(streamItem => {
              if (item.streamId === streamItem.id) {
                item.apiKey = streamItem.apiKey
              }
            })
          }
          this.treeList.forEach(treeItem => {
            if (treeItem.id === item.corpId) {
              item.corpName = treeItem.name
            }
          })
          tableData.push(item)
        })
      } else {
        this.total = 0
      }
      this.bannerList.forEach(item => {
        if (item.style === 'infoNum') {
          item.num = this.total
        }
      })
      this.tableData = tableData
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
        this.getDataList()
      } else if (item.buttonName === '重置') {
        this.resetData()
      } else if (item.buttonName === '导入') {
        this.upLoadDataFile()
      } else if (item.buttonName === '导出') {
        this.exportExcel()
      }
    },
    inputBannerChange(item) { // banner输入框内容改变
      if (item.placeholder === '请输入电池ID') {
        this.searchEquipmentCode = item.input
      } else if (item.placeholder === '请输入电池类型') {
        this.searchEquipmentTypeName = item.input
      } else if (item.placeholder === '请输入电池生产厂家') {
        this.searchManufacturer = item.input
      } else if (item.placeholder === 'BMS厂家') {
        this.searchBmsManufacturer = item.input
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
          equipmentName: '', // 电池名称*(String)
          equipmentTypeId: 0, // 自由填写的电池类型不依赖类型字典
          equipmentTypeName: '', // 电池类型名称(String)
          dataDetailsId: '', // 数据详情ObjectId
          dataDetailSerialNumber: [], // 数据详情序列号(自定义id)以上两个参数可以不传,要传一定都传(add 20201114)
          codeRule: '1', // 编码规则(用户自定义序列号长度)
          engageInternetStatus: 1, // 并网状态
          concernUserIds: [1], // 关注用户
          controlUserIds: [1], // 可控制监控的用户
          equipmentDetails: {
            streamId: '' // 电池对应的数据池id
          },

          equipmentCode: '', // 设备编码
          warehouseId: 1, // 库房id
          warehouseName: '', // 库房名称
          manufacturer: '', // 生产厂家
          manufacturerTime: '', // 生产日期
          batch: '', // 批次
          specifications: '', // 规格
          corpId: this.targetGroupNowForm.id, // 所属部门
          corpName: this.targetGroupNowForm.name, // 所属部门名称
          bmsManufacturer: '', // BMS生产厂家
          bmsVersion: '' // BMS版本
        }
      } else if (this.dialogStatus === 'modify') {
        this.form = {
          id: row.id,
          equipmentName: row.equipmentName, // 电池名称*(String)
          belongToSiteId: row.belongToSiteId, // 站址ObjectId
          belongToSiteName: row.belongToSiteName, // 站址名称(String)
          streamId: row.streamId, // 电池对应的数据池id
          apiKey: row.apiKey, // 客户实际使用的apiKey
          apiKeys: row.apiKeys,
          equipmentTypeId: row.equipmentTypeId, // 电池类型ObjectId
          equipmentTypeName: row.equipmentTypeName, // 电池类型名称(String)
          dataDetailsId: row.dataDetailsId, // 数据详情ObjectId
          dataDetailSerialNumber: row.dataDetailSerialNumber, // 数据详情序列号(自定义id)以上两个参数可以不传,要传一定都传(add 20201114)
          codeRule: row.codeRule, // 编码规则(用户自定义序列号长度)
          concernUserIds: row.concernUserIds, // 关注用户
          engageInternetStatus: row.engageInternetStatus, // 并网状态
          controlUserIds: row.controlUserIds, // 可控制监控的用户
          equipmentDetails: row.equipmentDetails, // 电池各种详细属性

          equipmentCode: row.equipmentCode, // 设备编码
          warehouseId: row.warehouseId, // 库房id
          warehouseName: row.warehouseName, // 库房名称
          manufacturer: row.manufacturer, // 生产厂家
          manufacturerTime: row.manufacturerTime, // 生产日期
          batch: row.batch, // 批次
          specifications: row.specifications, // 规格
          corpId: row.corpId, // 所属公司
          corpName: row.corpName, // 所属部门名称
          bmsManufacturer: row.bmsManufacturer, // BMS生产厂家
          bmsVersion: row.bmsVersion // BMS版本
        }
      } else if (this.dialogStatus === 'upload') {
        this.form = {
          fileList: []
        }
      }
      this.showDrawer = true // 表单可见
    },
    async getEquipTypeList() { // 获取电池类型列表
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
    async getSiteList() { // 获取电池列表
      const tableData = []
      const res = await findSiteListExactMatch({
        pageNum: 1,
        pageSize: 10000,
        filterFunc: ''
      })
      if (res.return) {
        const list = res.return.data
        if (list) {
          list.forEach(item => {
            item.key = item.id
            item.name = item.siteName
            tableData.push(item)
          })
        }
      }
      this.siteList = tableData
      return this.siteList.length !== 0
    },
    streamIdChange() { // 数据池id改变时
      const streamId = this.form.streamId
      this.form.equipmentDetails.streamId = streamId

      this.streamList.forEach(item => {
        if (streamId === item.key) {
          this.form.apiKey = item.apiKey
        }
      })
    },
    upLoadDataFile() {
      this.dialogStatus = 'upload'
      this.resetForm()
    },
    changeUserGroup(e) { // 修改部门
      this.form.corpId = e.id
    },
    siteChange() { // 修改站址
      this.siteList.forEach(item => {
        if (item.key === this.form.belongToSiteId) {
          this.form.belongToSiteName = item.name
        }
      })
    },
    /**
     * 新增或修改数据
     * @returns {Promise<void>}
     */
    async setData() { // 确认添加或修改
      const valid = await this.$refs['dataForm'].validate()
      if (valid) {
        if (this.dialogStatus === 'add' || this.dialogStatus === 'modify') {
          this.form.equipmentCode = this.form.equipmentCode.trim()
          this.form.equipmentTypeName = this.form.equipmentTypeName.trim()
          if (this.form.equipmentName === '') {
            this.form.equipmentName = this.form.equipmentCode
          }
        }
        if (this.dialogStatus === 'add') {
          await saveEquipment(this.form)
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
        } else if (this.dialogStatus === 'modify') {
          await modifyEquipment(this.form)
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
        } else if (this.dialogStatus === 'upload') {
          if (!this.form.fileList.length) {
            Message({ message: '请选择要导入的Excel文件', type: 'warning' })
            return
          }
          const importResponse = await equipmentImport({ fileList: this.form.fileList })
          const contentType = importResponse.headers?.['content-type'] || ''
          if (contentType.includes('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')) {
            exportBlobFile(importResponse)
            Message({
              message: '导入校验失败，请根据自动下载文件中的红色错误信息修改后重新导入',
              type: 'error',
              duration: 5 * 1000
            })
          } else {
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
      }
    },
    async remove(row) {
      const removeName = row.equipmentName
      const sure = await meAlert(this, {
        title: '确认删除',
        content: `此操作将永久删除${removeName}，是否继续？`
      })
      if (sure) {
        await deleteEquipment({
          id: row.id // 电池ObjectId
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
      const res = await equipmentExport({
        corpId: this.targetGroupNowForm.id,
        pageNum: this.page,
        pageSize: this.limit,
        equipmentCode: this.searchEquipmentCode,
        equipmentTypeName: this.searchEquipmentTypeName,
        manufacturer: this.searchManufacturer, // 生产厂家
        bmsManufacturer: this.searchBmsManufacturer // BMS生产厂家
      })
      const contentType = res.headers?.['content-type'] || ''
      if (!contentType.includes('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')) {
        const result = await this.parseBlobJson(res.data)
        Message({ message: result.msg || '导出失败', type: 'error' })
        return
      }
      exportBlobFile(res)
    },
    async uploadTemp() {
      const res = await equipmentImportTemplate()
      const contentType = res.headers?.['content-type'] || ''
      if (!contentType.includes('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')) {
        const result = await this.parseBlobJson(res.data)
        Message({ message: result.msg || '模板下载失败', type: 'error' })
        return
      }
      exportBlobFile(res)
    },
    async parseBlobJson(blob) {
      const text = await blob.text()
      return JSON.parse(text)
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
</style>
