<template>
  <div
    v-loading="detailLoading"
    element-loading-text="正在加载数据"
    element-loading-background="rgba(255, 255, 255, 0.85)"
    class="dcDetailCss"
  >
    <div class="dcDetailContentCss">
      <div class="closeButCss cursorPointer" @click="clickClose">
        <el-icon style="font-size: 20px"><component :is="$legacyIcon('el-icon-close')" /></el-icon>
      </div>
      <div class="topBannerCss">
        <div class="zdContentCss">
          <div class="zdDetailsCss">
            <el-tooltip class="item" effect="light" placement="top-end">
              <template #content>
                <span class="tooltipCss">{{ formData.belongToSiteName }}</span>
              </template>
              <div class="zdMsg1Css">站址名称：{{ formData.belongToSiteName }}</div>
            </el-tooltip>
            <div class="zdMsg2Css">站址编码：{{ formData.siteCode || '-' }}</div>
          </div>
          <div class="zdAddrCss" :title="formData.addr || '-'">地址：{{ formData.addr || '-' }}</div>
        </div>
        <div class="zdBasicCss">
          <div class="zdLogoCss">
            <img src="./imgs/dcsyLogo.png">
          </div>
          <div class="zdBCss">电池总负载：{{ siteTotalLoad || '-' }} A</div>
          <div class="zdBCss">电池组数：{{ batteryGroups.length }}组</div>
          <div class="dqdczCss">
            <div class="dczTitileCss">当前电池组</div>
            <div class="dqdczSelCss">
              <el-select v-model="selectedBatteryGroup" filterable @change="handleBatteryGroupChange">
                <el-option v-for="item in batteryGroups" :key="item.key" :label="item.name" :value="item.key" />
              </el-select>
            </div>
          </div>
        </div>
      </div>
      <div class="dcHeaderCss">
        <div class="tabContentCss">
          <div v-for="(item, index) in butList" :key="index" class="butContentCss cursorPointer" @click="selectBut(item)">
            <div class="butCss">
              <div v-if="item.imgUrl" class="butBord1Css">
                <div class="butImgCss">
                  <img :src="item.imgUrl">
                </div>
                <div class="butNameCss">{{ item.name }}</div>
              </div>
              <div v-else class="butBord2Css">
                <div class="butNameCss">{{ item.name }}</div>
              </div>
            </div>
            <div class="cursorPointer" :class="item.selectEd ? 'selectButCss' : 'notSelectButCss'" />
          </div>
        </div>
        <div class="dczxxContentCss">
          <!-- <div class="dczxxCss">
          <div class="dczxxBordCss">
            <div class="dczxxTitleCss">
              电池组名称：
            </div>
            <div class="dczxxMsgCss">{{ formData.equipmentName }}</div>
          </div>
        </div> -->
          <div class="dczxxCss">
            <div class="dczxxBordCss">
              <div class="dczxxTitleCss">
                电池组ID：
              </div>
              <div class="dczxxMsgCss">{{ formData.equipmentCode }}</div>
            </div>
          </div>
        </div>
        <div class="hqsjsjCss">获取数据时间：{{ dataDate || '-' }}</div>
      </div>
      <RealTimeData
        v-if="nowPageName === '实时数据'"
        :key="formData.id"
        :form-data="formData"
        :real-data-list="realDataList"
        @loading-change="handleChildLoadingChange"
      />
      <HistrData
        v-if="nowPageName === '历史数据'"
        :key="formData.id"
        :form-data="formData"
        :real-data-list="realDataList"
      />
      <RealtimeAlarmPanel
        v-if="nowPageName === '实时告警'"
        :key="formData.id"
        :form-data="formData"
        :real-data-list="realDataList"
        @alarm-count-change="handleAlarmCountChange"
      />
      <RealtimeWarningPanel
        v-if="nowPageName === '实时预警'"
        :key="formData.id"
        :form-data="formData"
        :real-data-list="realDataList"
        @warning-count-change="handleWarningCountChange"
      />
    </div>

  </div>
</template>

<script>
import lodash from 'lodash'
import RealTimeData from './comp/RealTimeData.vue'
import HistrData from './comp/HistrData.vue'
import RealtimeAlarmPanel from './comp/RealtimeAlarmPanel.vue'
import RealtimeWarningPanel from './comp/RealtimeWarningPanel.vue'
import { getapilistdata, getRealTimeData, getStreamLabelList } from '@/api/moniList.js'
import { findSiteListExactMatch } from '@/api/siteMana.js'
import { getEquipAllPoint } from '@/api/equipMana.js'
import { findAlarmRecord, getAlarmStatusByEquipment } from '@/api/trigger.js'
import { resolveBatteryPoint, REMAINING_DISCHARGE_TIME_POINT_DATA_ID } from '@/utils/battery-points.js'
import alarmImage from './imgs/告警3.png'
import warningImage from './imgs/预警3.png'

export default {
  // 保留改名前的组件名称，使既有路由缓存继续生效。
  name: 'dcDetail',
  components: {
    RealTimeData,
    HistrData,
    RealtimeAlarmPanel,
    RealtimeWarningPanel
  },
  props: {
    pageName: {
      type: String,
      default: '实时数据'
    },
    formMsg: {
      type: Object,
      default: () => {
        return {}
      }
    }
  },
  data() {
    return {
      dataDate: '', // 获取数据时间
      detailMainLoading: true,
      realTimeLoading: false,
      pageReady: false,
      detailRequestVersion: 0,

      selectedBatteryGroup: '', // 当前电池组
      batteryGroups: [],
      siteTotalLoad: '-', // 站址总负载

      butList: [
        { key: '实时数据', name: '实时数据', selectEd: false },
        { key: '历史数据', name: '历史数据', selectEd: false },
        { key: '实时告警', name: '实时告警', selectEd: false },
        { key: '实时预警', name: '实时预警', selectEd: false }
      ],

      nowPageName: '',

      formData: {},
      realDataList: []
    }
  },
  computed: {
    detailLoading() {
      return this.detailMainLoading || this.realTimeLoading
    }
  },
  mounted() {
    this.init()
  },
  beforeUnmount() {
    this.detailRequestVersion++
  },
  methods: {
    /**
     * 初始化页面数据：先渲染壳层，后台并行补数
     */
    init() {
      if (this.$route.query && this.$route.query.row) {
        this.formData = { ...JSON.parse(this.$route.query.row) }
      } else {
        this.formData = { ...this.formMsg }
      }
      const currentItem = {
        ...this.formData,
        key: this.formData.id,
        name: this.formData.equipmentName || this.formData.equipmentCode || String(this.formData.id)
      }
      this.batteryGroups = [currentItem]
      this.selectedBatteryGroup = this.formData.id
      this.siteTotalLoad = '-'

      const defaultTab = this.butList.find(item => item.key === this.pageName) || this.butList[0]
      this.selectBut(defaultTab)
      this.pageReady = true
      this.loadDetailData()
    },
    async loadDetailData() {
      const requestVersion = ++this.detailRequestVersion
      this.detailMainLoading = true
      try {
        const currentRealTimePromise = this.getRealTimeDataPromise(this.formData)
        const alarmStatusPromise = this.loadAlarmWarningStatus()
        await Promise.all([
          alarmStatusPromise,
          this.loadBatteryGroups(),
          this.loadDataMsg(currentRealTimePromise, requestVersion)
        ])
        const alarmStatus = await alarmStatusPromise
        if (requestVersion !== this.detailRequestVersion) return
        this.formData.hasAlarm = alarmStatus.hasAlarm
        this.formData.hasWarning = alarmStatus.hasWarning
        this.updateTabAlarmIcons()
      } catch (error) {
        if (requestVersion === this.detailRequestVersion) this.$message.error(error.message || '电池详情查询失败')
      } finally {
        if (requestVersion === this.detailRequestVersion) this.detailMainLoading = false
      }
    },
    updateTabAlarmIcons() {
      this.butList.forEach(item => {
        if (item.name === '实时告警') {
          item.imgUrl = this.formData.hasAlarm ? alarmImage : null
        } else if (item.name === '实时预警') {
          item.imgUrl = this.formData.hasWarning ? warningImage : null
        }
      })
    },
    async loadAlarmWarningStatus() { // 设置告警预警数据
      const item = this.formData
      if (item.collectStatus === 999) {
        return { hasAlarm: false, hasWarning: false }
      }
      try {
        const statusRes = await getAlarmStatusByEquipment({
          equipmentIds: [item.id],
          alarmStatus: 1
        })
        const statusData = statusRes.return && statusRes.return.data
        if (!Array.isArray(statusData)) {
          throw new Error('invalid alarm status response')
        }
        return {
          hasAlarm: statusData.some(statusItem => statusItem.alarmCount > 0 && String(statusItem.alarmType).indexOf('0') !== -1),
          hasWarning: statusData.some(statusItem => statusItem.alarmCount > 0 && String(statusItem.alarmType).indexOf('1') !== -1)
        }
      } catch (error) {
        console.error(error)
      }

      const [res, warningResponse] = await Promise.all([
        findAlarmRecord({
          pageNum: 1,
          pageSize: 1,
          equipmentId: item.id,
          alarmType: '0', // 告警
          alarmStatus: 1 // 实时
        }),
        findAlarmRecord({
          pageNum: 1,
          pageSize: 1,
          equipmentId: item.id,
          alarmType: '1', // 预警
          alarmStatus: 1 // 实时
        })
      ])
      const data = res.return.data
      const warningData = warningResponse.return.data
      return {
        hasAlarm: Array.isArray(data) && data.length > 0,
        hasWarning: Array.isArray(warningData) && warningData.length > 0
      }
    },
    async loadBatteryGroups() { // 获取电池组列表
      const tableData = []
      const res = await getEquipAllPoint({
        belongToSiteId: this.formData.belongToSiteId,
        pageNum: 1,
        pageSize: 1000
      })
      if (res.return) {
        const list = res.return.data
        list.forEach(item => {
          item.key = item.id
          item.name = item.equipmentName
          item.labelList = (item.dataDetails || []).map(dataItem => dataItem.labelId)
          tableData.push(item)
        })
      }
      if (tableData.length) {
        this.batteryGroups = tableData
        this.selectedBatteryGroup = this.formData.id
      }
      await this.loadPointData()
    },
    async loadPointData() {
      await this.setPointData()
    },
    async setPointData() { // 设置数据点数据
      let siteTotalLoad = 0 // 站址总负载
      let batteryGroupData
      try {
        const { equipList } = await getapilistdata({ equipList: this.batteryGroups })
        const safeList = Array.isArray(equipList) ? equipList : []
        this.batteryGroups = safeList
        batteryGroupData = safeList.map(item => ({ item, list: item.dataDetails || [] }))
      } catch (error) {
        console.error(error)
        batteryGroupData = await Promise.all(this.batteryGroups.map(async item => {
          try {
            return { item, list: await getRealTimeData(item) }
          } catch (requestError) {
            console.error(requestError)
            return { item, list: [] }
          }
        }))
      }
      batteryGroupData.forEach(({ item, list }) => {
        item.fdsc = '-'
        const dischargePoint = resolveBatteryPoint(list, {
          dataIdSuffix: REMAINING_DISCHARGE_TIME_POINT_DATA_ID,
          gatewayId: item.equipmentCode
        })
        list.forEach(labelItem => {
          const pointName = labelItem.labelName || labelItem.name || ''
          const lowerPointName = pointName.toLowerCase()
          if (pointName.indexOf('组电压') !== -1) {
            item.zdy = labelItem.realTimeVal
          } else if (pointName.indexOf('组电流') !== -1) {
            item.zdl = labelItem.realTimeVal
            const current = parseFloat(item.zdl)
            if (Number.isFinite(current)) {
              siteTotalLoad += current
            }
          } else if (lowerPointName.indexOf('soh') !== -1) {
            item.soh = labelItem.realTimeVal
          } else if (pointName.trim() === 'SOC') {
            item.soc = labelItem.realTimeVal
          }
        })
        if (dischargePoint) {
          const hasValue = dischargePoint.realTimeVal !== null && dischargePoint.realTimeVal !== undefined && dischargePoint.realTimeVal !== '' && dischargePoint.realTimeVal !== '异常'
          item.fdsc = hasValue ? (dischargePoint.realTimeVal === '999' ? '未放电' : dischargePoint.realTimeVal) : '-'
        }
      })
      this.siteTotalLoad = siteTotalLoad.toFixed(1)
    },
    getRealTimeDataPromise(formData) {
      const dataDetails = formData && formData.dataDetails
      const hasCachedData = Array.isArray(dataDetails) && (
        formData.collectStatus !== undefined ||
        dataDetails.some(item => Object.prototype.hasOwnProperty.call(item, 'realTimeVal'))
      )
      return hasCachedData ? Promise.resolve(dataDetails) : getRealTimeData(formData)
    },
    async getPointMetadataMap(formData) {
      const metadataMap = new Map()
      if (!formData || !formData.apiKey) return metadataMap
      try {
        const res = await getStreamLabelList(formData.apiKey)
        const list = res && res.return && res.return.labelList
        if (Array.isArray(list)) {
          list.forEach(item => {
            if (item && item.labelId != null) {
              metadataMap.set(String(item.labelId), item)
            }
          })
        }
      } catch (error) {
        console.error(error)
      }
      return metadataMap
    },
    attachPointMetadata(list, metadataMap) {
      const safeList = Array.isArray(list) ? list : []
      return safeList.map(item => {
        const metadata = metadataMap.get(String(item.labelId))
        if (!metadata) return item
        return {
          ...item,
          dataId: metadata.dataId || item.dataId || '',
          gatewayId: metadata.gatewayId || item.gatewayId || ''
        }
      })
    },
    async loadDataMsg(currentRealTimePromise, requestVersion = this.detailRequestVersion) {
      const requestedForm = { ...this.formData }
      const [realDataList, zdxxRes, pointMetadataMap] = await Promise.all([
        currentRealTimePromise || this.getRealTimeDataPromise(requestedForm),
        findSiteListExactMatch({
          id: requestedForm.belongToSiteId,
          pageNum: 1,
          pageSize: 1,
          filterFunc: ''
        }),
        this.getPointMetadataMap(requestedForm)
      ])
      // 快速切换电池时，仅允许当前电池的详情请求写回页面。
      if (requestVersion !== this.detailRequestVersion || String(requestedForm.id) !== String(this.formData.id)) return
      this.realDataList = this.attachPointMetadata(realDataList, pointMetadataMap) // 获取实时数据列表
      this.realDataList.forEach(item => {
        if (item.name.indexOf('组电压') !== -1) {
          this.dataDate = item.realTimeDate
        }
      })
      const formData = lodash.cloneDeep(this.formData)
      const list = zdxxRes.return.data
      const siteItem = Array.isArray(list) ? list[0] : null
      if (siteItem && formData.belongToSiteId === siteItem.id) {
        formData.siteCode = siteItem.siteSerialNumber
        formData.addr = siteItem.siteDetails ? siteItem.siteDetails.addr : '-'
      }
      this.formData = formData
    },
    async handleBatteryGroupChange() { // 电池组切换时
      const requestVersion = ++this.detailRequestVersion
      this.batteryGroups.forEach(item => {
        if (item.key === this.selectedBatteryGroup) {
          this.formData = item
        }
      })
      this.detailMainLoading = true
      this.realTimeLoading = false
      this.realDataList = []
      this.dataDate = ''
      try {
        const [, status] = await Promise.all([
          this.loadDataMsg(undefined, requestVersion),
          this.loadAlarmWarningStatus()
        ])
        if (requestVersion !== this.detailRequestVersion) return
        this.formData.hasAlarm = status.hasAlarm
        this.formData.hasWarning = status.hasWarning
        this.updateTabAlarmIcons()
      } catch (error) {
        if (requestVersion === this.detailRequestVersion) this.$message.error(error.message || '电池详情切换失败')
      } finally {
        if (requestVersion === this.detailRequestVersion) this.detailMainLoading = false
      }
    },
    handleChildLoadingChange(loading) {
      this.realTimeLoading = loading
    },
    handleAlarmCountChange(count) {
      this.formData.hasAlarm = Number(count) > 0
      this.updateTabAlarmIcons()
    },
    handleWarningCountChange(count) {
      this.formData.hasWarning = Number(count) > 0
      this.updateTabAlarmIcons()
    },
    selectBut(row) {
      this.butList.forEach(item => {
        item.selectEd = false
      })
      row.selectEd = true
      this.nowPageName = row.name
      if (row.name !== '实时数据') {
        this.realTimeLoading = false
      }
    },
    clickClose() {
      this.$emit('clickClose')
    }
  }
}
</script>

<style lang="stylus" scoped>
butStyle() {
  position relative
  padding-bottom 5px
  width 6vw
}

boxShadow = 0 0 3px 2px rgba(0, 0, 0, 0.2)

.dcDetailCss {
  position relative
  width 100%
  height 100%
  min-height 0
  display flex
  flex-direction column
  overflow hidden
  box-sizing border-box
  background-color #F7F7F7
  .dcDetailContentCss {
    width 100%
    height auto
    flex 1
    min-height 0
    overflow-y auto
    overflow-x hidden
    box-sizing border-box
  }
  .closeButCss {
    position absolute
    top 8px
    right 8px
    z-index 10
    color #fff
    display flex
    align-items center
    justify-content center
    width 28px
    height 28px
    border-radius 50%
    background-color rgba(0, 0, 0, 0.35)
    &:hover {
      background-color rgba(0, 0, 0, 0.5)
    }
  }
  .topBannerCss {
    position relative
    width 100%
    height 4vw
    background-color #1C4267
    border-radius 0.2vw
    .zdContentCss {
      float left
      margin-left 4.7%
      padding-right 3%
      width 44%
      color #fff
      text-align right
      .zdDetailsCss {
        // 隔离名称、编码的浮动，避免 vw 小数像素取整后挤压下一行地址。
        display flow-root
        width 100%
        min-height 2.2vw
        line-height 2.2vw
        font-size 1vw
        .zdMsg1Css {
          float right
          width 17vw
          white-space nowrap
          overflow hidden
          text-overflow ellipsis
        }
        .zdMsg2Css {
          float right
          margin-right 0.5vw
        }
      }
      .zdAddrCss {
        width 100%
        height 1.5vw
        line-height 1.5vw
        font-size 0.8vw
        // 长地址保持单行，鼠标悬停仍可查看完整地址。
        white-space nowrap
        overflow hidden
        text-overflow ellipsis
      }
    }
    .zdBasicCss {
      float left
      color #fff
      font-size 1vw
      .zdLogoCss {
        float left
        margin 0.5vw 2.8vw 0 0
        width 1.9vw
        height 3vw
      }
      .zdBCss {
        float left
        margin-right 1vw
        line-height 4vw
      }
      .dqdczCss {
        float left
        line-height 4vw
        .dczTitileCss {
          float left
          margin-right 0.5vw
        }
        .dqdczSelCss {
          float left
          margin-top -3px
          width 10vw
        }
      }
    }
  }
  .topBannerCss::after {
    content ''
    position absolute
    bottom 0
    left 46%
    width 8%
    height 0
    border-bottom 5px solid #7DFFFF
    border-left 5px solid transparent
    border-right 5px solid transparent
  }
  .dcHeaderCss {
    position relative
    margin 0.5vw 0
    width 100%
    height 4vw
    background url('./imgs/head_bg.png') no-repeat
    background-size 100% 100%
    .tabContentCss {
      position absolute
      top 1.9vw
      left 0.5vw
      .butContentCss {
        float left
        margin-right 0.2vw
        width 6vw
        height 2vw
        .butCss {
          position relative
          width 6vw
          height 2vw
          font-size 0.8vw
          font-weight bold
          background url('./imgs/btn.png') no-repeat
          background-size 100% 100%
          color #fff
          text-align center
          .butBord1Css {
            position absolute
            top 50%
            left 50%
            transform translate(-50%, -50%)
            width 5vw
            text-align center
            .butImgCss {
              float left
              width 1vw
              height 1vw
            }
            .butNameCss {
              float left
              margin-left 0.1vw
            }
          }
          .butBord2Css {
            position absolute
            top 50%
            left 50%
            transform translate(-50%, -50%)
            width 4vw
          }
        }
        .notSelectButCss {
          butStyle()
        }
        .selectButCss {
          butStyle()
        }
        .selectButCss::after {
          content ''
          position absolute
          bottom 0
          left 0
          width 100%
          height 0
          border-bottom 5px solid #7DFFFF
          border-left 5px solid transparent
          border-right 5px solid transparent
        }
      }
    }
    .dczxxContentCss {
      position absolute
      top 50%
      left 50%
      transform translate(-50%, -50%)
      color #1C4267
      width 25vw
      .dczxxCss {
        width 100%
        height 2vw
        font-size 1vw
        font-weight bolder
        text-align center
        .dczxxBordCss {
          display inline-block
          line-height normal
          padding 0 10px
          .dczxxTitleCss {
            float left
            line-height 1.5vw
            text-align left
          }
          .dczxxMsgCss {
            float left
            line-height 1.5vw
            text-align right
          }
        }
      }
    }
    .hqsjsjCss {
      position absolute
      top 2.8vw
      right 0.5vw
      font-size 0.8vw
    }
  }
}
img {
  width 100%
  height 100%
}
</style>

<style lang="stylus">
  .tooltipCss {
    font-size 1vw
  }
</style>
