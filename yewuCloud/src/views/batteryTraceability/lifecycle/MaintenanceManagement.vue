<template>
  <div v-loading="showLoading" class="contentCss">
    <div class="dataContentCss">
      <Banner
        :list="bannerList"
        @click-button="clickBannerBtn"
        @input-change="inputBannerChange"
        @input-keyup-enter="inputBannerKeyupEnter"
        @select-change="selectBannerChange"
        @time-change="timeChange"
      />
      <el-scrollbar class="scrollbarCss">
        <el-table
          :key="tableKey"
          height="615"
          :data="tableData"
          fit
          highlight-current-row
          :empty-text="queryError || '无维修记录'"
        >
          <el-table-column label="序号" align="center" width="50">
            <template #default="scope">
              <span>{{ scope.row.rowNumber }}</span>
            </template>
          </el-table-column>
          <el-table-column label="电池ID" align="center" width="120">
            <template #default="scope">
              <span>{{ scope.row.batteryCode }}</span>
            </template>
          </el-table-column>
          <el-table-column label="电池类型" align="center">
            <template #default="scope">
              <span>{{ scope.row.equipmentTypeName || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="站址名称" align="center">
            <template #default="scope">
              <span>{{ scope.row.siteSerialNumber }}</span>
            </template>
          </el-table-column>
          <el-table-column label="故障描述" align="center">
            <template #default="scope">
              <span>{{ scope.row.faultDescription }}</span>
            </template>
          </el-table-column>
          <el-table-column label="图片" align="center">
            <template #default="scope">
              <img style="width: 50px;height: 50px;" :src="scope.row.imgUrl">
            </template>
          </el-table-column>
          <el-table-column label="运维时间" align="center">
            <template #default="scope">
              <span>{{ scope.row.maintenanceTime }}</span>
            </template>
          </el-table-column>
          <el-table-column label="标识" align="center">
            <template #default="scope">
              <span>{{ scope.row.bs }}</span>
            </template>
          </el-table-column>
          <el-table-column
            label="操作"
            align="center"
            class-name="small-padding fixed-width"
            width="220"
          >
            <template #default="scope">
              <el-button
                type="primary"
                size="small"
                @click="goDetail(scope.row)"
              >查看详情</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-row>
          <el-pagination
            :current-page="currentPage"
            :page-sizes="[10, 20, 30, 40]"
            :page-size="pageSize"
            layout="total, sizes, prev, pager, next, jumper"
            :total="total"
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
          />
        </el-row>
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
          <el-col :span="24">
            <el-form-item label="电池ID" prop="batteryCode">
              <el-input v-model="form.batteryCode" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="所属公司" prop="siteSerialNumber">
              <el-input v-model="form.companyName" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="故障描述" prop="faultDescription">
              <el-input v-model="form.faultDescription" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="故障详情" prop="siteSerialNumber">
              <el-input v-model="form.faultDetail" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="图片">
              <UploadFile :limit="9" :file-name-list="form.fileList" @select-file="selectFile" />
              <div class="allImgContentCss">
                <div v-for="(fielItem, fielIndex) in form.files" :key="fielIndex" class="imgContentCss">
                  <el-image
                    class="imgCss"
                    :src="fielItem.AFilePath"
                    fit="fill"
                    :preview-src-list="([fielItem.AFilePath])"
                  />
                </div>
              </div>
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
              <UploadFile :limit="1" :file-name-list="form.fileList" @select-file="selectMsgFile" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <div class="fixedBottomCss">
        <el-button @click="showDrawer = false">取消</el-button>
        <el-button type="primary" @click="dialogStatus === 'upload'? saveFile() : addNewMaintain()">确定</el-button>
      </div>
    </elDrawer>

    <el-dialog v-model="dialogDetails" title="" width="40%">
      <MaintenanceDetails v-if="dialogDetails" :form-data="detail" />
    </el-dialog>

  </div>
</template>

<script>
import { Message } from '@/utils/element-shim'
import Banner from '@/components/ccpcComp/Banner.vue'
import UploadFile from '@/components/a_myComp/uploadFile/UploadFile.vue'
import { fileIsImg, objectSort, exportBlobFile } from '@/utils/utils.js'
import { processDrawList } from '@/utils/ccpcData.js'
import { findEquipmentTypeList, findSiteGroupArr, findEquipmentmitrcd,
  uploadEquipmentmitrcdImage, getCurrentDateTimeString, addEquipmentmitrcd, findEquipmentcirlog } from '@/api/basicData.js'
import { findSiteListExactMatch } from '@/api/siteMana.js'
import { findEquipmentListExactMatch } from '@/api/equipMana.js'
import { findAlarmUsers, deletetrigger, deleteAlarmUsers } from '@/api/trigger.js'
import MaintenanceDetails from './comp/MaintenanceDetails.vue'
import { findSelf, corpList } from '@/api/user.js'
import { equipmentmitrcdExport, equipmentmitrcdImport } from '@/api/batteryTraceability.js'
import totalImage from '@/assets/imgs/总量.png'
import { loadMaintenanceTable, readMaintenanceFilters, maintenanceErrorMessage } from './maintenance-data.js'
import refreshPage from './refresh-page.js'

export default {
  mixins: [refreshPage],
  // 保留改名前的组件名称，使既有路由缓存继续生效。
  name: 'ywgl',
  components: {
    Banner,
    UploadFile,
    MaintenanceDetails
  },
  data() {
    const validateESN = (rule, value, callback) => { // 采集周期校验
      const equipmentSerialNumber = []
      value.forEach(item => {
        if (parseInt(this.form.codeRule) >= parseInt(item.id)) {
          equipmentSerialNumber.push({
            id: item.id, symbel: item.symbel
          })
        }
      })
      let isAllNotNull = true // 是否全部不为空
      equipmentSerialNumber.forEach(item => {
        if (item.symbel === '') {
          isAllNotNull = false
        }
      })
      if (isAllNotNull) { // 全部不为空
        callback()
      } else {
        callback(new Error('库房ID不能为空'))
      }
    }

    const validateDataReportInterval = (rule, value, callback) => { // 数据填表校验规则
      let max = 10000
      switch (this.form.equipmentDetails.sjtbTbjgType) {
        case '日': max = 28; break
        case '月': max = 11; break
      }
      if (this.form.equipmentDetails.sjtbTbjg > max) {
        callback(new Error(`${this.form.equipmentDetails.sjtbTbjgType}填报间隔最大值为${max}`))
      } else {
        callback()
      }
    }
    const validateInspectionReportInterval = (rule, value, callback) => { // 数据填表校验规则
      let max = 10000
      switch (this.form.equipmentDetails.xjtbTbjgType) {
        case '日': max = 28; break
        case '月': max = 11; break
      }
      if (this.form.equipmentDetails.xjtbTbjg > max) {
        callback(new Error(`${this.form.equipmentDetails.xjtbTbjgType}填报间隔最大值为${max}`))
      } else {
        callback()
      }
    }
    return {
      showLoading: false,
      queryError: '',
      submitting: false, // 新增与导入共用提交锁，避免连点和切换表单重复写入。
      bannerList: [
        { style: 'infoNum', icon: totalImage, title: '运维总次数', num: '0', unit: '个' },
        { style: 'infoNum', icon: totalImage, title: '运维总电池数', num: '0', unit: '个' },
        { style: 'input', placeholder: '请输入电池ID', input: '', width: '10vw' },
        { style: 'input', placeholder: '请输入电池类型', input: '', width: '10vw' },
        { style: 'select', placeholder: '运维次数大于', value: '', list: [
          { key: '0', name: '全部运维次数电池' },
          { key: '5', name: '5' },
          { key: '10', name: '10' },
          { key: '20', name: '20' }
        ], width: '10vw' },
        { style: 'selectTimeSlot', name: '选择时间段', timeDateArr: [],
          startPlaceholder: '开始日期', endPlaceholder: '结束日期', datePickerType: 'datetimerange',
          width: '24vw', size: 'default'
        },
        { style: 'button', buttonName: '查询', type: 'primary' },
        { style: 'button', buttonName: '重置', type: '' }
      ],

      targetGroupNowForm: { // 当前部门导航树信息
        level: 0, // 目录级别
        parentId: 0, // 父id
        id: 1, // 自己的id
        name: '' // 名称
      },
      treeList: [ // 部门列表
        // { level: 0, parentId: 0, id: 1, name: '企业' }
      ],

      searchTree: '', // 查询tree节点
      allGroupList: [], // 所以分组列表
      groupList: [], // 分组列表
      fileList: [],
      tableKey: 1, // 参数值变化，表格对应改变
      total: 0, // 数据总条数
      limit: 10, // 每次加载数据条数
      detail: {},
      currentPage: 1,
      pageSize: 10,

      searchSiteList: [], // 用于查询的站址列表
      searchSiteId: '', // 查询的站址
      searchEquipName: '', // 查询设备

      showDrawer: false, // 抽屉是否显示
      textMap: { // 表单弹窗标题
        add: '添加运维',
        modify: '修改运维'
      },
      dialogStatus: '', // 表单状态值为'add'或'modify' ，与 textMap 对应
      rules: {
        equipmentName: [{ required: true, trigger: 'blur', message: '库房名称不能为空' }],
        equipmentSerialNumber: [{ required: true, trigger: 'blur', validator: validateESN }],
        sjtbTbjg: [{ required: true, trigger: 'blur', validator: validateDataReportInterval }],
        xjtbTbjg: [{ required: true, trigger: 'blur', validator: validateInspectionReportInterval }]
      },

      allTableData: [], // 全部数据
      tableData: [],
      form: {},

      adminList: [], // 管理员列表
      processDrawList, // 工艺图模板列表

      startTime: '', // 开始时间
      endTime: '', // 结束时间

      equipmentCode: '',

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

      dialogDetails: false
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
  methods: {
    async init() {
      // 运维页没有部门树，不再因无关的部门接口失败而阻断运维查询。
    },
    async getTreeList() {
      const treeList = []
      const res = await corpList() // 列表（用户部门树）
      const list = res.return.data
      if (list) {
        list.forEach(item => {
          item.name = item.corpName
          if (item.name !== '中国铁塔股份有限公司' && item.name !== '四川省分公司') {
            item.expand = false // 不展开
          }
          treeList.push(item)
        })
      }
      const fristTree = treeList[0]
      if (fristTree) {
        fristTree.selected = true
        this.targetGroupNowForm = fristTree
      }
      this.treeList = objectSort(treeList, 'level')
    },
    // my添加
    clickBannerBtn(item) {
      if (item.buttonName === '新增') {
        return this.add()
      } else if (item.buttonName === '查询') {
        return this.search()
      } else if (item.buttonName === '重置') {
        return this.resetData()
      } if (item.buttonName === '导入') {
        this.upLoadDataFile()
      } else if (item.buttonName === '导出') {
        this.exportExcel()
      }
    },
    // 添加新的运维内容
    async addNewMaintain() {
      if (this.submitting) return
      // 固定本次表单快照，等待请求时编辑字段不能改变已开始的提交内容。
      const form = { ...this.form }
      const equipmentCode = String(form.batteryCode || '').trim()
      const files = Array.isArray(form.fileList) ? form.fileList.filter(Boolean) : []
      if (!equipmentCode) {
        Message({ message: '请输入电池ID', type: 'warning', duration: 1000 })
        return
      }
      if (files.length === 0) {
        Message({ message: '请至少选择一张运维图片', type: 'warning', duration: 1000 })
        return
      }

      this.submitting = true
      try {
        const userRes = await findSelf({})
        const user = userRes?.return?.user
        if (!user?.id) throw new Error('当前登录用户信息无效')

        // 电池 ID 以设备编码为主做精确匹配；仅对旧数据没有编码的设备兼容名称匹配。
        let equipmentResponse = await findEquipmentListExactMatch({
          equipmentCode, pageNum: 1, pageSize: 10000
        })
        let equipmentRows = Array.isArray(equipmentResponse?.return?.data)
          ? equipmentResponse.return.data : []
        const exactCodeRows = equipmentRows.filter(item => String(item.equipmentCode || '').trim() === equipmentCode)
        if (exactCodeRows.length > 1) throw new Error('设备编码对应多个设备，请先检查设备资料')
        let equipment = exactCodeRows[0]
        if (!equipment) {
          equipmentResponse = await findEquipmentListExactMatch({
            equipmentName: equipmentCode, pageNum: 1, pageSize: 10000
          })
          equipmentRows = Array.isArray(equipmentResponse?.return?.data)
            ? equipmentResponse.return.data : []
          const exactNameRows = equipmentRows.filter(item => {
            const code = String(item.equipmentCode || '').trim()
            return (!code || code === '-') && String(item.equipmentName || '').trim() === equipmentCode
          })
          if (exactNameRows.length > 1) throw new Error('电池ID对应多个设备，请使用设备编码')
          equipment = exactNameRows[0]
        }
        if (!equipment?.id) throw new Error('电池ID不存在')

        // 新设备接口直接给出当前状态时，以状态为准，允许没有历史流转日志的存量设备上报。
        // 只有状态缺失或为未知值时，才从流转记录中按时间、ID取最新一条。
        const currentStatus = Number(equipment.positionStatus)
        const hasCurrentStatus = Number.isInteger(currentStatus) && currentStatus >= 1 && currentStatus <= 6
        if (!hasCurrentStatus) {
          const circulationResponse = await findEquipmentcirlog({ equipmentId: equipment.id })
          const circulationRows = Array.isArray(circulationResponse?.return?.data)
            ? circulationResponse.return.data : []
          const toNumber = value => {
            if (value == null || value === '') return Number.NEGATIVE_INFINITY
            const number = Number(value)
            if (Number.isFinite(number)) return number
            const date = new Date(value).getTime()
            return Number.isFinite(date) ? date : Number.NEGATIVE_INFINITY
          }
          const latest = circulationRows.slice().sort((left, right) => {
            const leftTime = toNumber(left.createTime ?? left.create_time ?? left.operateTime ?? left.time)
            const rightTime = toNumber(right.createTime ?? right.create_time ?? right.operateTime ?? right.time)
            if (leftTime !== rightTime) return rightTime - leftTime
            return toNumber(right.id) - toNumber(left.id)
          })[0]
          if (!latest || Number(latest.operateType) !== 3) throw new Error('电池没上站')
        } else if (currentStatus !== 3) {
          throw new Error('电池没上站')
        }

        // 页面允许最多 9 张图片，全部上传并保存，首张继续写入旧版 imageUrl 字段。
        const imageResponses = await Promise.all(files.map(file => uploadEquipmentmitrcdImage({ file })))
        const imageUrls = imageResponses.map(response => {
          if (Number(response?.code) !== 200 || !response?.data?.url) {
            throw new Error(response?.msg || '运维图片上传失败')
          }
          return response.data.url
        })
        const data = {
          equipmentId: equipment.id,
          maintainUserId: user.id,
          maintainUserName: user.username,
          maintainTitle: form.faultDescription || '',
          maintainContent: form.faultDetail || '',
          maintainPhotos: imageUrls,
          maintainDetails: {
            create_time: getCurrentDateTimeString(),
            imageUrl: imageUrls[0],
            bs: '业务云上报',
            province_company: form.companyName || '',
            fault_detail: form.faultDetail || ''
          }
        }
        const result = await addEquipmentmitrcd(data)
        if (Number(result?.code) !== 200) throw new Error(result?.msg || '运维记录添加失败')
        this.showDrawer = false
        Message({ message: '添加成功', type: 'success', duration: 1000 })
        await this.search()
      } catch (error) {
        Message({
          message: maintenanceErrorMessage(error, '运维记录添加失败，请检查网络或稍后重试'),
          type: 'warning',
          duration: 2000
        })
      } finally {
        this.submitting = false
      }
    },
    // 输入、次数、日期和按钮统一进入同一条查询链，避免互相覆盖筛选条件。
    selectBannerChange() {
      return this.search()
    },
    timeChange() {
      return this.search()
    },
    inputBannerChange() {
      // 文本通过组件双向绑定更新；点击查询或回车时才提交，避免逐字请求。
    },
    inputBannerKeyupEnter() {
      return this.search()
    },
    search() {
      this.currentPage = 1
      return this.getEquipmentmitrcd()
    },
    getDataList() {
      return this.search()
    },
    resetData() {
      this.bannerList.forEach(item => {
        if (item.style === 'input') item.input = ''
        if (item.style === 'select') item.value = ''
        if (item.style === 'selectTimeSlot') item.timeDateArr = []
      })
      this.equipmentCode = ''
      this.startTime = ''
      this.endTime = ''
      return this.search()
    },
    handleCurrentChange(value) {
      if (this.currentPage === value) return
      this.currentPage = value
      return this.getEquipmentmitrcd()
    },
    handleSizeChange(value) {
      this.pageSize = value
      this.currentPage = 1
      return this.getEquipmentmitrcd()
    },
    async getEquipmentmitrcd() {
      if (!this.pageActive) return
      const requestId = ++this.queryVersion
      const isCurrent = () => this.pageActive && requestId === this.queryVersion
      this.showLoading = true
      this.queryError = ''
      try {
        const filters = readMaintenanceFilters(this.bannerList)
        filters.pageNum = this.currentPage
        filters.pageSize = this.pageSize
        this.equipmentCode = filters.equipmentCode
        this.startTime = filters.startTime
        this.endTime = filters.endTime
        const result = await loadMaintenanceTable({
          findRecords: findEquipmentmitrcd,
          findEquipment: findEquipmentListExactMatch,
          findSite: findSiteListExactMatch
        }, filters, isCurrent)
        if (!isCurrent() || !result) return
        this.total = result.total
        // 删除记录或并发更新导致当前页越界时，回到实际最后一页重新请求。
        const maxPage = Math.max(1, Math.ceil(this.total / this.pageSize))
        if (this.currentPage > maxPage) {
          this.currentPage = maxPage
          return this.getEquipmentmitrcd()
        }
        this.tableData = result.rows
        this.bannerList[0].num = String(this.total)
        this.bannerList[1].num = String(result.equipmentCount)
      } catch (error) {
        if (!isCurrent()) return
        this.queryError = maintenanceErrorMessage(error)
        this.tableData = []
        this.total = 0
        this.bannerList[0].num = '0'
        this.bannerList[1].num = '0'
        Message({ message: this.queryError, type: 'error' })
      } finally {
        if (isCurrent()) this.showLoading = false
      }
    },
    async getSiteGroupArr() { // 获取站址分组列表
      const res = await findSiteGroupArr()
      if (res.siteGroupArr && res.siteGroupArr.siteGroupArrayList) {
        const allGroupList = []
        res.siteGroupArr.siteGroupArrayList.forEach(item => {
          allGroupList.push({
            groupId: item.id,
            level: item.level,
            parentId: item.parentId,
            treeId: item.treeId,
            id: item.treeId,
            siteGroupName: item.siteGroupName,
            name: item.siteGroupName
          })
        })
        this.allGroupList = allGroupList
        this.groupList = allGroupList
      }
      // this.getSiteList() // 获取站址列表
    },
    setTableData() { // 设置显示的表格数据
      return this.getEquipmentmitrcd()
    },
    getMsgList(e) {
      // 页码改变时触发
      this.currentPage = e.page || 1
      this.pageSize = e.limit || this.pageSize
      this.setTableData() // 设置显示的表格数据
    },
    add() { // 点击添加按钮
      if (this.submitting) return
      // 新增和导入共用抽屉，进入新增时清理旧文件，防止 Excel 文件混入图片提交。
      this.form = { batteryCode: '', companyName: '', faultDescription: '', faultDetail: '', fileList: [], files: [] }
      this.dialogStatus = 'add'
      this.showDrawer = true // 表单可见
    },
    modify(row) { // 点击修改按钮
      this.dialogStatus = 'modify'
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
    async getSiteList() { // 获取站址列表
      const siteList = []
      const res = await findSiteListExactMatch()
      let list = []
      if (res.list) {
        list = res.list
      }
      list.forEach(item => {
        item.key = item.id
        item.name = item.siteName
        siteList.push(item)
      })
      this.siteList = siteList
      const searchSiteList = JSON.parse(JSON.stringify(siteList))
      searchSiteList.splice(0, 0, { key: '', name: '全部' })
      this.searchSiteList = searchSiteList
      return this.siteList.length !== 0
    },
    equipmentTypeChange() { // 库房类型改变
      this.equipTypeList.forEach(item => {
        if (this.form.equipmentTypeId === item.key) {
          this.form.equipmentTypeName = item.name
        }
      })
    },
    belongToSiteChange() { // 站址改变
      this.siteList.forEach(item => {
        if (this.form.belongToSiteId === item.key) {
          this.form.belongToSiteName = item.name
          this.apiKeys = item.apiKeys ? item.apiKeys : []
          this.form.apiKeys = this.apiKeys
          this.form.apiKey = this.apiKeys[0] ? this.apiKeys[0].key : ''
          this.form.equipmentDetails.apiKey = this.apiKeys[0] ? this.apiKeys[0].key : ''
        }
      })
    },
    apiKeyChange() { // apiKey改变时
      this.form.equipmentDetails.apiKey = this.form.apiKey
    },
    selectFile(data) {
      const files = Array.isArray(data?.fileList) ? data.fileList : []
      const validFiles = []
      let hasOversizedFile = false
      files.forEach(item => {
        if (item.size > 5 * 1024 * 1024) {
          hasOversizedFile = true
          return
        }
        const isImgMsg = fileIsImg(item)
        if (!isImgMsg?.sureIsImg) {
          Message({ message: `${item.name}不是图片`, type: 'warning', duration: 3000 })
          return
        }
        validFiles.push(item)
      })
      if (hasOversizedFile) {
        Message({ message: '上传文件不能大于5M', type: 'warning', duration: 3000 })
      }
      this.form.fileList = validFiles
    },
    upLoadDataFile() {
      if (this.submitting) return
      this.dialogStatus = 'upload'
      this.form = {
        fileList: []
      }
      this.showDrawer = true // 表单可见
    },
    selectDataFile() {

    },
    async getTriggerList(row) { // 获取触发器列表
      const alarmRes = await findAlarmUsers({
        equipmentId: row.id, // 所属库房Id(String)
        equipmentName: row.equipmentName // 所属库房名称(String)
      })
      const alarmList = alarmRes.list
      if (alarmList) {
        alarmList.forEach(item => {
          item.apiKey = row.apiKey
          this.sureRemove(item)
        })
      }
    },
    sureRemove(item) { // 删除触发器
      deletetrigger({
        Coding: item.coding,
        Apikey: item.apiKey
      })
      deleteAlarmUsers({
        id: item.id // 告警代码
      })
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

      this.getEquipmentmitrcd()
    },
    goDetail(row) { // 去往库房详情界面
      this.dialogDetails = true
      this.detail = row
    },
    selectMsgFile(data) { // 选择文件
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
      try {
        const filters = readMaintenanceFilters(this.bannerList)
        const res = await equipmentmitrcdExport({
          equipmentCode: filters.equipmentCode,
          equipmentTypeName: filters.equipmentTypeName,
          minimumCount: filters.minimumCount,
          startTime: filters.startTime,
          endTime: filters.endTime
        })
        // 直接下载本次响应，不再拼接旧服务器的 HTTPS /temp 地址。
        const contentType = res.headers?.['content-type'] || res.data?.type || ''
        if (!contentType.includes('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')) {
          let result = {}
          try { result = JSON.parse(await res.data.text()) } catch { /* 下面统一给出中文错误。 */ }
          throw new Error(result.msg || '运维信息导出失败，请稍后重试')
        }
        exportBlobFile(res)
      } catch (error) {
        Message({ message: maintenanceErrorMessage(error, '运维信息导出失败，请检查网络或稍后重试'), type: 'error' })
      }
    },
    async saveFile() { // 保存文件
      if (this.submitting) return
      if (!this.form.fileList?.length) {
        Message({ message: '请选择要导入的 Excel 文件', type: 'warning' })
        return
      }
      const fileList = this.form.fileList.slice()
      this.submitting = true
      try {
        const result = await equipmentmitrcdImport({ fileList })
        const summary = result?.return || result?.data || {}
        // 兼容后端历史拼写 dumplicateCount，并始终把三类数量展示给用户。
        const readCount = (...values) => {
          const value = values.find(item => item !== undefined && item !== null && item !== '')
          const number = Number(value)
          return Number.isFinite(number) ? number : 0
        }
        const successCount = readCount(summary.successCount, summary.sucessCount)
        const duplicateCount = readCount(summary.duplicateCount, summary.dumplicateCount,
          summary.duplcateCount)
        const failCount = readCount(summary.failCount, summary.failureCount)
        this.$notify({
          title: '上传请求发送成功',
          message: `导入完成：成功 ${successCount} 条，重复 ${duplicateCount} 条，失败 ${failCount} 条`,
          type: 'success',
          duration: 3 * 1000
        })
        await this.search()
        this.showDrawer = false
      } catch (error) {
        Message({ message: maintenanceErrorMessage(error, '运维信息导入失败，请检查网络或稍后重试'), type: 'error' })
      } finally {
        this.submitting = false
      }
    }
  }
}
</script>

<style lang="stylus" scoped>
.dataContentCss {
  float right
  width 100%
  height 100%
  .scrollbarCss {
    height calc(100% - 50px)
    padding 10px 20px 0 10px
  }
  .allImgContentCss {
    margin-top 10px
    .imgContentCss {
      float left
      margin 5px
      width 50px
      height 50px
      position relative
      .delectCss {
        position absolute
        top 10px
        left 20px
        z-index 999
      }
      .imgCss {
        width 100%
        height 100%
      }
    }
  }
}
</style>
