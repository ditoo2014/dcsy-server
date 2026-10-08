<template>
  <div class="sssjCss">
    <div class="chartContentCss">
      <div class="chartCss">
        <div class="multipleYAxesCss">
          <MultipleYAxes
            id="realTimeDataEchartsBasicBar1"
            width="100%"
            height="100%"
            :data-change-key="basicBardataChangeKey"
            :chart-data="basicBarData1"
            link-group="realTimeDayCharts"
            :enable-data-zoom="true"
            :dynamic-symbol-on-zoom="true"
          />
        </div>
        <div class="multipleYAxesCss">
          <MultipleYAxes
            id="realTimeDataEchartsBasicBar2"
            width="100%"
            height="100%"
            :data-change-key="basicBardataChangeKey"
            :chart-data="basicBarData2"
            link-group="realTimeDayCharts"
            :enable-data-zoom="true"
            :show-data-zoom-slider="true"
            :dynamic-symbol-on-zoom="true"
          />
        </div>
        <div class="multipleYAxesCss">
          <MultipleYAxes
            id="realTimeDataEchartsBasicBar3"
            width="100%"
            height="100%"
            :data-change-key="basicBardataChangeKey"
            :chart-data="basicBarData3"
          />
        </div>
      </div>
    </div>
    <div class="dcDetailsMsgContentCss">
      <template v-if="showBasicInfo">
        <div class="dcDetailsMsgCss">
          <div class="dcTitleContentCss">
            <div class="dcTitleCss">【 电池信息 】</div>
          </div>
          <div class="dcztContentCss">
            <div class="dcztStatusContentCss">
              <div class="dcTZtTitleCss">电池状态：</div>
              <div class="dcXCss">
                <img v-if="dcztImg" :src="dcztImg">
              </div>
              <div class="dcTZtCss">{{ dzctMsg }}</div>
            </div>
            <div class="dcTypeContentCss">
              <div class="dcTypeTitleCss">电池类型：</div>
              <div class="dcTypeMsgCss" :title="formData.equipmentTypeName || '-'">
                {{ formData.equipmentTypeName || '-' }}
              </div>
            </div>
          </div>
          <div class="dcDetialsGzMsgCss">
            <div v-for="item in dcxx1List" :key="item.title" class="dcGzMsgContentCss">
              <div class="dcGzTitelCss">{{ item.title }}：</div>
              <div class="dcGzMsgCss">{{ item.msg }}</div>
            </div>
            <div v-for="item in dcxx2List" :key="item.title" class="dcGzMsgContentCss">
              <div class="dcGzTitelCss">{{ item.title }}：</div>
              <div class="dcGzMsgCss">{{ item.title === '剩余放电时长' && item.msg === '-' ? '未放电' : item.msg }}</div>
            </div>
          </div>
        </div>
        <div class="mbxxDetailsMsgCss">
          <div class="dczwTitleContentCss">
            <div class="dczwTitleCss">电池组温度：</div>
            <div class="dczwXCss">温差：{{ realTimeWcData }}</div>
          </div>
          <div class="mbxxContentCss">
            <div v-for="item in dczwdList" :key="item.title" class="mbxxCss">
              <div class="mbxxTitleCss">{{ item.title + '#' }}</div>
              <div class="mbxxValCss">{{ item.val }}</div>
            </div>
          </div>
        </div>
        <div class="qtwdDetailsMsgCss">
          <div class="qtwdTitleCss">其他温度：</div>
          <div class="qtwdMsgContentCss">
            <div class="qtwdMsgCss">
              <div class="qtwdMsgTitleCss">MOS温度：</div>
              <div class="qtwdMsgDataCss">{{ mosWd }}</div>
            </div>
            <div class="qtwdMsgCss">
              <div class="qtwdMsgTitleCss">环境温度：</div>
              <div class="qtwdMsgDataCss">{{ hjWd }}</div>
            </div>
          </div>
        </div>
        <div class="mbxxDetailsMsgCss">
          <div class="dczwTitleContentCss">
            <div class="dczwTitleCss">单体电压：</div>
            <div class="dczwXCss">压差：{{ realTimeYcData }}</div>
          </div>
          <div class="mbxxContentCss">
            <div v-for="item in dtdyList" :key="item.title" class="mbxxCss">
              <div class="mbxxTitleCss">{{ item.title + '#' }}</div>
              <div class="mbxxValCss">{{ item.val }}</div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script>
import MultipleYAxes from '@/components/a_myComp/echarts/MultipleYAxes.vue'
import { getdata } from '@/api/moniList.js'
import { dateConvStamp, format0800CSTDateString, dateAddOrSubtract, formatDateString } from '@/utils/utils.js'
import {
  PRESSURE_POINT_DATA_ID,
  TEMPERATURE_DIFFERENCE_POINT_DATA_ID,
  resolveBatteryPoint,
  resolveBatteryHealthStatus
} from '@/utils/battery-points.js'
import floatChargeImage from '../imgs/浮充.png'
import chargingImage from '../imgs/充电.png'
import dischargingImage from '../imgs/放电.png'

function formatAxisLabel(value, decimalPlaces, unit) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return ''
  }
  const roundedValue = Number(numericValue.toFixed(decimalPlaces))
  return `${roundedValue} ${unit}`
}

export default {
  components: {
    MultipleYAxes
  },
  props: {
    formData: {
      type: Object,
      default: () => {
        return {}
      }
    },
    realDataList: { // 实时数据列表
      type: Array,
      default: () => {
        return []
      }
    }
  },
  emits: ['loading-change'],
  data() {
    return {
      showBasicInfo: false,
      loadSeq: 0,

      labelList: [], // 数据点列表

      historyLabelList: [], // 历史数据点列表

      dzctMsg: '--',
      dcztImg: null,

      basicBardataChangeKey: 0,
      basicBarData1: {
        colors: ['#0ecc96', '#FF00F8', '#22c4ff'],
        legend: {
          show: true,
          color: '#000',
          fontSize: 14,
          fontWeight: 'bold',
          data: ['组电压', 'SOC', '组电流'] // 曲线标题
        },
        xAxis: { // 横轴：当天完整时间轴 00:00:00 - 23:59:59
          type: 'time',
          color: '#000',
          min: null,
          max: null,
          data: []
        },
        yAxis: [{ // 纵轴
          type: 'value',
          name: '组电压：V \n SOC：%',
          position: 'left',
          alignTicks: true,
          emptyRange: { min: 0, max: 100, interval: 20 },
          nameTextStyle: {
            fontSize: 14,
            fontWeight: 'bold'
          },
          axisLine: {
            show: true,
            lineStyle: {
              color: '#000'
            }
          },
          axisLabel: {
            formatter: '{value} '
          }
        }, {
          type: 'value',
          name: '组电流：A',
          position: 'right',
          alignTicks: true,
          emptyRange: { min: -100, max: 100, interval: 20 },
          nameTextStyle: {
            fontSize: 14,
            fontWeight: 'bold'
          },
          axisLine: {
            show: true,
            lineStyle: {
              color: '#000'
            }
          },
          axisLabel: {
            formatter: '{value} A'
          }
        }],
        showSeriesLable: 'true', // 是否显示曲线上的文字，为字符串'true'时显示
        grid: {
          bottom: 48
        },
        tooltip: {
          trigger: 'axis',
          triggerOn: 'mousemove',
          confine: true,
          axisPointer: {
            type: 'line',
            snap: true
          }
        },
        series: [{
          name: '组电压',
          type: 'line',
          yAxisIndex: 0,
          showSymbol: true,
          symbol: 'emptyCircle',
          symbolSize: 2,
          emphasis: {
            focus: 'none',
            scale: 2
          },
          data: []
        }, {
          name: 'SOC',
          type: 'line',
          yAxisIndex: 0,
          showSymbol: true,
          symbol: 'emptyCircle',
          symbolSize: 2,
          emphasis: {
            focus: 'none',
            scale: 2
          },
          data: []
        }, {
          name: '组电流',
          type: 'line',
          yAxisIndex: 1,
          showSymbol: true,
          symbol: 'emptyCircle',
          symbolSize: 2,
          emphasis: {
            focus: 'none',
            scale: 2
          },
          data: []
        }]
      },

      basicBarData2: {
        colors: ['#E2DF6E', '#FF5E00'],
        legend: {
          show: true,
          color: '#000',
          data: ['温差', '压差'] // 曲线标题
        },
        xAxis: { // 横轴：当天完整时间轴 00:00:00 - 23:59:59
          type: 'time',
          color: '#000',
          min: null,
          max: null,
          data: []
        },
        yAxis: [{ // 纵轴
          type: 'value',
          name: '温差：℃',
          position: 'left',
          alignTicks: true,
          emptyRange: { min: 0, max: 5, interval: 1 },
          nameTextStyle: {
            fontSize: 14,
            fontWeight: 'bold'
          },
          axisLine: {
            show: true,
            lineStyle: {
              color: '#000'
            }
          },
          axisLabel: {
            formatter: value => formatAxisLabel(value, 1, '℃')
          }
        }, {
          type: 'value',
          name: '压差：V',
          position: 'right',
          alignTicks: true,
          emptyRange: { min: 0, max: 0.5, interval: 0.1 },
          nameTextStyle: {
            fontSize: 14,
            fontWeight: 'bold'
          },
          axisLine: {
            show: true,
            lineStyle: {
              color: '#000'
            }
          },
          axisLabel: {
            formatter: value => formatAxisLabel(value, 3, 'V')
          }
        }],
        showSeriesLable: 'true', // 是否显示曲线上的文字，为字符串'true'时显示
        grid: {
          bottom: 48
        },
        tooltip: {
          trigger: 'axis',
          triggerOn: 'mousemove',
          confine: true,
          axisPointer: {
            type: 'line',
            snap: true
          }
        },
        series: [{
          name: '温差',
          type: 'line',
          yAxisIndex: 0,
          showSymbol: true,
          symbol: 'emptyCircle',
          symbolSize: 2,
          emphasis: {
            focus: 'none',
            scale: 2
          },
          data: []
        }, {
          name: '压差',
          type: 'line',
          yAxisIndex: 1,
          showSymbol: true,
          symbol: 'emptyCircle',
          symbolSize: 2,
          emphasis: {
            focus: 'none',
            scale: 2
          },
          data: []
        }]
      },

      basicBarData3: {
        colors: ['#22c4ff', '#ffaaff'],
        tooltip: {
          trigger: 'axis',
          axisPointer: {
            type: 'shadow',
            shadowStyle: {
              color: 'rgba(0,0,0,0.1)' // 白色半透明阴影
            }
          }
        },
        legend: {
          show: true,
          color: '#000',
          data: ['累计放电量', '累计充电量'] // 曲线标题
        },
        xAxis: { // 横轴
          color: '#000',
          data: []
        },
        yAxis: [{ // 纵轴
          type: 'value',
          name: '电量：AH',
          position: 'left',
          alignTicks: true,
          emptyRange: { min: 0, max: 1000, interval: 200 },
          axisLine: {
            show: true,
            lineStyle: {
              color: '#000'
            }
          },
          axisLabel: {
            formatter: '{value} AH'
          }
        }],
        showSeriesLable: 'true', // 是否显示曲线上的文字，为字符串'true'时显示
        series: [
          {
            name: '累计放电量',
            type: 'bar',
            label: {
              show: true,
              position: 'top',
              color: '#000'
            },
            yAxisIndex: 0,
            data: []
          },
          {
            name: '累计充电量',
            type: 'bar',
            label: {
              show: true,
              position: 'top',
              color: '#000'
            },
            yAxisIndex: 0,
            data: []
          }]
      },

      dcxx1List: [
        { title: '组电压', msg: '缺失' },
        { title: 'SOC', msg: '缺失' },
        { title: '组电流', msg: '缺失' },
        { title: 'SOH', msg: '缺失' }
      ],
      dcxx2List: [
        { title: '健康状态', msg: '缺失' },
        { title: '剩余放电时长', msg: '-' },
        { title: '累计放电量', msg: '缺失' },
        { title: '循环次数', msg: '缺失' }
      ],

      realTimeWcData: '-', // 温差
      dczwdList: [], // 电池组温度列表

      realTimeYcData: '-', // 压差
      dtdyList: [], // 单体电压列表

      hjWd: '-', // 环境温度
      mosWd: '-', // MOS温度

      dataDate: '', // 实时数据日期
      startTime: '', // 开始时间
      endTime: '' // 结束时间
    }
  },
  watch: {
    realDataList() {
      this.init()
    }
  },
  mounted() {
    this.init()
  },
  beforeUnmount() {
    // 切换电池或关闭详情后，丢弃仍在返回的旧历史请求。
    this.loadSeq++
  },
  methods: {
    /**
     * 实时数据弹框
     * @returns {Promise<void>}
     */
    async init() {
      const loadSeq = ++this.loadSeq
      if (!this.realDataList || !this.realDataList.length) {
        this.showBasicInfo = false
        this.$emit('loading-change', false)
        return
      }
      this.$emit('loading-change', true)
      try {
        this.getRealTimeDataMsg()
        this.showBasicInfo = true
        this.setxAxisData()
        await this.getHistoryData(loadSeq)
      } finally {
        if (loadSeq === this.loadSeq) {
          this.$emit('loading-change', false)
        }
      }
    },
    isPointActive(val) {
      return val === 1 || val === '1' || Number(val) === 1
    },
    resolveBatteryChargeState(list) {
      const stateDefs = [
        { suffix: '浮充', msg: '浮充', img: floatChargeImage },
        { suffix: '充电', msg: '充电', img: chargingImage },
        { suffix: '放电', msg: '放电', img: dischargingImage }
      ]
      for (let i = 0; i < stateDefs.length; i++) {
        const def = stateDefs[i]
        const matched = list.find(item => {
          const name = item.labelName || item.name || ''
          return name.endsWith(def.suffix) && this.isPointActive(item.realTimeVal)
        })
        if (matched) {
          this.dzctMsg = def.msg
          this.dcztImg = def.img
          return
        }
      }
      this.dzctMsg = '--'
      this.dcztImg = null
    },
    /** 秒级时间戳 → 'YYYY-MM-DD HH:mm:ss' */
    formatSecToDateTime(sec) {
      const str = formatDateString(sec * 1000, 'timeStmapReset')
      return str || ''
    },
    formatSecToDayPrefix(sec) {
      const str = this.formatSecToDateTime(sec)
      return str.length >= 10 ? str.slice(0, 10) : ''
    },
    /** 历史查询结束时间（秒）：有有效 dataDate 用数据时刻，否则用当前时刻 */
    resolveHistoryEndSec() {
      if (this.dataDate && this.dataDate.length >= 19) {
        const ts = Math.floor(dateConvStamp(this.dataDate.slice(0, 19)) / 1000)
        if (Number.isFinite(ts) && ts > 0) {
          return ts
        }
      }
      return Math.floor(Date.now() / 1000)
    },
    /** 自然日 00:00:00（秒），以 endSec 所在日为基准 */
    resolveTodayStartSec(endSec) {
      const dayPrefix = this.formatSecToDayPrefix(endSec)
      if (!dayPrefix) {
        return endSec - 86400
      }
      const ts = Math.floor(dateConvStamp(`${dayPrefix} 00:00:00`) / 1000)
      return Number.isFinite(ts) && ts > 0 ? ts : endSec - 86400
    },
    /** 近 N 天起点：从 endSec 所在日往前 N 天的 00:00:00 */
    resolveLastDaysStartSec(endSec, days) {
      const endDateStr = this.formatSecToDateTime(endSec)
      if (!endDateStr) {
        return endSec - days * 86400
      }
      let cursor = endDateStr
      for (let i = 0; i < days; i++) {
        cursor = dateAddOrSubtract(-1, cursor, 'date', 'date')
      }
      const ts = Math.floor(dateConvStamp(`${cursor.slice(0, 10)} 00:00:00`) / 1000)
      return Number.isFinite(ts) && ts > 0 ? ts : endSec - days * 86400
    },
    setxAxisData() { // 设置x轴数据
      const endSec = this.resolveHistoryEndSec()
      this.endTime = endSec
      this.startTime = this.resolveLastDaysStartSec(endSec, 6)

      const endDateStr = this.formatSecToDateTime(endSec)
      const basicBarXData = [endDateStr]
      for (let i = 0; i < 6; i++) {
        const date = dateAddOrSubtract(-1, basicBarXData[basicBarXData.length - 1], 'date', 'date')
        basicBarXData.push(date)
      }
      const xAxisDate = []
      basicBarXData.forEach(item => {
        if (item && item.length >= 10) {
          xAxisDate.push(item.slice(5, 10))
        }
      })
      this.basicBarData3.xAxis.data = xAxisDate.reverse()
      this.syncChart12DayTimeAxis()
    },
    /** 图1/图2：固定当天 00:00 - 次日 00:00，12 个刻度（每 2 小时） */
    getChart12AxisTickValues(dayStartTs) {
      const twoHourMs = 2 * 3600 * 1000
      return Array.from({ length: 12 }, (_, i) => dayStartTs + i * twoHourMs)
    },
    syncChart12DayTimeAxis() {
      const range = this.getCurrentDayTimeRange()
      if (!range.dayStartTs) {
        return
      }
      const tickValues = this.getChart12AxisTickValues(range.dayStartTs)
      const axisMax = range.dayStartTs + 24 * 3600 * 1000
      const axisLabelFormatter = (value) => {
        const d = new Date(value)
        const pad = (n) => String(n).padStart(2, '0')
        const time = `${pad(d.getHours())}:${pad(d.getMinutes())}`
        const seconds = `${time}:${pad(d.getSeconds())}`
        if (d.getMilliseconds() !== 0) {
          return `${seconds}.${String(d.getMilliseconds()).padStart(3, '0')}`
        }
        return d.getSeconds() === 0 ? time : seconds
      }
      ;[this.basicBarData1, this.basicBarData2].forEach(chartData => {
        chartData.xAxis.type = 'time'
        chartData.xAxis.min = range.dayStartTs
        chartData.xAxis.max = axisMax
        chartData.xAxis.data = []
        chartData.xAxis.axisLabel = {
          rotate: 0,
          fontSize: 12,
          customValues: tickValues,
          formatter: axisLabelFormatter
        }
        chartData.xAxis.axisTick = {
          show: true,
          alignWithLabel: true,
          customValues: tickValues
        }
      })
    },
    /**
     * 获取实时数据
     */
    getRealTimeDataMsg() { // 获取实时数据
      this.dczwdList = []
      this.dtdyList = []
      this.dzctMsg = '--'
      this.dcztImg = null
      // 历史数据列表
      const historyLabelList = []
      // 实时数据列表
      const list = this.realDataList
      const environmentTemperaturePoint = list.find(item => {
        const pointName = item.labelName || item.name || ''
        return pointName.trim() === '环境温度'
      })
      const temperatureDifferencePoint = resolveBatteryPoint(list, {
        dataIdSuffix: TEMPERATURE_DIFFERENCE_POINT_DATA_ID,
        aliases: ['温差'],
        gatewayId: this.formData.equipmentCode
      })
      const pressurePoint = resolveBatteryPoint(list, {
        dataIdSuffix: PRESSURE_POINT_DATA_ID,
        aliases: ['压差'],
        gatewayId: this.formData.equipmentCode
      })
      const remainingDischargePoint = resolveBatteryPoint(list, {
        dataIdSuffix: 'alg_fangdianshichang',
        exactNames: ['放电时长'],
        gatewayId: this.formData.equipmentCode
      })
      this.realTimeWcData = '-'
      this.realTimeYcData = '-'
      this.hjWd = '-'
      this.resolveBatteryChargeState(list)
      const healthStatusItem = this.dcxx2List.find(infoItem => infoItem.title === '健康状态')
      if (healthStatusItem) {
        healthStatusItem.msg = '-'
      }
      const cycleCountItem = this.dcxx2List.find(infoItem => infoItem.title === '循环次数')
      const cycleCountPoint = resolveBatteryPoint(list, {
        aliases: ['循环次数'],
        gatewayId: this.formData.equipmentCode
      })
      if (cycleCountItem) {
        const rawCycleCount = cycleCountPoint && cycleCountPoint.realTimeVal
        const cycleCount = rawCycleCount == null || String(rawCycleCount).trim() === '' || rawCycleCount === '异常'
          ? NaN
          : Number(rawCycleCount)
        cycleCountItem.msg = Number.isFinite(cycleCount) ? `${cycleCount}次` : '-'
      }
      const remainingDischargeItem = this.dcxx2List.find(infoItem => infoItem.title === '剩余放电时长')
      if (remainingDischargeItem) {
        const rawDischarge = remainingDischargePoint && remainingDischargePoint.realTimeVal
        if (rawDischarge == null || String(rawDischarge).trim() === '' || rawDischarge === '异常') {
          remainingDischargeItem.msg = '-'
        } else {
          remainingDischargeItem.msg = rawDischarge === '999' ? '未放电' : `${rawDischarge}分钟`
        }
      }
      list.forEach(item => {
        // 电池信息框数据初始化
        this.dcxx1List.forEach(dcxx1Item => {
          if (item.name.indexOf('组电压') !== -1 && dcxx1Item.title === '组电压') {
            item.dataName = '组电压'
            // 添加到数据列表
            historyLabelList.push(item)
            // 刷新电池电压值
            if (item.realTimeVal === '异常') {
              dcxx1Item.msg = '-'
            } else {
              dcxx1Item.msg = parseFloat(item.realTimeVal).toFixed(2) + 'V'
            }
            // 实时数据日期
            this.dataDate = item.realTimeDate
          } else if (item.name === 'SOC' && dcxx1Item.title === 'SOC') {
            if (item.realTimeVal === '异常') {
              dcxx1Item.msg = '-'
            } else {
              dcxx1Item.msg = item.realTimeVal + '%'
            }
            item.dataName = 'SOC'
            historyLabelList.push(item)
          } else if (item.name === 'SOH' && dcxx1Item.title === 'SOH') {
            if (item.realTimeVal === '异常') {
              dcxx1Item.msg = '-'
            } else {
              dcxx1Item.msg = item.realTimeVal + '%'
            }
            if (healthStatusItem) {
              healthStatusItem.msg = resolveBatteryHealthStatus(item.realTimeVal)
            }
          } else if (item.name.indexOf('组电流') !== -1 && dcxx1Item.title === '组电流') {
            if (item.realTimeVal === '异常') {
              dcxx1Item.msg = '-'
            } else {
              dcxx1Item.msg = item.realTimeVal + 'A'
            }
            item.dataName = '组电流'
            historyLabelList.push(item)
          }
        })
        this.dcxx2List.forEach(dcxx2Item => {
          if (item.name.indexOf('累计放电量') !== -1 && dcxx2Item.title === '累计放电量') {
            if (item.realTimeVal === '异常') {
              dcxx2Item.msg = '-'
            } else {
              dcxx2Item.msg = item.realTimeVal + 'AH'
            }
          }
        })

        // 点位信息
        const pointName = item.labelName || item.name || ''
        // 获取名称中的温度值
        const wdMatch = pointName.match(/温度(\d+)$/)
        if (wdMatch) {
          const title = wdMatch[1]
          const val = item.realTimeVal === '异常' ? '-' : parseFloat(item.realTimeVal).toFixed(2) + '℃'
          const existing = this.dczwdList.find(d => d.title === title)
          if (existing) {
            existing.val = val
          } else {
            this.dczwdList.push({ title, val })
          }
        }
        const dyMatch = pointName.match(/电压(\d+)$/)
        if (dyMatch) {
          const title = dyMatch[1]
          const val = item.realTimeVal === '异常' ? '-' : parseFloat(item.realTimeVal).toFixed(3) + 'V'
          const existing = this.dtdyList.find(d => d.title === title)
          if (existing) {
            existing.val = val
          } else {
            this.dtdyList.push({ title, val })
          }
        } else if (item === environmentTemperaturePoint) {
          const environmentTemperature = Number(item.realTimeVal)
          this.hjWd = Number.isFinite(environmentTemperature)
            ? `${environmentTemperature.toFixed(1)}℃`
            : '-'
        } else if (item.name.indexOf('MOS温度') !== -1 && item.realTimeVal !== '异常') {
          this.mosWd = parseFloat(item.realTimeVal).toFixed(1) + '℃' // MOS温度
        } else if (item === temperatureDifferencePoint) {
          item.dataName = '温差'
          const temperatureDifference = Number(item.realTimeVal)
          this.realTimeWcData = Number.isFinite(temperatureDifference)
            ? `${temperatureDifference.toFixed(1)}℃`
            : '-'
          historyLabelList.push(item)
        } else if (item === pressurePoint) {
          item.dataName = '压差'
          const pressure = Number(item.realTimeVal)
          this.realTimeYcData = Number.isFinite(pressure) ? `${pressure.toFixed(3)}V` : '-'
          historyLabelList.push(item)
        } else if (item.name.indexOf('累计放电量') !== -1) {
          item.dataName = '累计放电量'
          historyLabelList.push(item)
        } else if (item.name.indexOf('累计充电量') !== -1) {
          historyLabelList.push(item)
          item.dataName = '累计充电量'
        }
      })

      this.dczwdList.sort((a, b) => Number(a.title) - Number(b.title))
      this.dtdyList.sort((a, b) => Number(a.title) - Number(b.title))
      this.labelList = list
      this.historyLabelList = historyLabelList
    },
    /**
     * 按 x 轴日期筛选，每天只保留最后一条历史数据（与 basicBarData3 近 7 日展示对齐）
     */
    buildBar3DailyLastData(seriesdata, xAxisDateLabels) {
      const lastByDay = {}
      seriesdata.forEach(yDataItem => {
        if (!yDataItem || !yDataItem.date) {
          return
        }
        const dayKey = yDataItem.date.slice(5, 10)
        const prev = lastByDay[dayKey]
        if (!prev || yDataItem.date >= prev.date) {
          lastByDay[dayKey] = yDataItem
        }
      })
      return xAxisDateLabels.map(dayLabel => lastByDay[dayLabel] || null)
    },
    applyBar3DailySeriesData(seriesItem, seriesdata, xAxisDateLabels) {
      const dailyData = this.buildBar3DailyLastData(seriesdata, xAxisDateLabels)
      seriesItem.data = dailyData.map(d => {
        if (!d) {
          return null
        }
        if (d.itemStyle) {
          return { value: d.value, itemStyle: d.itemStyle }
        }
        return d.value
      })
    },
    /** 上图1/2：当天 00:00:00 - 23:59:59 时间轴；结束时刻为 dataDate 或当前时刻 */
    getCurrentDayTimeRange() {
      const endSec = this.resolveHistoryEndSec()
      const dayPrefix = this.formatSecToDayPrefix(endSec)
      const dayStartStr = `${dayPrefix} 00:00:00`
      const dayEndStr = `${dayPrefix} 23:59:59`
      const dataEndStr = (this.dataDate && this.dataDate.length >= 19)
        ? this.dataDate.slice(0, 19)
        : this.formatSecToDateTime(endSec)
      return {
        startTime: this.resolveTodayStartSec(endSec),
        endTime: endSec,
        dayStartStr,
        dayEndStr,
        dataEndStr,
        dayStartTs: dateConvStamp(dayStartStr),
        dayEndTs: dateConvStamp(dayEndStr)
      }
    },
    filterChart12DayData(seriesdata) {
      const { dayStartStr, dataEndStr } = this.getCurrentDayTimeRange()
      if (!dayStartStr || !dataEndStr) {
        return seriesdata
      }
      return seriesdata.filter(point => {
        if (!point || !point.date) {
          return false
        }
        return point.date >= dayStartStr && point.date <= dataEndStr
      })
    },
    toChart12TimeSeriesValues(seriesdata) {
      return seriesdata.map(point => {
        const ts = dateConvStamp(point.date)
        if (point.itemStyle) {
          return { value: [ts, point.value], itemStyle: point.itemStyle }
        }
        return [ts, point.value]
      })
    },
    applyChart12SeriesData(chartData, seriesItem, seriesdata, dataName) {
      if (seriesItem.name !== dataName) {
        return
      }
      const daySeries = this.filterChart12DayData(seriesdata)
      seriesItem.data = this.toChart12TimeSeriesValues(daySeries)
    },
    buildHistoryRequestConfig(item, endSec, todayStartSec, weekStartSec) {
      let startTime
      let endTime = endSec
      switch (item.dataName) {
        case '组电压':
        case '组电流':
        case 'SOC':
        case '压差':
        case '温差':
          startTime = todayStartSec
          break
        case '累计放电量':
        case '累计充电量':
          startTime = weekStartSec
          break
        default:
          startTime = endSec - 86400
      }
      if (!Number.isFinite(startTime) || !Number.isFinite(endTime) || startTime >= endTime) {
        startTime = endSec - 86400
        endTime = endSec
      }
      return { item, startTime, endTime }
    },
    buildHistorySeries(data) {
      const seriesdata = []
      if (data?.error) throw new Error(data.error)
      if (!data || (data.times == null && data.values == null && data.status == null)) return seriesdata
      if (!Array.isArray(data.times) || !Array.isArray(data.values) || !Array.isArray(data.status) ||
          data.times.length !== data.values.length || data.times.length !== data.status.length) {
        throw new Error('历史数据返回格式异常')
      }
      const times = data.times.slice().reverse()
      const values = data.values.slice().reverse()
      const status = data.status.slice().reverse()
      values.forEach((item, index) => {
        if (status[index] < 0) {
          let value = 0
          if (index - 1 >= 0) {
            values[index] = values[index - 1]
            value = values[index - 1]
          }
          seriesdata.push({
            time: times[index],
            date: format0800CSTDateString(times[index]),
            value,
            itemStyle: { color: 'red' }
          })
        } else {
          seriesdata.push({
            time: times[index],
            date: format0800CSTDateString(times[index]),
            value: item
          })
        }
      })
      return seriesdata
    },
    async getHistoryDataBatch(requestList) {
      const res = await getdata({
        Apikey: this.formData.apiKey,
        Function: 'historydatapasstimelist',
        requests: requestList.map(({ item, startTime, endTime }) => ({
          labelId: item.labelId,
          dateStart: startTime,
          dateEnd: endTime,
          pageNum: 1,
          pageSize: 10000,
          timeType: 'second',
          timeLength: '30'
        }))
      })
      if (Number(res?.code) !== 200) throw new Error(res?.msg || '批量历史数据查询失败')
      const dataList = res.return && res.return.data
      if (!Array.isArray(dataList) || dataList.length !== requestList.length) {
        throw new Error('批量历史数据返回格式异常')
      }
      return requestList.map((requestItem, index) => ({
        item: requestItem.item,
        seriesdata: this.buildHistorySeries(dataList[index])
      }))
    },
    async getHistoryDataFallback(requestList) {
      return Promise.all(requestList.map(async requestItem => {
        try {
          const res = await getdata({
            Apikey: this.formData.apiKey,
            Function: 'historydatapasstime',
            Pagenum: 1,
            Pagesize: 10000,
            Datestart: requestItem.startTime,
            Dateend: requestItem.endTime,
            labelID: requestItem.item.labelId,
            timeType: 'second',
            timeLength: 30
          })
          if (Number(res?.code) !== 200) throw new Error(res?.msg || '历史数据查询失败')
          return {
            item: requestItem.item,
            seriesdata: this.buildHistorySeries(res.return)
          }
        } catch (error) {
          // 保留失败标记，由上层统一提示，不再把故障当成成功的空曲线。
          return { item: requestItem.item, seriesdata: [], error: error.message || '历史数据查询失败' }
        }
      }))
    },
    async getHistoryData(loadSeq = this.loadSeq) { // 获取历史数据
      const endSec = this.resolveHistoryEndSec()
      const todayStartSec = this.resolveTodayStartSec(endSec)
      const weekStartSec = this.resolveLastDaysStartSec(endSec, 6)
      const requestList = this.historyLabelList.map(item => {
        return this.buildHistoryRequestConfig(item, endSec, todayStartSec, weekStartSec)
      })
      if (!requestList.length) {
        this.basicBardataChangeKey++
        return
      }

      let historyDataList
      try {
        historyDataList = await this.getHistoryDataBatch(requestList)
      } catch (error) {
        console.error(error)
        historyDataList = await this.getHistoryDataFallback(requestList)
      }
      if (loadSeq !== this.loadSeq) return
      const failures = historyDataList.filter(item => item.error)
      if (failures.length) {
        this.$message.error(`有 ${failures.length} 个测点的历史数据查询失败，请检查历史存储服务`)
      }

      historyDataList.forEach(({ item, seriesdata }) => {
        switch (item.dataName) {
          case '组电压':
          case '组电流':
          case 'SOC':
            this.basicBarData1.series.forEach(seriesItem => {
              this.applyChart12SeriesData(this.basicBarData1, seriesItem, seriesdata, item.dataName)
            })
            break
          case '压差':
          case '温差':
            this.basicBarData2.series.forEach(seriesItem => {
              this.applyChart12SeriesData(this.basicBarData2, seriesItem, seriesdata, item.dataName)
            })
            break
          case '累计放电量':
          case '累计充电量':
            {
              const xAxisDateLabels = this.basicBarData3.xAxis.data
              this.basicBarData3.series.forEach(seriesItem => {
                if (seriesItem.name === item.dataName) {
                  this.applyBar3DailySeriesData(seriesItem, seriesdata, xAxisDateLabels)
                }
              })
            }
            break
        }
      })
      this.basicBardataChangeKey++
    }
  }
}
</script>

<style lang="stylus" scoped>
  boxShadow = 0 0 3px 2px rgba(0,109,255, 0.4)
  box2Shadow = 2px 2px 3px 2px rgba(0, 0, 0, 0.2)

  .sssjCss {
    width 100%
    display flow-root
    .chartContentCss {
      float left
      margin 0 0.2vw
      width 70%
      height 44vw
      box-shadow boxShadow
      .chartCss {
        width 100%
        height 100%
        .multipleYAxesCss {
          margin-top 1vw
          width 98%
          height 13.5vw
          border-bottom 1px solid #F6F5FA
        }
      }
    }
    .dcDetailsMsgContentCss {
      float left
      margin-left 0.4vw
      width 29%
      .dcDetailsMsgCss {
        width 100%
        height 11vw
        border-radius 5px
        box-shadow boxShadow
        .dcTitleContentCss {
          width 100%
          height 2vw
          line-height 2vw
          .dcTitleCss {
            float left
            font-size 1vw
            font-weight bold
          }
        }
        .dcztContentCss {
          display flex
          align-items center
          padding 0.1vw 0 0.1vw 1vw
          width 100%
          height 1.5vw
          line-height 1.5vw
          .dcztStatusContentCss {
            display flex
            align-items center
            width 50%
            min-width 0
          }
          .dcTZtTitleCss {
            float left
            margin 0 0.5vw 0 0
            font-size 0.8vw
            height 0.8vw
          }
          .dcXCss {
            float left
            margin 0.1vw 0.1vw 0 0
            width 1.5vw
            height 1.2vw
          }
          .dcTZtCss {
            float left
            font-size 0.8vw
          }
          .dcTypeContentCss {
            display flex
            align-items center
            width 50%
            min-width 0
            font-size 0.8vw
            .dcTypeTitleCss {
              flex none
              margin-right 0.5vw
            }
            .dcTypeMsgCss {
              min-width 0
              overflow hidden
              text-overflow ellipsis
              white-space nowrap
            }
          }
        }
        .dcDetialsGzMsgCss {
          padding-top 0.5vw
          width 100%
          .dcGzMsgContentCss {
            float left
            padding-left 1vw
            width 50%
            height 1.6vw
            line-height 1.6vw
            .dcGzTitelCss {
              float left
              margin-right 0.5vw
            }
            .dcGzMsgCss {
              float left
            }
          }
        }
      }
      .mbxxDetailsMsgCss {
        margin 0.5vw 0 0.5vw 0
        width 100%
        height 13.75vw
        border-radius 5px
        box-shadow boxShadow
        .dczwTitleContentCss {
          padding-left 1vw
          width 100%
          height 2vw
          line-height 2vw
          .dczwTitleCss {
            float left
            font-size 1vw
            font-weight bold
          }
          .dczwXCss {
            float right
            margin 0.1vw 1vw 0 0
            font-size 0.8vw
          }
        }
        .mbxxContentCss {
          margin 0.3vw 0 0 2%
          width 98%
          height calc(13.75vw - 2.3vw)
          overflow-y auto
          &::after {
            content ''
            display block
            clear both
          }
          .mbxxCss {
            float left
            margin 0 2.16% 0.61vw 0
            width 14.5%
            height 2.25vw
            text-align center
            font-size 0.8vw
            box-shadow box2Shadow
            .mbxxTitleCss {
              width 100%
              height 1.1vw
              line-height 1.1vw
            }
            .mbxxValCss {
              width 100%
              line-height 1vw
            }
          }
        }
      }
      .qtwdDetailsMsgCss {
        margin 0.5vw 0
        height 4vw
        border-radius 5px
        box-shadow boxShadow
        .qtwdTitleCss {
          padding-left 1vw
          width 100%
          height 2vw
          line-height 2vw
          font-size 1vw
          font-weight bold
        }
        .qtwdMsgContentCss {
          padding-left 20%
          width 100%
          height 1.5vw
          line-height 1.5vw
          font-size 0.8vw
          .qtwdMsgCss {
            float left
            width 40%
            .qtwdMsgTitleCss {
              float left
            }
            .qtwdMsgDataCss {
              float left
            }
          }
        }
      }
    }
  }

  img {
    width 100%
    height 100%
  }
</style>
