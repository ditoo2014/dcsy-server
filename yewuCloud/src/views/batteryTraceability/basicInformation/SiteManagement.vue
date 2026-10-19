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
          height="625"
          :data="tableData"
          fit
          highlight-current-row
          @selection-change="handleSelectionChange"
        >
          <el-table-column label="序号" width="50" align="center">
            <template #default="scope">
              <span>{{ scope.row.rowNumber }}</span>
            </template>
          </el-table-column>
          <el-table-column label="站址编码" width="220" align="center">
            <template #default="scope">
              <span>{{ scope.row.siteSerialNumber }}</span>
            </template>
          </el-table-column>
          <el-table-column label="站址名称" width="250" align="center">
            <template #default="scope">
              <span>{{ scope.row.siteName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="基站地址" width="320" align="center">
            <template #default="scope">
              <span>{{ scope.row.addr }}</span>
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
    <elDrawer v-model="showDrawer" :title="textMap[dialogStatus]" :mask-closable="false" width="500">
      <el-scrollbar style="height:80vh">
        <el-form
          v-if="dialogStatus === 'add' || dialogStatus === 'modify'"
          ref="dataForm"
          :rules="rules"
          :model="form"
          size="small"
          label-width="140px"
        >
          <el-row>
            <el-col v-if="dialogStatus === 'add'" :span="20">
              <el-form-item label="所属公司">
                <span>{{ form.corpName }}</span>
              </el-form-item>
            </el-col>
            <el-col v-if="dialogStatus === 'modify'" :span="20">
              <el-form-item label="所属公司">
                <MyElTree :navi-name-list="modifyTreeList" @click-tree="changeUserGroup" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="站址编码" prop="siteSerialNumber">
                <el-input v-model="form.siteSerialNumber" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="站址名称" prop="siteName">
                <el-input v-model="form.siteName" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="基站地址" prop="addr">
                <el-input v-model="form.addr" disabled>
                  <template #append>
                    <el-button :icon="$legacyIcon('el-icon-search')" @click="chooseMap" />
                  </template>
                </el-input>
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
      </el-scrollbar>
      <div class="fixedBottomCss">
        <el-button @click="showDrawer = false">取消</el-button>
        <el-button type="primary" @click="setData()">确定</el-button>
      </div>
    </elDrawer>

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
import MyElTree from '@/components/sharedComponents/elTree/MyElTree.vue' // 左侧导航
import Pagination from '@/components/Pagination' // 分页组件
import Banner from '@/components/ccpcComp/Banner.vue'
import SelectMapArea from '@/components/sharedComponents/selectMap/SelectMapArea.vue'
import ExportExcel from '@/components/sharedComponents/exportExcel/ExportExcel.vue'
import UploadFile from '@/components/sharedComponents/uploadFile/UploadFile.vue'
import { corpList } from '@/api/user.js'
import { findSiteListExactMatch, saveSite, deleteSite, modifySite } from '@/api/sites.js'
import { meAlert, objectSort, exportBlobFile } from '@/utils/utils.js'
import { parseUserInfo } from '@/utils/safeJson.js'
import { siteExport, siteImport, siteImportTemplate } from '@/api/batteryTraceability.js'
import { chinaRegions } from '@/components/sharedComponents/mapComplex/index.js'
import lodash from 'lodash'
import totalImage from '@/assets/imgs/total.png'

export default {
  // 组件名称与路由名称一致，确保页面缓存正常匹配。
  name: 'SiteManagement',
  components: {
    MyElTree,
    Pagination,
    Banner,
    SelectMapArea,
    ExportExcel,
    UploadFile
  },
  data() {
    const validateApiKey = (rule, value, callback) => {
      if (value.length === 0) {
        callback(new Error('apiKey必填'))
      } else {
        let isCallBack = true // 是否通过校验
        value.forEach(item => {
          if (item.key === '') {
            isCallBack = false
            callback(new Error('apiKey不能为空'))
            return
          }
          if (item.key.indexOf(' ') !== -1) {
            isCallBack = false
            callback(new Error('apiKey不能包含空格'))
            return
          }
          if (item.name === '') {
            isCallBack = false
            callback(new Error('apiKey名称不能为空'))
            return
          }
        })
        if (isCallBack) {
          callback()
        }
      }
    }
    return {
      userInfo: parseUserInfo({}),

      bannerList: [
        { style: 'infoNum', icon: totalImage, title: '站址总量', num: '0', unit: '个' },
        { style: 'input', placeholder: '请输入站址编码', input: '' },
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

      searchSiteList: [], // 用于查询的站址列表
      searchSiteSerialNumber: '', // 查询的站址名称
      searchSiteId: '', // 查询的站址
      searchEquipName: '', // 查询设备

      showDrawer: false, // 抽屉是否显示
      textMap: { // 表单弹窗标题
        add: '添加站址',
        modify: '修改站址',
        upload: '导入excel'
      },
      dialogStatus: '', // 表单状态值为'add'或'modify' ，与 textMap 对应
      rules: {
        apiKeys: [{ required: true, trigger: 'blur', validator: validateApiKey }],
        siteSerialNumber: [{ required: true, trigger: 'blur', message: '站址编码不能为空' }],
        siteName: [{ required: true, trigger: 'blur', message: '站址名称不能为空' }],
        addr: [{ required: true, trigger: 'blur', message: '地理位置不能为空' }]
      },

      tableData: [],
      selectedTableData: [], // 选中数据
      form: {
        apiKey: '', // 对应采集云APIKkey*(String)
        apiKeys: [], // 多个apiKey
        siteName: '', // 站址名称*(String)
        siteSerialNumber: '', // 站址名称
        siteGroupId: 1, // 站址所属分组ObjectId*
        siteGroupName: '全部', // 站址所属分组名称*(String)
        siteGroupLevel: 0, // 站址分组级别
        corpId: '', // 所属部门
        addr: '', // 地理位置
        provinceName: '', // 所属省名称
        cityName: '', // 所属城市名称
        longitude: '', // 经度
        latitude: '' // 纬度
      },

      adminList: [], // 管理员列表

      equipTypeList: [], // 站址类型列表
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
      const allTableData = await this.getAllDataList() // 获取全部数据列表
      await this.getTreeList(allTableData)
      this.getDataList()
    },
    async getAllDataList() { // 获取全部数据列表
      const res = await findSiteListExactMatch({
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
      const { treeList, chengduCompany } = this.setTreeExpand({
        list,
        allTableData
      })

      this.treeList = objectSort(treeList, 'level')
      this.treeList.splice(3, 0, chengduCompany)
      const fristTree = treeList[0]
      if (fristTree) {
        fristTree.selected = true
        this.targetGroupNowForm = fristTree
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
      this.searchSiteSerialNumber = ''
      this.bannerList.forEach(item => {
        if (item.placeholder === '请输入站址编码') {
          item.input = this.searchSiteSerialNumber
        }
      })
      this.search()
    },
    async getDataList() {
      const tableData = []
      const res = await findSiteListExactMatch({
        pageNum: this.page,
        pageSize: this.limit,
        filterFunc: 'BinC', // ConE:通过设备关注用户查询, BinC:通过站址绑定部门查询
        siteSerialNumber: this.searchSiteSerialNumber, // 站址编码
        siteName: this.searchEquipName, // 站址名称
        corpId: this.targetGroupNowForm.id
      })
      if (res.return) {
        this.total = res.return.total
        const list = res.return.data
        if (list) {
          list.forEach((item, index) => {
            item.rowNumber = (this.page - 1) * this.limit + index + 1
            this.treeList.forEach(treeItem => {
              if (item.corpId === treeItem.id) {
                item.corpName = treeItem.name
              }
            })
            const detailMsg = item.siteDetails
            item.addr = detailMsg.addr
            item.cityName = detailMsg.cityName
            item.latitude = detailMsg.latitude
            item.longitude = detailMsg.longitude
            item.provinceName = detailMsg.provinceName
            tableData.push(item)
          })
        }
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
      if (item.placeholder === '请输入站址编码') {
        this.searchSiteSerialNumber = item.input
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
          apiKey: '', // 对应采集云APIKkey*(String)
          apiKeys: [], // 多个apiKey
          siteName: '', // 站址名称*(String)
          siteSerialNumber: '', // 站址名称
          siteGroupId: 1, // 站址所属分组ObjectId*
          siteGroupName: '全部', // 站址所属分组名称*(String)
          siteGroupLevel: 0, // 站址分组级别
          corpId: this.targetGroupNowForm.id, // 所属部门
          corpName: this.targetGroupNowForm.name, // 所属部门名称
          addr: '', // 地理位置
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
          apiKey: row.apiKey, // 对应采集云APIKkey*(String)
          apiKeys: row.apiKeys ? row.apiKeys : [], // 多个apiKey
          siteName: row.siteName, // 站址名称*(String)
          siteSerialNumber: row.siteSerialNumber, // 站址名称
          siteGroupId: row.siteGroupId, // 站址所属分组ObjectId*
          siteGroupName: row.siteGroupName, // 站址所属分组名称*(String)
          siteGroupLevel: row.siteGroupLevel, // 站址分组级别
          corpId: row.corpId, // 所属部门
          corpName: row.corpName, // 所属部门名称
          addr: row.addr, // 地理位置
          provinceName: row.provinceName, // 所属省名称
          cityName: row.cityName, // 所属城市名称
          longitude: row.longitude, // 经度
          latitude: row.latitude, // 纬度
          equipmentTypeId: row.equipmentTypeId, // 设施类型ObjectId
          equipmentTypeName: row.equipmentTypeName // 设施类型名称(String)
        }
      } else if (this.dialogStatus === 'upload') {
        this.form = {
          fileList: []
        }
      }
      this.showDrawer = true // 表单可见
    },
    changeUserGroup(e) { // 修改部门
      this.form.corpId = e.id
    },
    addApiKey() { // 添加apiKey标签
      this.form.apiKeys.push({
        key: '', name: ''
      })
    },
    removeApikey(index) { // 删除apiKey的标签
      const apiKeys = []
      this.form.apiKeys.forEach((item, i) => {
        if (index !== i) {
          apiKeys.push(item)
        }
      })
      this.form.apiKeys = apiKeys
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
            if (this.form.apiKeys.length === 0) {
              this.form.apiKeys = [{ key: 'defaultTest', name: 'defaultTest' }]
            }
            this.form.apiKey = this.form.apiKeys[0].key
            saveSite(this.form).then(() => {
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
            modifySite(this.form).then(() => {
              this.$notify({
                title: '修改请求发送成功',
                message: `成功修改`,
                type: 'success',
                duration: 3 * 1000
              })
              this.Ordervalue = -1
              this.getDataList()
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
            const importResponse = await siteImport({
              fileList: this.form.fileList
            })
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
      const removeName = row.siteName
      const sure = await meAlert(this, {
        title: '确认删除',
        content: `此操作将永久删除${removeName}，是否继续？`
      })
      if (sure) {
        await deleteSite({
          id: row.id // 站址ObjectId
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
    clickDragTree() {

    },
    handleSelectionChange(e) { // 选中tab复选框
      this.selectedTableData = e
    },
    upLoadDataFile() { // 上传数据文件
      this.dialogStatus = 'upload'
      this.resetForm()
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
      const res = await siteExport({
        pageNum: this.page,
        pageSize: this.limit,
        filterFunc: 'BinC', // ConE:通过设备关注用户查询, BinC:通过站址绑定部门查询
        siteSerialNumber: this.searchSiteSerialNumber, // 站址名称
        siteName: this.searchEquipName, // 站址名称
        corpId: this.targetGroupNowForm.id
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
      const res = await siteImportTemplate()
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
