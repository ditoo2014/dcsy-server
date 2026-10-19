<template>
  <div class="contentCss">
    <div class="dataContentCss">
      <Banner
        :list="bannerList"
        @select-change="bannerSelectChange"
        @time-change="timeChange"
        @click-button="clickBannerBtn"
      />
      <el-scrollbar class="scrollbarCss">
        <el-table
          :key="tableKey"
          height="485"
          :data="tableData"
          fit
          highlight-current-row
        >
          <el-table-column label="序号" align="center" width="50">
            <template #default="scope">
              <span>{{ scope.row.rowNumber }}</span>
            </template>
          </el-table-column>
          <el-table-column label="告警名称" align="center">
            <template #default="scope">
              <span>{{ scope.row.alarmCode }}</span>
            </template>
          </el-table-column>
          <el-table-column label="电池ID" align="center" width="120">
            <template #default="scope">
              <span>{{ scope.row.equipmentCode }}</span>
            </template>
          </el-table-column>
          <el-table-column label="站址名称" align="center">
            <template #default="scope">
              <span>{{ scope.row.siteName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="预警等级" align="center">
            <template #default="scope">
              <span>{{ scope.row.alarmStage }}</span>
            </template>
          </el-table-column>
          <el-table-column label="数据值" align="center">
            <template #default="scope">
              <span>{{ scope.row.alarmValue }}</span>
            </template>
          </el-table-column>
          <el-table-column label="恢复值" align="center">
            <template #default="scope">
              <span>{{ scope.row.recoverValue }}</span>
            </template>
          </el-table-column>
          <el-table-column label="触发时间" align="center">
            <template #default="scope">
              <span>{{ scope.row.cfsj }}</span>
            </template>
          </el-table-column>
          <el-table-column label="恢复时间" align="center">
            <template #default="scope">
              <span>{{ scope.row.hfsj }}</span>
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

  </div>
</template>

<script>
import Pagination from '@/components/Pagination' // 分页组件
import Banner from '@/components/ccpcComp/Banner.vue'
import { dateConvStamp, formatDateString } from '@/utils/utils.js'
import { findSiteListExactMatch } from '@/api/sites.js'
import { findEquipmentListExactMatch } from '@/api/equipment.js'
import { findAlarmRecord } from '@/api/trigger.js'

export default {
  // 组件名称与路由名称一致，确保页面缓存正常匹配。
  name: 'WarningHistory',
  components: {
    Pagination,
    Banner
  },
  data() {
    return {
      bannerList: [
        { style: 'select', placeholder: '请输入站址名称', value: '', list: [] },
        { style: 'select', placeholder: '请输入电池ID', value: '', list: [] },
        { style: 'selectTimeSlot', name: '选择触发时间', timeDateArr: [],
          startPlaceholder: '开始时间', endPlaceholder: '结束时间', datePickerType: 'datetimerange',
          width: '22vw', size: 'default'
        },
        { style: 'button', buttonName: '重置', type: '' },
        { style: 'button', buttonName: '导出当页', type: 'primary', icon: 'el-icon-download' },
        { style: 'button', buttonName: '导出全部', type: 'primary', icon: 'el-icon-download' }
      ],

      tableKey: 1, // 参数值变化，表格对应改变
      total: 0, // 数据总条数
      limit: 10, // 每次加载数据条数
      page: 1, // 页码

      tableData: [],
      equipList: [],
      startTime: null,
      endTime: null
    }
  },
  mounted() {
    this.init()
  },
  methods: {
    async init() {
      await this.getSiteList()
      await this.getEquipList()
      this.getDataList()
    },
    resetData() { // 重置数据
      this.bannerList.forEach(item => {
        switch (item.placeholder) {
          case '请输入电池ID':
            item.value = ''
            break
          case '请输入站址名称':
            item.value = ''
            break
        }
        if (item.style === 'selectTimeSlot') item.timeDateArr = []
      })
      this.startTime = null
      this.endTime = null
      this.page = 1
      this.getDataList()
    },
    getSearchParams(pageNum = this.page, pageSize = this.limit) {
      let siteId = null
      let equipmentId = null
      this.bannerList.forEach(item => {
        if (item.placeholder === '请输入站址名称') siteId = item.value === '' ? null : item.value
        if (item.placeholder === '请输入电池ID') equipmentId = item.value === '' ? null : item.value
      })
      return {
        pageNum,
        pageSize,
        siteId,
        equipmentId,
        startTime: this.startTime,
        endTime: this.endTime,
        alarmType: '1',
        alarmStatus: 0
      }
    },
    async getDataList() {
      const tableData = []
      const res = await findAlarmRecord(this.getSearchParams())
      const data = res.return.data
      if (data) {
        data.forEach((item, index) => {
          item.rowNumber = (this.page - 1) * this.limit + index + 1
          item.cfsj = formatDateString(item.beginTime, 'timeStmapReset')
          if (item.endTime) {
            item.hfsj = formatDateString(item.endTime, 'timeStmapReset')
          } else {
            item.hfsj = '-'
          }
          if (item.hfsj === '-') {
            item.recoverValue = '-'
          }
          switch (item.alarmStage) {
            case 1:
              item.alarmStage = '一'
              break
            case 2:
              item.alarmStage = '二'
              break
            case 3:
              item.alarmStage = '三'
              break
            case 4:
              item.alarmStage = '四'
              break
          }
          this.equipList.forEach(equip => {
            if (item.equipmentId === equip.id) {
              item.equipmentCode = equip.equipmentCode
            }
          })
          tableData.push(item)
        })
        this.total = res.return.total
      } else {
        this.total = 0
      }
      this.tableData = tableData
    },
    getMsgList(e) {
      // 页码改变时触发
      this.page = e.page
      this.limit = e.limit
      this.getDataList()
    },
    bannerSelectChange() { // banner 下拉框内容改变
      this.page = 1
      this.getDataList()
    },
    timeChange(item) { // 按触发时间查询
      if (item.timeDateArr && item.timeDateArr.length === 2) {
        this.startTime = dateConvStamp(formatDateString(item.timeDateArr[0], 'formatDate')) / 1000
        this.endTime = dateConvStamp(formatDateString(item.timeDateArr[1], 'formatDate')) / 1000
      } else {
        this.startTime = null
        this.endTime = null
      }
      this.page = 1
      this.getDataList()
    },
    clickBannerBtn(item) { // 点击banner的按钮
      if (item.buttonName === '查询') {
        this.getDataList()
      } else if (item.buttonName === '重置') {
        this.resetData()
      } else if (item.buttonName === '导出当页') {
        this.exportExcel()
      } else if (item.buttonName === '导出全部') {
        this.exportAllExcel()
      }
    },
    async getSiteList() { // 获取站址列表
      const siteList = [
        { key: '', name: '全部站址预警' }
      ]
      const res = await findSiteListExactMatch({
        pageNum: 1,
        pageSize: 10000
      })
      if (res.return && res.return.data) {
        const list = res.return.data
        list.forEach(item => {
          item.key = item.id
          item.name = item.siteName
          siteList.push(item)
        })
      }
      this.bannerList.forEach(item => {
        if (item.placeholder === '请输入站址名称') {
          item.list = siteList
        }
      })
    },
    async getEquipList() {
      const tableData = [
        { key: '', name: '全部电池预警' }
      ]
      const res = await findEquipmentListExactMatch({
        pageNum: 1,
        pageSize: 10000
      })
      if (res.return && res.return.data) {
        const list = res.return.data
        list.forEach(item => {
          if (item.positionStatus === 3) {
            item.key = item.id
            item.name = item.equipmentName
            tableData.push(item)
          }
        })
      }
      this.bannerList.forEach(item => {
        if (item.placeholder === '请输入电池ID') {
          item.list = tableData
        }
      })
      this.equipList = tableData
    },
    async exportExcel() { // 导出excel表格
      // 本地导出 excel start
      import('@/vendor/exportToExcel').then(excel => {
        const tHeader = ['预警名称', '电池ID', '站址名称', '预警等级', '数据值', '恢复值', '触发时间', '恢复时间']
        const data = this.formatJson(this.tableData, tHeader.length)
        excel.export_json_to_excel({
          header: tHeader,
          data,
          filename: 'table-list'
        })
      })
      // 本地导出 excel end
    },
    formatJson(tableData, tHeaderLength) {
      const arr = [] // 声明一维数组
      for (let x = 0; x < tableData.length; x++) {
        arr[x] = [] // 声明二维
        for (let y = 0; y < tHeaderLength; y++) {
          arr[x][y] = ''
        }
      }
      tableData.forEach((item, index) => {
        arr[index][0] = item.alarmCode
        arr[index][1] = item.equipmentName
        arr[index][2] = item.siteName
        arr[index][3] = item.alarmStage
        arr[index][4] = item.alarmValue
        arr[index][5] = item.recoverValue
        arr[index][6] = item.cfsj
        arr[index][7] = item.hfsj
      })
      return arr
    },
    async exportAllExcel() { // 导出全部预警
      // 本地导出 excel start
      const tableData = []
      const res = await findAlarmRecord(this.getSearchParams(1, 100000))
      const data = res.return.data
      if (data) {
        data.forEach((item, index) => {
          item.rowNumber = index
          item.cfsj = formatDateString(item.beginTime, 'timeStmapReset')
          if (item.endTime) {
            item.hfsj = formatDateString(item.endTime, 'timeStmapReset')
          } else {
            item.hfsj = '-'
          }
          if (item.hfsj === '-') {
            item.recoverValue = '-'
          }
          switch (item.alarmStage) {
            case 1:
              item.alarmStage = '一'
              break
            case 2:
              item.alarmStage = '二'
              break
            case 3:
              item.alarmStage = '三'
              break
            case 4:
              item.alarmStage = '四'
              break
          }
          this.equipList.forEach(equip => {
            if (item.equipmentId === equip.id) {
              item.equipmentCode = equip.equipmentCode
            }
          })
          tableData.push(item)
        })
      }

      import('@/vendor/exportToExcel').then(excel => {
        const tHeader = ['预警名称', '电池ID', '站址名称', '预警等级', '数据值', '恢复值', '触发时间', '恢复时间']
        const data = this.formatJson(tableData, tHeader.length)
        excel.export_json_to_excel({
          header: tHeader,
          data,
          filename: 'table-list'
        })
      })
      // 本地导出 excel end
    }
  }
}
</script>

<style lang="stylus" scoped>
.dataContentCss {
  width 100%
  height 100%
  .scrollbarCss {
    height calc(100% - 50px)
    padding 10px 20px 0 10px
  }
}
</style>
