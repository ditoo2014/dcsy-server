<template>
  <div v-loading="showLoading" class="contentCss">
    <el-scrollbar class="treeContentCss allheight">
      <MyElTree :navi-name-list="treeList" @click-tree="chooseTargetgroup" />
    </el-scrollbar>
    <div class="dataContentCss">
      <Banner
        :list="bannerList"
        @click-button="clickBannerBtn"
        @input-keyup-enter="inputBannerKeyupEnter"
        @select-change="selectBannerChange"
      />
      <el-scrollbar class="scrollbarCss">
        <el-table
          :key="tableKey"
          height="575"
          :data="tableData"
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
              <span>{{ scope.row.batteryCode }}</span>
            </template>
          </el-table-column>
          <el-table-column label="电池类型" align="center">
            <template #default="scope">
              <span>{{ scope.row.equipmentTypeName || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" align="center">
            <template #default="scope">
              <span>{{ scope.row.statusName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="最后出现位置" align="center">
            <template #default="scope">
              <span>{{ scope.row.location }}</span>
            </template>
          </el-table-column>
          <el-table-column
            label="轨迹追踪"
            align="center"
            class-name="small-padding fixed-width"
            width="220"
          >
            <template #default="scope">
              <el-button type="primary" size="small" @click="goDetail(scope.row)">
                查看详情
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <Pagination
          v-show="total>0"
          :page="page"
          :limit="limit"
          :total="total"
          @pagination="getMsgList"
        />
      </el-scrollbar>
    </div>

    <!-- 选择地图弹出层 -->
    <el-dialog v-model="dialogMap" title="轨迹追踪" width="60%">
      <div style="height: 600px;">
        <MapDriving
          v-if="dialogMap"
          :map-msg="mapMsg"
        />
      </div>
    </el-dialog>

  </div>
</template>

<script>
import Pagination from '@/components/Pagination' // 分页组件
import refreshPage from './refresh-page.js'
import { Message } from '@/utils/element-shim'
import Banner from '@/components/ccpcComp/Banner.vue'
import MapDriving from '@/components/a_myComp/mapDriving/MapDriving'
import MyElTree from '@/components/a_myComp/elTree/MyElTree.vue' // 左侧导航
import { corpList } from '@/api/user.js'
import { objectSort } from '@/utils/utils.js'
import { findEquipmentListExactMatch } from '@/api/equipMana.js'
import { findEquipmentcirlog } from '@/api/basicData.js'
import { parseUserInfo } from '@/utils/safe-json.js'

export default {
  mixins: [refreshPage],
  // 保留改名前的组件名称，使既有路由缓存继续生效。
  name: 'gjzz',
  components: {
    Pagination,
    Banner,
    MapDriving,
    MyElTree
  },
  data() {
    return {
      showLoading: false,
      userInfo: parseUserInfo({}),

      bannerList: [
        { style: 'input', placeholder: '请输入电池ID', input: '' },
        { style: 'input', placeholder: '请输入电池类型', input: '' },
        { style: 'select', placeholder: '请选择状态', value: '', list: [
          { key: '', name: '全部状态' },
          { key: 0, name: '未知状态' },
          { key: 1, name: '入库' },
          { key: 2, name: '出库' },
          { key: 3, name: '上站入网' },
          { key: 4, name: '退网下站' },
          { key: 6, name: '检测淘汰(结束)' },
          { key: 5, name: '检测待用' }]
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

      tableKey: 1, // 参数值变化，表格对应改变
      total: 0, // 数据总条数
      limit: 10, // 每次加载数据条数
      page: 1, // 页码

      tableData: [],

      mapMsg: {}, // 地图数据
      dialogMap: false
    }
  },
  methods: {
    async init() {
      await this.getTreeList()
    },
    async getTreeList() {
      const treeList = []
      const res = await corpList() // 列表（用户部门树）
      const list = res.return.data
      let chengduCompany = {} // 成都市分公司信息
      if (list) {
        list.forEach(item => {
          item.name = item.corpName
          if (item.name !== '中国铁塔股份有限公司' && item.name !== '四川省分公司') {
            item.expand = false // 不展开
          }
          if (item.name === '成都市分公司') {
            chengduCompany = item
          } else {
            treeList.push(item)
          }
        })
      }
      this.treeList = objectSort(treeList, 'level')
      this.treeList.splice(3, 0, chengduCompany)
      const fristTree = treeList[0]
      if (fristTree) {
        fristTree.selected = true
        this.targetGroupNowForm = fristTree
      }
    },
    resetData() { // 重置数据
      this.bannerList.forEach(item => {
        switch (item.placeholder) {
          case '请输入电池ID':
            item.input = ''
            break
          case '请输入电池类型':
            item.input = ''
            break
          case '请选择状态':
            item.value = ''
            break
        }
      })
      this.search()
    },
    async getDataList() {
      if (!this.pageActive) return
      const requestId = ++this.queryVersion
      const isCurrent = () => this.pageActive && requestId === this.queryVersion
      this.showLoading = true
      try {
        let equipmentCode = ''
        let equipmentTypeName = ''
        let selectedStatus = '' // 选择的状态
        this.bannerList.forEach(item => {
          switch (item.placeholder) {
            case '请输入电池ID': equipmentCode = item.input; break
            case '请输入电池类型': equipmentTypeName = item.input; break
            case '请选择状态': selectedStatus = item.value; break
          }
        })
        const tableData = []
        const res = await findEquipmentListExactMatch({
          corpId: this.targetGroupNowForm.id,
          equipmentCode,
          equipmentTypeName,
          positionStatus: selectedStatus,
          pageNum: this.page,
          pageSize: this.limit,
          // 列表只读取最后一条流转记录，完整轨迹留到点击详情时查询。
          includeLatestCirculation: true,
          latestCirculationOrder: 'time'
        })
        if (!isCurrent()) return
        let rowNumber = (this.page - 1) * this.limit
        if (res.return && res.return.data) {
          const list = res.return.data
          this.total = res.return.total
          for (const item of list) {
            rowNumber = rowNumber + 1
            let statusName = ''
            switch (item.positionStatus) {
              case 0:
                statusName = '未知状态'
                break
              case 1:
                statusName = '入库'
                break
              case 2:
                statusName = '出库'
                break
              case 3:
                statusName = '上站入网'
                break
              case 4:
                statusName = '退网下站'
                break
              case 6:
                statusName = '检测淘汰(结束)'
                break
              case 5:
                statusName = '检测待用'
                break
            }
            const latestLog = item.latestCirculationLog
            const data = {
              equipmentId: item.id,
              rowNumber: rowNumber,
              equipmentCode: item.equipmentCode,
              equipmentTypeName: item.equipmentTypeName || '',
              batteryCode: item.equipmentCode,
              statusName: statusName,
              // 无记录显示“无”；有记录但没有地址时保持空白，不回退到其他历史位置。
              location: latestLog ? latestLog.operateDetails?.locate : '无'
            }
            tableData.push(data)
          }
        } else {
          this.total = 0
        }
        this.bannerList.forEach(item => {
          if (item.style === 'infoNum') {
            item.num = this.total.toString()
          }
        })
        this.tableData = tableData
      } catch {
        if (!isCurrent()) return
        this.tableData = []
        this.total = 0
        this.bannerList[0].num = '0'
        Message({ message: '轨迹追踪加载失败，请稍后重试', type: 'error' })
      } finally {
        if (isCurrent()) this.showLoading = false
      }
    },
    getMsgList(e) {
      // 页码改变时触发
      this.page = this.limit === e.limit ? e.page : 1
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
    async selectBannerChange() { // banner选择状态
      this.search()
    },
    inputBannerKeyupEnter() { // banner输入框回车
      this.search()
    },
    search() {
      this.page = 1
      return this.getDataList()
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
    async goDetail(row) { // 去往库房详情界面
      let statusName = ''
      switch (row.positionStatus) {
        case 0:
          statusName = '未知状态'
          break
        case 1:
          statusName = '入库'
          break
        case 2:
          statusName = '出库'
          break
        case 3:
          statusName = '上站入网'
          break
        case 4:
          statusName = '退网下站'
          break
        case 6:
          statusName = '检测淘汰(结束)'
          break
        case 5:
          statusName = '检测待用'
          break
      }
      const logres = await findEquipmentcirlog({
        equipmentId: row.equipmentId
      })
      const dataList = logres.return && Array.isArray(logres.return.data)
        ? logres.return.data
        : []
      if (dataList.length === 0) {
        this.$alert('在站存量电池，直接入网，无流转记录！', '无法查看详情', {
          showConfirmButton: false
        }).catch(() => {})
        return
      }

      const operateTypeNameMap = {
        0: '未知状态',
        1: '入库',
        2: '出库',
        3: '上站入网',
        4: '退网下站',
        5: '检测待用',
        6: '检测淘汰(结束)'
      }
      // 按业务流转阶段排序，阶段相同时再按时间、日志 id 排序。
      // 这样 operateType 最大的记录始终是业务上的最后位置。
      const sortedLogs = dataList.slice().sort((left, right) => {
        const typeDiff = (Number(left.operateType) || 0) - (Number(right.operateType) || 0)
        if (typeDiff !== 0) return typeDiff
        const timeDiff = (Number(left.createTime) || 0) - (Number(right.createTime) || 0)
        if (timeDiff !== 0) return timeDiff
        return (Number(left.id) || 0) - (Number(right.id) || 0)
      })
      const pointList = []
      sortedLogs.forEach(log => {
        const details = log.operateDetails || {}
        const longitude = Number.parseFloat(details.longitude)
        const latitude = Number.parseFloat(details.latitude)
        const coordinateValid = Number.isFinite(longitude) && Number.isFinite(latitude) &&
          longitude >= -180 && longitude <= 180 && latitude >= -90 && latitude <= 90
        if (!coordinateValid) return
        pointList.push({
          id: log.id,
          longitude,
          latitude,
          operateType: log.operateType,
          operateTypeName: operateTypeNameMap[Number(log.operateType)] || '未知状态',
          locate: details.locate || '',
          storeroomAddress: details.storeroom_address || '',
          commander: details.commander || '',
          phone: details.phone || '',
          createTime: details.createTime || log.createTime || ''
        })
      })
      if (pointList.length === 0) {
        this.$alert('流转记录暂无有效位置信息！', '无法查看详情', {
          showConfirmButton: false
        }).catch(() => {})
        return
      }
      pointList.forEach((point, index) => {
        point.pointType = pointList.length === 1
          ? 'end'
          : index === 0
            ? 'start'
            : index === pointList.length - 1 ? 'end' : 'middle'
      })
      const lastPosition = pointList[pointList.length - 1]
      this.mapMsg = {
        equipmentId: row.equipmentId,
        equipmentCode: row.equipmentCode,
        batteryCode: row.equipmentCode,
        statusName,
        pointList,
        lastPosition,
        location: lastPosition.locate || lastPosition.storeroomAddress || '无'
      }
      this.dialogMap = true
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
