<template>
  <div class="contentCss">
    <el-scrollbar class="treeContentCss">
      <MyElTree
        :navi-name-list="treeList"
        :empty-text="treeEmptyText"
        @click-tree="chooseTargetgroup"
      />
    </el-scrollbar>
    <div class="dataContentCss">
      <Banner
        :list="bannerList"
        @click-button="clickBannerBtn"
        @input-change="inputBannerChange"
        @input-keyup-enter="inputBannerKeyupEnter"
      />
      <el-scrollbar class="scrollbarCss">
        <div
          v-loading="tableLoading"
          element-loading-text="正在加载数据"
          class="tableLoadingWrap"
        >
          <el-table
            :key="tableKey"
            height="630"
            :data="tableData"
            empty-text=""
            fit
            highlight-current-row
          >
            <el-table-column label="序号" align="center" width="50">
              <template #default="scope">
                <span>{{ scope.row.rowNumber }}</span>
              </template>
            </el-table-column>
            <el-table-column label="电池ID" align="center" width="120">
              <template #default="scope">
                <span>{{ scope.row.equipmentCode }}</span>
              </template>
            </el-table-column>
            <el-table-column label="电池类型" align="center">
              <template #default="scope">
                <span>{{ scope.row.equipmentTypeName || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="站址名称" align="center">
              <template #default="scope">
                <span>{{ scope.row.belongToSiteName }}</span>
              </template>
            </el-table-column>
            <el-table-column label="组电压" align="center">
              <template #default="scope">
                <span>{{ scope.row.zdy }}</span>
              </template>
            </el-table-column>
            <el-table-column label="组电流" align="center">
              <template #default="scope">
                <span>{{ scope.row.zdl }}</span>
              </template>
            </el-table-column>
            <el-table-column label="SOH" align="center">
              <template #default="scope">
                <span>{{ scope.row.soh }}</span>
              </template>
            </el-table-column>
            <el-table-column label="SOC" align="center">
              <template #default="scope">
                <span>{{ scope.row.soc }}</span>
              </template>
            </el-table-column>
            <el-table-column label="剩余放电时长" align="center">
              <template #default="scope">
                <span>{{ scope.row.fdsc === '-' ? '未放电' : scope.row.fdsc }}</span>
              </template>
            </el-table-column>
            <el-table-column label="告警状态" align="center">
              <template #default="scope">
                <el-tag v-if="scope.row.collectStatus === 999" type="info">未采集</el-tag>
                <el-tag v-else-if="scope.row.hasAlarm === true" type="danger">告警</el-tag>
                <el-tag v-else-if="scope.row.hasAlarm === false" type="success">正常</el-tag>
                <el-tag v-else type="info">等待</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="预警状态" align="center">
              <template #default="scope">
                <el-tag v-if="scope.row.collectStatus === 999" type="info">未采集</el-tag>
                <el-tag v-else-if="scope.row.hasWarning === true" type="danger">预警</el-tag>
                <el-tag v-else-if="scope.row.hasWarning === false" type="success">正常</el-tag>
                <el-tag v-else type="info">等待</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="采集状态" align="center">
              <template #default="scope">
                <el-tag v-if="scope.row.collectStatus === 1" type="success">正常</el-tag>
                <el-tag v-else-if="scope.row.collectStatus === 2" type="warning">离线</el-tag>
                <el-tag v-else-if="scope.row.collectStatus === ''" type="info">等待</el-tag>
                <el-tag v-else-if="scope.row.collectStatus === 999" type="info">未采集</el-tag>
                <el-tag v-else type="danger">异常</el-tag>
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
          <Pagination
            v-show="total>0"
            v-model:page="page"
            v-model:limit="limit"
            :total="total"
            @pagination="getMsgList"
          />
        </div>
      </el-scrollbar>
    </div>

    <div class="jkztjcCss">
      <!-- 中间弹出 -->
      <el-dialog
        v-model="showDialog"
        class="dcDetailDialog"
        title="电池详情"
        width="90%"
        destroy-on-close
      >
        <BatteryDetail
          v-if="showDialog"
          :form-msg="equipData"
          @click-close="showDialog = false"
        />
      </el-dialog>
    </div>

  </div>
</template>

<script>
import Pagination from '@/components/Pagination' // 分页组件
import Banner from '@/components/ccpcComp/Banner.vue'
import MyElTree from '@/components/a_myComp/elTree/MyElTree.vue' // 左侧导航
import BatteryDetail from './BatteryDetail.vue'
import { corpList, findSelf } from '@/api/user.js'
import { processDrawList } from '@/utils/ccpcData.js'
import { findEquipmentListExactMatch, getEquipAllPoint } from '@/api/equipMana.js'
import { getapilistdata } from '@/api/moniList.js'
import { objectSort } from '@/utils/utils.js'
import { resolveBatteryPoint, REMAINING_DISCHARGE_TIME_POINT_DATA_ID } from '@/utils/battery-points.js'
import {
  buildQueryableCorpKeys,
  findDefaultCorp,
  resolveCorpQueryId
} from '@/utils/corp-scope.js'
import { countbyequipment, getAlarmStatusByEquipment } from '@/api/trigger.js'
import { getCountbycorp } from '@/api/dashboard.js'
import { chinaRegions } from '@/components/a_myComp/mapComplex/index.js'
import lodash from 'lodash'

export default {
  // KeepAlive include uses the existing lower-case route name.
  // eslint-disable-next-line vue/component-definition-name-casing
  name: 'jkztjc',
  components: {
    Pagination,
    Banner,
    MyElTree,
    BatteryDetail
  },
  data() {
    return {
      userInfo: {},
      queryableCorpKeys: [],
      bannerList: [
        { style: 'input', placeholder: '请输入电池ID', input: '' },
        { style: 'input', placeholder: '请输入电池类型', input: '' },
        { style: 'input', placeholder: '请输入站址名称', input: '' },
        { style: 'button', buttonName: '查询', type: 'primary' },
        { style: 'button', buttonName: '重置', type: '' }
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
      treeEmptyText: '正在加载部门',
      targetGroupForm: { // 预添加或修改导航树信号
        level: 0, // 目录级别
        parentId: 0, // 父id
        id: 1, // 自己的id
        name: '企业' // 名称
      },
      modifyTreeList: [], // 用户修改部门列表

      tableKey: 1, // 参数值变化，表格对应改变
      tableLoading: true, // 表格加载状态
      tableRequestSeq: 0, // 仅允许最后一次查询更新表格
      searchTimer: null,
      total: 0, // 数据总条数
      limit: 10, // 每次加载数据条数
      page: 1, // 页码

      searchSiteName: '', // 查询的站址
      searchEquipmentCode: '', // 查询电池id
      searchEquipmentTypeName: '', // 查询电池类型

      showDrawer: false, // 抽屉是否显示

      allTableData: [], // 全部数据
      tableData: [],

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

      showDialog: false,

      equipData: {},

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
  beforeUnmount() {
    this.clearSearchTimer()
    this.tableRequestSeq++
  },
  methods: {
    async init() {
      this.tableLoading = true
      this.treeEmptyText = '正在加载部门'
      try {
        const [corpRes, userRes] = await Promise.all([
          corpList(), // 列表（用户部门树）
          findSelf()
        ])
        const currentUser = userRes && userRes.return && userRes.return.user
        if (!currentUser) throw new Error('invalid current user response')
        this.userInfo = currentUser

        const defaultCorp = this.getTreeList({}, corpRes)
        this.treeEmptyText = '暂无可用部门'
        if (!defaultCorp) {
          this.total = 0
          this.tableData = []
          return
        }

        // 设备数量只影响树节点展开状态，不应阻塞部门树和右侧列表。
        this.refreshCorpCounts(corpRes)
        await this.getDataList()
      } catch (error) {
        console.error(error)
        this.treeEmptyText = '部门加载失败'
        this.total = 0
        this.tableData = []
      } finally {
        this.tableLoading = false
      }
    },
    async getCorpCountMap() {
      try {
        const res = await getCountbycorp({
          filterFunc: 'BinC',
          positionStatus: null,
          equipmentName: '',
          corpList: []
        })
        const corpIds = res && res.return && res.return.corp
        const countList = res && res.return && res.return.count
        if (!Array.isArray(corpIds) || !Array.isArray(countList)) {
          throw new Error('invalid equipment count response')
        }
        const countMap = {}
        corpIds.forEach((corpId, index) => {
          countMap[corpId] = Number(countList[index]) || 0
        })
        return countMap
      } catch (error) {
        console.error(error)
        try {
          const allTableData = await this.getAllDataList()
          const countMap = {}
          allTableData.forEach(item => {
            countMap[item.corpId] = (countMap[item.corpId] || 0) + 1
          })
          return countMap
        } catch (fallbackError) {
          console.error(fallbackError)
          return {}
        }
      }
    },
    async refreshCorpCounts(corpRes) {
      try {
        const corpCountMap = await this.getCorpCountMap()
        this.getTreeList(corpCountMap, corpRes, true)
      } catch (error) {
        console.error(error)
      }
    },
    async getAllDataList() { // 获取全部数据列表
      const res = await findEquipmentListExactMatch({
        pageNum: 1,
        pageSize: 1000000
      })
      const list = res && res.return && res.return.data
      return Array.isArray(list) ? list : []
    },
    getTreeList(corpCountMap, res, preserveSelection = false) {
      const selectedCorpId = preserveSelection && this.targetGroupNowForm
        ? this.targetGroupNowForm.id
        : null
      const list = res && res.return && Array.isArray(res.return.data) ? res.return.data : []
      const { treeList, cdMsg } = this.setTreeExpand({
        list,
        corpCountMap
      })

      this.treeList = objectSort(treeList, 'level')
      if (cdMsg && cdMsg.id != null) {
        this.treeList.splice(3, 0, cdMsg)
      }
      this.treeList.forEach(item => { item.selected = false })
      this.queryableCorpKeys = buildQueryableCorpKeys(this.treeList, this.userInfo)
      const previousCorp = selectedCorpId == null
        ? null
        : this.treeList.find(item => String(item.id) === String(selectedCorpId))
      const defaultCorp = previousCorp || findDefaultCorp(
        this.treeList,
        this.userInfo,
        this.queryableCorpKeys
      )
      if (defaultCorp) {
        defaultCorp.selected = true
        this.targetGroupNowForm = defaultCorp
      } else {
        this.targetGroupNowForm = {}
      }
      this.modifyTreeList = lodash.cloneDeep(this.treeList)
      return defaultCorp
    },
    setTreeExpand(params) {
      const { list, corpCountMap } = params
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
            regionItem.dataNum += corpCountMap[corpId] || 0
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
    resetData() { // 重置数据
      this.searchEquipmentCode = ''
      this.searchEquipmentTypeName = ''
      this.searchSiteName = ''
      this.bannerList.forEach(item => {
        switch (item.placeholder) {
          case '请输入电池ID':
            item.input = this.searchEquipmentCode
            break
          case '请输入电池类型':
            item.input = this.searchEquipmentTypeName
            break
          case '请输入站址名称':
            item.input = this.searchSiteName
            break
        }
      })
      this.search()
    },
    async getDataList() {
      const requestSeq = ++this.tableRequestSeq
      this.tableLoading = true
      try {
        const tableData = []
        const res = await getEquipAllPoint({
          corpId: resolveCorpQueryId(
            this.targetGroupNowForm.id,
            this.queryableCorpKeys,
            this.userInfo
          ),
          pageNum: this.page,
          pageSize: this.limit,
          positionStatus: 3, // 当前电池状态
          equipmentCode: this.searchEquipmentCode,
          equipmentTypeName: this.searchEquipmentTypeName,
          belongToSiteName: this.searchSiteName
        })
        if (requestSeq !== this.tableRequestSeq) return

        let total = 0
        if (res.return && res.return.data) {
          const list = res.return.data
          total = res.return.total
          list.forEach((item, index) => {
            item.rowNumber = (this.page - 1) * this.limit + index + 1
            item.collectStatus = ''
            item.hasAlarm = null
            item.hasWarning = null
            const labelList = []
            item.dataDetails.forEach(dataItem => {
              labelList.push(dataItem.labelId)
            })
            item.labelList = labelList
            tableData.push(item)
          })
        }
        const pointData = await this.setPointData(tableData) // 设置数据点数据
        if (requestSeq !== this.tableRequestSeq) return

        await this.loadAlarmWarningStatus(pointData) // 设置告警预警数据
        if (requestSeq !== this.tableRequestSeq) return

        this.total = total
        this.tableData = pointData
        this.tableKey++
      } catch (error) {
        console.error(error)
      } finally {
        if (requestSeq === this.tableRequestSeq) {
          this.tableLoading = false
        }
      }
    },
    async setPointData(tableData) { // 设置数据点数据
      if (!tableData.length) return []

      const { equipList } = await getapilistdata({
        equipList: tableData
      })
      const safeList = Array.isArray(equipList) ? equipList : []
      safeList.forEach(item => {
        item.zdy = '-'
        item.zdl = '-'
        item.soh = '-'
        item.soc = '-'
        item.fdsc = '-'
        const dataDetails = item.dataDetails || []
        const dischargePoint = resolveBatteryPoint(dataDetails, {
          dataIdSuffix: REMAINING_DISCHARGE_TIME_POINT_DATA_ID,
          exactNames: ['放电时长'],
          gatewayId: item.equipmentCode
        })
        dataDetails.forEach(labelItem => {
          const hasValue = labelItem.realTimeVal !== null && labelItem.realTimeVal !== undefined && labelItem.realTimeVal !== '' && labelItem.realTimeVal !== '异常'
          if (labelItem.labelName.indexOf('组电压') !== -1) {
            const voltage = Number(labelItem.realTimeVal)
            item.zdy = hasValue && Number.isFinite(voltage) ? voltage.toFixed(2) : '-'
          } else if (labelItem.labelName.indexOf('组电流') !== -1) {
            item.zdl = hasValue ? labelItem.realTimeVal : '-'
          } else if (labelItem.labelName === 'SOH') {
            item.soh = hasValue ? labelItem.realTimeVal : '-'
          } else if (labelItem.labelName === 'SOC') {
            item.soc = hasValue ? labelItem.realTimeVal : '-'
          }
        })
        if (dischargePoint) {
          const hasValue = dischargePoint.realTimeVal !== null && dischargePoint.realTimeVal !== undefined && dischargePoint.realTimeVal !== '' && dischargePoint.realTimeVal !== '异常'
          item.fdsc = hasValue ? (dischargePoint.realTimeVal === '999' ? '未放电' : dischargePoint.realTimeVal) : '-'
        }
      })
      return safeList
    },
    async loadAlarmWarningStatus(tableData) { // 设置告警预警数据
      const equipIds = []
      tableData.forEach(item => {
        if (item.collectStatus !== 999) {
          equipIds.push(item.id)
        }
      })
      if (equipIds.length === 0) return

      try {
        const statusRes = await getAlarmStatusByEquipment({
          equipmentIds: equipIds,
          alarmStatus: 1
        })
        const statusData = statusRes.return && statusRes.return.data
        if (!Array.isArray(statusData)) {
          throw new Error('invalid alarm status response')
        }
        tableData.forEach(item => {
          if (item.collectStatus !== 999) {
            item.hasAlarm = false
            item.hasWarning = false
          }
        })
        statusData.forEach(statusItem => {
          const tableItem = tableData.find(item => item.id === statusItem.equipmentId)
          if (!tableItem || tableItem.collectStatus === 999 || !statusItem.alarmCount) return
          if (String(statusItem.alarmType).indexOf('0') !== -1) tableItem.hasAlarm = true
          if (String(statusItem.alarmType).indexOf('1') !== -1) tableItem.hasWarning = true
        })
        return
      } catch (error) {
        console.error(error)
      }

      const [res, warningResponse] = await Promise.all([
        countbyequipment({
          equipmentIds: equipIds,
          alarmType: '0', // 告警
          alarmStatus: 1 // 实时
        }),
        countbyequipment({
          equipmentIds: equipIds,
          alarmType: '1', // 预警
          alarmStatus: 1 // 实时
        })
      ])
      const data = res.return?.data
      if (Array.isArray(data)) {
        tableData.forEach(item => {
          if (item.collectStatus === 999) {
            return
          }
          item.hasAlarm = false
          data.forEach(gjItem => {
            if (item.id === gjItem.equipmentId && gjItem.alarmCount && gjItem.alarmCount > 0) {
              item.hasAlarm = true
            }
          })
        })
      }
      const warningData = warningResponse.return?.data
      if (Array.isArray(warningData)) {
        tableData.forEach(item => {
          if (item.collectStatus === 999) {
            return
          }
          item.hasWarning = false
          warningData.forEach(gjItem => {
            if (item.id === gjItem.equipmentId && gjItem.alarmCount && gjItem.alarmCount > 0) {
              item.hasWarning = true
            }
          })
        })
      }
    },
    getMsgList(e) {
      // 页码改变时触发
      this.page = e.page
      this.limit = e.limit
      this.getDataList()
    },
    clickBannerBtn(item) { // 点击banner的按钮
      if (item.buttonName === '查询') {
        this.search()
      } else if (item.buttonName === '重置') {
        this.resetData()
      }
    },
    inputBannerChange(item) { // banner输入框内容改变
      if (item.placeholder === '请输入电池ID') {
        this.searchEquipmentCode = item.input
      } else if (item.placeholder === '请输入电池类型') {
        this.searchEquipmentTypeName = item.input
      } else if (item.placeholder === '请输入站址名称') {
        this.searchSiteName = item.input
      }
      this.scheduleSearch()
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
    search() {
      this.clearSearchTimer()
      this.page = 1
      this.getDataList()
    },
    scheduleSearch() {
      this.clearSearchTimer()
      this.searchTimer = setTimeout(() => {
        this.searchTimer = null
        this.search()
      }, 300)
    },
    clearSearchTimer() {
      if (this.searchTimer) {
        clearTimeout(this.searchTimer)
        this.searchTimer = null
      }
    },
    goDetail(row) { // 去往详情界面
      this.equipData = row
      this.showDialog = true
    }
  }
}
</script>

<style lang="stylus" scoped>
searchContentHeight = 50px
.contentCss {
  width 100%
  height 100%
  min-height 0
  display flex
  overflow hidden
}
.treeContentCss {
  flex 0 0 15%
  width auto
  height 100%
  min-height 0
  padding 5px
  border-right 1px solid #EAEDF7
  background-color white
  .treeSearchCss {
    height searchContentHeight
  }
}
.dataContentCss {
  flex 1
  min-width 0
  height 100%
  min-height 0
  display flex
  flex-direction column
  .scrollbarCss {
    flex 1
    min-height 0
    padding 10px 20px 0 10px
  }
  .tableLoadingWrap {
    display flex
    flex-direction column
    min-height 0
  }
}

</style>

<style lang="stylus">
.jkztjcCss {
  height: 100%;
}

/* The dialog is teleported to body, so these rules target its own class. */
.dcDetailDialog {
  display: flex;
  flex-direction: column;
  height: 80vh;
  max-height: calc(100vh - 32px);
  margin: 8vh auto 0 !important;
  box-sizing: border-box;

  .el-dialog__header {
    display: none;
  }

  .el-dialog__body {
    flex: 1;
    min-height: 0;
    height: auto !important;
    padding: 0 !important;
    overflow: hidden;
    box-sizing: border-box;
  }
}
</style>
