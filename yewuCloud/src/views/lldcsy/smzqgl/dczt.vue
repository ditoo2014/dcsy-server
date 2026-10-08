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
          height="615"
          :data="tableData"
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
          <el-table-column label="状态" align="center">
            <template #default="scope">
              <span>{{ scope.row.zt }}</span>
            </template>
          </el-table-column>
          <el-table-column label="位置" align="center" width="300">
            <template #default="scope">
              <span>{{ scope.row.zdwz }}</span>
            </template>
          </el-table-column>
          <!-- <el-table-column label="位置" align="center">
            <template #default="scope">
              <span>{{ scope.row.wz }}</span>
            </template>
          </el-table-column> -->
          <el-table-column label="责任人" align="center">
            <template #default="scope">
              <span>{{ scope.row.zzr }}</span>
            </template>
          </el-table-column>
          <el-table-column label="电话" align="center">
            <template #default="scope">
              <span>{{ scope.row.dh }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作时间" align="center">
            <template #default="scope">
              <span>{{ scope.row.czsj }}</span>
            </template>
          </el-table-column>
          <el-table-column
            label="流转记录"
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
          v-model:page="page"
          v-model:limit="limit"
          :total="total"
          @pagination="getMsgList"
        />
      </el-scrollbar>

    </div>

    <!-- 中间弹出 -->
    <el-dialog v-model="dialogLzjl" title="" width="40%">
      <Lzjl
        v-if="dialogLzjl"
        :form-data="form"
      />
    </el-dialog>

  </div>
</template>

<script>
import Pagination from '@/components/Pagination' // 分页组件
import Banner from '@/components/ccpcComp/Banner.vue'
import MyElTree from '@/components/a_myComp/elTree/MyElTree.vue' // 左侧导航
import Lzjl from './comp/Lzjl.vue'
import { corpList } from '@/api/user.js'
import { formatDateString, objectSort } from '@/utils/utils.js'
import { findEquipmentListExactMatch } from '@/api/equipMana.js'
import { findEquipmentcirlog } from '@/api/basicData.js'
import { parseUserInfo } from '@/utils/safe-json.js'
import { findSiteListExactMatch } from '@/api/siteMana.js'
import totalImage from '@/assets/imgs/总量.png'
import startImage from './imgs/起点.png'
import middleImage from './imgs/中间.png'
import endImage from './imgs/终点.png'

export default {
  components: {
    Pagination,
    Banner,
    MyElTree,
    Lzjl
  },
  data() {
    return {
      showLoading: true,
      userInfo: parseUserInfo({}),

      bannerList: [
        { style: 'infoNum', icon: totalImage, title: '电池总数量', num: '', unit: '组' },
        { style: 'input', placeholder: '请输入电池ID', input: '' },
        { style: 'input', placeholder: '请输入电池类型', input: '' },
        { style: 'select', placeholder: '请选择状态', value: '', filterable: false, list: [
          { key: '', name: '全部状态' },
          { key: '1', name: '入库' },
          { key: '2', name: '出库' },
          { key: '3', name: '上站入网' },
          { key: '4', name: '退网下站' },
          { key: '6', name: '检测淘汰(结束)' },
          { key: '5', name: '检测待用' }]
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

      siteList: [], // 站点列表

      dialogLzjl: false
    }
  },
  mounted() {
    this.init()
  },
  methods: {
    async init() {
      await this.getTreeList()
      await this.getAllSiteList()
      this.getDataList()
    },
    async getTreeList() {
      this.tableData = []
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
      if (cdMsg && cdMsg.id) {
        this.treeList.splice(3, 0, cdMsg)
      }
      const firstTree = this.treeList[0]
      if (firstTree) {
        firstTree.selected = true
        this.targetGroupNowForm = {
          level: firstTree.level,
          parentId: firstTree.parentId,
          id: firstTree.id,
          name: firstTree.name
        }
      }
    },
    async getAllSiteList() { // 获取全部站点列表
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
      this.siteList = allTableData
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
      this.showLoading = true
      let equipmentCode = ''
      let equipmentTypeName = ''
      let selectZt = '' // 选择的状态
      this.bannerList.forEach(item => {
        switch (item.placeholder) {
          case '请输入电池ID': equipmentCode = item.input; break
          case '请输入电池类型': equipmentTypeName = item.input; break
          case '请选择状态': selectZt = item.value; break
        }
      })
      const tableData = []
      const query = {
        equipmentCode,
        equipmentTypeName,
        pageNum: this.page,
        pageSize: this.limit,
        includeLatestCirculation: true
      }
      if (this.targetGroupNowForm.id !== undefined && this.targetGroupNowForm.id !== null && this.targetGroupNowForm.id !== '') {
        query.corpId = this.targetGroupNowForm.id
      }
      if (selectZt !== '' && selectZt !== null && selectZt !== undefined) {
        query.positionStatus = Number(selectZt)
      }
      const res = await findEquipmentListExactMatch(query)
      if (res.return && res.return.data) {
        const list = res.return.data
        this.total = res.return.total
        list.forEach((item, index) => {
          this.siteList.forEach(siteItem => {
            if (item.belongToSiteId === siteItem.id) {
              item.zdwz = siteItem.siteDetails.addr
            }
          })

          item.xh = (this.page - 1) * this.limit + index + 1
          const latestLog = item.latestCirculationLog || {}
          const operateDetails = latestLog.operateDetails || {}
          item.zzr = operateDetails.commander || '-'
          item.dh = operateDetails.phone || '-'
          item.czsj = this.formatCirculationTime(operateDetails.createTime || latestLog.createTime)
          let zt = ''
          switch (item.positionStatus) {
            case 0:
              zt = '未知状态'
              break
            case 1:
              zt = '入库'
              break
            case 2:
              zt = '出库'
              break
            case 3:
              zt = '上站入网'
              break
            case 4:
              zt = '退网下站'
              break
            case 6:
              zt = '检测淘汰(结束)'
              break
            case 5:
              zt = '检测待用'
              break
          }
          item.zt = zt
          tableData.push(item)
        })
      } else {
        this.total = 0
      }
      this.bannerList.forEach(item => {
        if (item.style === 'infoNum') {
          item.num = this.total.toString()
        }
      })
      this.tableData = tableData
      this.showLoading = false
    },
    getMsgList(e) {
      // 页码改变时触发
      this.page = e.page
      this.limit = e.limit
      this.getDataList()
    },
    clickBannerBtn(item) { // 点击banner的按钮
      if (item.buttonName === '添加库房') {
        this.add()
      } else if (item.buttonName === '查询') {
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
      this.getDataList()
    },
    formatCirculationTime(value) {
      if (value === undefined || value === null || value === '') {
        return '-'
      }
      const valueText = String(value).trim()
      if (valueText === '' || valueText === '0') {
        return '-'
      }
      return /^(\d{10}|\d{13})$/.test(valueText)
        ? formatDateString(Number(valueText), 'timeStmapReset')
        : valueText
    },
    chooseTargetgroup(e) { // 点击左侧导航
      const selected = e.titleColor !== ''
      this.targetGroupNowForm = {
        level: e.level,
        parentId: e.parentId,
        id: selected ? e.id : undefined,
        name: selected ? e.title : ''
      }
      this.search()
    },
    async goDetail(row) { // 去往库房详情界面
      // let zt = ''
      // switch (row.positionStatus) {
      //   case 0:
      //     zt = '未知状态'
      //     break
      //   case 1:
      //     zt = '入库'
      //     break
      //   case 2:
      //     zt = '出库'
      //     break
      //   case 3:
      //     zt = '上站入网'
      //     break
      //   case 4:
      //     zt = '退网下站'
      //     break
      //   case 6:
      //     zt = '检测淘汰(结束)'
      //     break
      //   case 5:
      //     zt = '检测待用'
      //     break
      // }
      const logres = await findEquipmentcirlog({
        equipmentId: row.id
      })
      const operateDetailList = []
      if (logres.return && logres.return.data.length !== 0) {
        const dataList = logres.return.data
        dataList.forEach((item, index) => {
          const operateDetails = item.operateDetails
          if (dataList.length === 2) {
            switch (index) {
              case 0: operateDetails.icon = startImage; break
              case 1: operateDetails.icon = endImage; break
            }
          } else if (dataList.length >= 3) {
            switch (index) {
              case 0: operateDetails.icon = startImage; break
              case 1: operateDetails.icon = middleImage; break
              case 2: operateDetails.icon = endImage; break
            }
          }
          operateDetailList.push(operateDetails)
        })
      }
      if (operateDetailList.length === 0) {
        this.$alert('在站存量电池，直接入网，无流转记录！', '无法查看详情', {
          showConfirmButton: false
        }).catch(() => {})
        return
      }
      this.form = row
      this.dialogLzjl = true
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
