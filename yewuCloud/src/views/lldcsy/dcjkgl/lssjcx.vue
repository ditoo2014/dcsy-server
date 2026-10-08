<template>
  <div class="contentCss">
    <el-scrollbar class="treeContentCss">
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
                <span>{{ scope.row.xh }}</span>
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
            <el-table-column label="放电时长" align="center">
              <template #default="scope">
                <span>{{ scope.row.fdsc === '-' ? '未放电' : scope.row.fdsc }}</span>
              </template>
            </el-table-column>
            <el-table-column label="告警状态" align="center">
              <template #default="scope">
                <el-tag v-if="scope.row.collectStatus === 999" type="info">未采集</el-tag>
                <el-tag v-else-if="scope.row.isGj === true" type="danger">告警</el-tag>
                <el-tag v-else-if="scope.row.isGj === false" type="success">正常</el-tag>
                <el-tag v-else type="info">等待</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="预警状态" align="center">
              <template #default="scope">
                <el-tag v-if="scope.row.collectStatus === 999" type="info">未采集</el-tag>
                <el-tag v-else-if="scope.row.isYj === true" type="danger">预警</el-tag>
                <el-tag v-else-if="scope.row.isYj === false" type="success">正常</el-tag>
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
                >查看历史</el-button>
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
      >
        <dcDetail
          v-if="showDialog"
          :form-msg="equipData"
          page-name="历史数据"
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
import dcDetail from './dcDetail.vue'
import { corpList, findSelf } from '@/api/user.js'
import { processDrawList } from '@/utils/ccpcData.js'
import { findEquipmentListExactMatch, getEquipAllPoint } from '@/api/equipMana.js'
import { getapilistdata } from '@/api/moniList.js'
import { objectSort } from '@/utils/utils.js'
import { resolveBatteryPoint } from '@/utils/battery-points.js'
import {
  buildQueryableCorpKeys,
  findDefaultCorp,
  resolveCorpQueryId
} from '@/utils/corp-scope.js'
import { countbyequipment } from '@/api/trigger.js'
import { chinaRegions } from '@/components/a_myComp/mapComplex/index.js'
import lodash from 'lodash'

export default {
  components: {
    Pagination,
    Banner,
    MyElTree,
    dcDetail
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
      targetGroupForm: { // 预添加或修改导航树信号
        level: 0, // 目录级别
        parentId: 0, // 父id
        id: 1, // 自己的id
        name: '企业' // 名称
      },
      modifyTreeList: [], // 用户修改部门列表

      tableKey: 1, // 参数值变化，表格对应改变
      tableLoading: true, // 表格加载状态
      total: 0, // 数据总条数
      limit: 10, // 每次加载数据条数
      page: 1, // 页码

      searchSiteName: '', // 查询的站址
      searchEquipmentCode: '', // 查询设备
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

      tbjgTypeList: [ // 填报间隔类型列表
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
  methods: {
    async init() {
      this.tableLoading = true
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
      } finally {
        this.tableLoading = false
      }
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
    async getTreeList(allTableData, res) {
      const list = res.return.data
      const { treeList, cdMsg } = this.setTreeExpand({
        list,
        allTableData
      })

      this.treeList = objectSort(treeList, 'level')
      this.treeList.splice(3, 0, cdMsg)
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
        if (res.return && res.return.data) {
          const list = res.return.data
          this.total = res.return.total
          list.forEach((item, index) => {
            item.xh = (this.page - 1) * this.limit + index + 1
            item.collectStatus = ''
            item.isGj = null
            item.isYj = null
            const labelList = []
            item.dataDetails.forEach(dataItem => {
              labelList.push(dataItem.labelId)
            })
            item.labelList = labelList
            tableData.push(item)
          })
        } else {
          this.total = 0
        }
        this.tableData = tableData
        await this.setPointData() // 设置数据点数据
        await this.setGjYj() // 设置告警预警数据
      } catch (error) {
        console.error(error)
        this.total = 0
        this.tableData = []
      } finally {
        this.tableLoading = false
      }
    },
    async setPointData() { // 设置数据点数据
      const { equipList } = await getapilistdata({
        equipList: this.tableData
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
          dataIdSuffix: 'alg_fangdianshichang',
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
      this.tableData = safeList
    },
    async setGjYj() { // 设置告警预警数据
      const equipIds = []
      this.tableData.forEach(item => {
        if (item.collectStatus !== 999) {
          equipIds.push(item.id)
        }
      })
      if (equipIds.length === 0) {
        this.tableKey++
        return
      }
      const res = await countbyequipment({
        equipmentIds: equipIds,
        alarmType: '0', // 告警
        alarmStatus: 1 // 实时
      })
      const data = res.return?.data
      if (Array.isArray(data)) {
        this.tableData.forEach(item => {
          if (item.collectStatus === 999) {
            return
          }
          item.isGj = false
          data.forEach(gjItem => {
            if (item.id === gjItem.equipmentId && gjItem.alarmCount && gjItem.alarmCount > 0) {
              item.isGj = true
            }
          })
        })
      }
      const yjRes = await countbyequipment({
        equipmentIds: equipIds,
        alarmType: '1', // 预警
        alarmStatus: 1 // 实时
      })
      const yjData = yjRes.return?.data
      if (Array.isArray(yjData)) {
        this.tableData.forEach(item => {
          if (item.collectStatus === 999) {
            return
          }
          item.isYj = false
          yjData.forEach(gjItem => {
            if (item.id === gjItem.equipmentId && gjItem.alarmCount && gjItem.alarmCount > 0) {
              item.isYj = true
            }
          })
        })
      }
      this.tableKey++
    },
    getMsgList(e) {
      // 页码改变时触发
      this.page = e.page
      this.limit = e.limit
      this.getDataList()
    },
    clickBannerBtn(item) { // 点击banner的按钮
      if (item.buttonName === '查询') {
        this.getDataList()
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
    search() {
      this.page = 1
      this.getDataList()
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
