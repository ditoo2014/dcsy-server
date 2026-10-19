<template>
  <div class="hisDataCss">
    <div v-if="isDevelopment" class="historyNotesCss">
      <el-tooltip
        class="item"
        effect="dark"
        placement="top"
        content="历史数据显示的内容为【名称中包含‘电池状态’的数据点】"
      >
        <el-button size="small" type="primary">备注信息</el-button>
      </el-tooltip>
    </div>
    <div class="parameterSelectionContentCss">
      <div class="titleCss">【参数选择】</div>
      <div class="parameterSelectionCss">
        <div class="parameterSelectionTitleCss">电池组参数：</div>
        <div class="parameterSelectionOptionsCss">
          <MyCheckbox
            :check-list="batteryParameterOptions"
            @select-change="handleBatteryParameterSelectionChange"
          />
        </div>
        <div class="parameterSelectionFooterCss" />
      </div>
      <div class="parameterSelectionCss">
        <div class="parameterSelectionTitleCss">单体电压：</div>
        <div class="parameterSelectionOptionsCss">
          <MyCheckbox
            :check-list="cellVoltageOptions"
            @select-change="handleCellVoltageSelectionChange"
          />
        </div>
        <div class="parameterSelectionFooterCss" />
      </div>
    </div>
    <div class="sliceSelectionContentCss">
      <div class="titleCss">【切片快速选择】</div>
      <div
        v-loading="dischargeSliceLoading"
        class="sliceSelectionCss"
        element-loading-background="rgba(255, 255, 255, 0.75)"
      >
        <div class="quickSelectionTitleCss">放电切片快速选择</div>
        <div class="sliceHistoryContentCss">
          <div class="sliceHistoryCss">
            <div class="sliceHistoryTitleCss">历史放电记录</div>
            <div class="sliceHistoryDateContentCss">
              <div class="sliceHistoryDateCss">
                <el-date-picker
                  v-model="dischargeTimeRange"
                  size="small"
                  type="datetimerange"
                  start-placeholder="开始日期"
                  end-placeholder="结束日期"
                  :disabled="dischargeSliceLoading"
                  :disabled-date="isDischargeDateDisabled"
                  @calendar-change="handleDischargeCalendarChange"
                  @change="timeChange('放电切片快速选择')"
                />
              </div>
            </div>
            <div class="dischargeCountCss">放电次数<span class="dischargeCountValueCss">{{ allDischargeRecords.length }}</span>次</div>
          </div>
          <div class="dischargeRecordsContentCss">
            <div class="dischargeRecordsTitleCss">放电记录</div>
            <div class="dischargeRecordsCss">
              <div class="dischargeRecordsTabCss">
                <div v-if="dischargeRecords.length > 0" class="dischargeRecordsListCss">
                  <div
                    v-for="(item, index) in dischargeRecords"
                    :key="index"
                    class="dischargeRecordItemCss cursorPointer"
                    :class="item.selected ? 'dischargeRecordSelectedCss': ''"
                    @click="clickTable1(item)"
                  >{{ item.dateArr }}</div>
                </div>
                <div v-else class="dischargeRecordsListCss emptyDataCss">无</div>
                <div class="pagCss">
                  <Pagination
                    v-show="total1>0"
                    v-model:page="page1"
                    v-model:limit="limit1"
                    padding="0"
                    layout="prev, pager, next"
                    :small="true"
                    :total="total1"
                    :page-sizes="[2, 20, 30, 50, 100]"
                    @pagination="getMsgList1"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="sliceSelectionFooterCss" />
      </div>
      <div
        v-loading="chargeSliceLoading"
        class="sliceSelectionCss"
        element-loading-background="rgba(255, 255, 255, 0.75)"
      >
        <div class="quickSelectionTitleCss">充电切片快速选择</div>
        <div class="sliceHistoryContentCss">
          <div class="sliceHistoryCss">
            <div class="sliceHistoryTitleCss">历史充电记录</div>
            <div class="sliceHistoryDateContentCss">
              <div class="sliceHistoryDateCss">
                <el-date-picker
                  v-model="chargeTimeRange"
                  size="small"
                  type="datetimerange"
                  start-placeholder="开始日期"
                  end-placeholder="结束日期"
                  :disabled="chargeSliceLoading"
                  :disabled-date="isChargeDateDisabled"
                  @calendar-change="handleChargeCalendarChange"
                  @change="timeChange('充电切片快速选择')"
                />
              </div>
            </div>
            <div class="dischargeCountCss">充电次数<span class="dischargeCountValueCss">{{ allChargeRecords.length }}</span>次</div>
          </div>
          <div class="dischargeRecordsContentCss">
            <div class="dischargeRecordsTitleCss">充电记录</div>
            <div class="dischargeRecordsCss">
              <div class="dischargeRecordsTabCss">
                <div v-if="chargeRecords.length > 0" class="dischargeRecordsListCss">
                  <div
                    v-for="(item, index) in chargeRecords"
                    :key="index"
                    class="dischargeRecordItemCss cursorPointer"
                    :class="item.selected ? 'dischargeRecordSelectedCss': ''"
                    @click="clickTable2(item)"
                  >{{ item.dateArr }}</div>
                </div>
                <div v-else class="dischargeRecordsListCss emptyDataCss">无</div>
                <div class="pagCss">
                  <Pagination
                    v-show="total2>0"
                    v-model:page="page2"
                    v-model:limit="limit2"
                    padding="0"
                    layout="prev, pager, next"
                    :small="true"
                    :total="total2"
                    :page-sizes="[2, 20, 30, 50, 100]"
                    @pagination="getMsgList2"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="sliceSelectionFooterCss" />
      </div>
    </div>

    <div
      v-loading="curveSliceLoading"
      class="trendCurveContentCss"
      element-loading-background="rgba(255, 255, 255, 0.75)"
    >
      <div class="trendCurveHeaderCss">
        <div class="trendCurveTitleCss">【变化曲线】</div>
        <div class="trendCurveLegendCss">
          <div class="statusIconCss statusIcon1Css" />
          <div class="statusNameCss">浮充</div>
          <div class="statusIconCss statusIcon2Css" />
          <div class="statusNameCss">放电</div>
          <div class="statusIconCss statusIcon3Css" />
          <div class="statusNameCss">充电</div>
        </div>
        <div class="queryIntervalDateCss">
          <el-date-picker
            v-model="timeArr"
            type="datetimerange"
            size="small"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            :disabled="curveSliceLoading"
            :disabled-date="curveDisabledDate"
            @calendar-change="curveCalendarChange"
            @change="timeChange('')"
          />
        </div>
        <div class="queryIntervalCss">
          <div class="queryIntervalTitleCss">查询间隔：</div>
          <div class="inputContentCss">
            <el-input
              v-model="timeLength"
              placeholder="间隔长度"
              type="number"
              size="small"
              :min="0"
            />
          </div>
          <div class="inputContentCss">
            <el-select
              v-model="timeType"
              size="small"
            >
              <el-option
                v-for="item in timeTypeList"
                :key="item.key"
                :label="item.name"
                :value="item.key"
              />
            </el-select>
          </div>
        </div>
        <div class="exportCss">
          <el-button
            size="small"
            type="primary"
            :icon="$legacyIcon('el-icon-download')"
            @click="exportExcel"
          >导出数据</el-button>
        </div>
      </div>
      <div class="multipleYAxesContentCss">
        <MixChart id="minChart" background-color="" height="100%" width="100%" :chart-data="pointChartData" />
        <div class="smultipleYAxesFooterCss" />
      </div>
    </div>
  </div>
</template>

<script>
import MixChart from '@/components/sharedComponents/echarts/MixChart.vue'
import Pagination from '@/components/Pagination' // 分页组件
import MyCheckbox from '@/components/sharedComponents/myCheckbox/MyCheckbox.vue'
import { getdata } from '@/api/monitoring.js'
import {
  TOTAL_DISCHARGE_CAPACITY_POINT_DATA_ID,
  TOTAL_CHARGE_CAPACITY_POINT_DATA_ID,
  PRESSURE_POINT_DATA_ID,
  TEMPERATURE_DIFFERENCE_POINT_DATA_ID,
  resolveBatteryPoint
} from '@/utils/batteryPoints.js'
import { getTime, formatDateString, dateConvStamp, format0800CSTDateString } from '@/utils/utils.js'
import { parseUserInfo } from '@/utils/safeJson.js'

// 接口统一使用秒；界面仍保留原版的秒、分钟、小时和天选择。
const HISTORY_INTERVAL_SECONDS = { second: 1, minute: 60, hour: 3600, day: 86400 }
const DEFAULT_HISTORY_COLOR = 'rgba(122,247,190)'
const HISTORY_QUERY_PAGE_SIZE = 100000

export default {
  components: {
    MixChart,
    Pagination,
    MyCheckbox
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
  data() {
    return {
      isDevelopment: import.meta.env.DEV, // 是否是开发者模式
      userInfo: parseUserInfo({}),
      isAdmin: false, // 是否是最高权限管理员

      allBatteryParametersChecked: true,
      batteryParameterOptions: [
        { checked: true, name: '组电压' }, { checked: true, name: '组电流' },
        { checked: false, name: '温度1' }, { checked: false, name: '温度2' },
        { checked: false, name: '温度3' }, { checked: false, name: '温度4' },
        { checked: false, name: '环境温度' }, { checked: false, name: 'MOS温度' },
        { checked: false, name: 'SOC', label: 'soc' }, { checked: false, name: 'SOH', label: 'soh' },
        { checked: false, name: '充电量', label: '充电电量' }, { checked: false, name: '放电量', label: '放电电量' },
        { checked: false, name: '温差' }
      ],
      selectedBatteryParameters: [],
      batteryParametersIndeterminate: true,

      allCellVoltagesChecked: true,
      cellVoltageOptions: [
        { checked: false, name: '1节', label: '电压1' }, { checked: false, name: '2节', label: '电压2' },
        { checked: false, name: '3节', label: '电压3' }, { checked: false, name: '4节', label: '电压4' },
        { checked: false, name: '5节', label: '电压5' }, { checked: false, name: '6节', label: '电压6' },
        { checked: false, name: '7节', label: '电压7' }, { checked: false, name: '8节', label: '电压8' },
        { checked: false, name: '9节', label: '电压9' }, { checked: false, name: '10节', label: '电压10' },
        { checked: false, name: '11节', label: '电压11' }, { checked: false, name: '12节', label: '电压12' },
        { checked: false, name: '13节', label: '电压13' }, { checked: false, name: '14节', label: '电压14' },
        { checked: false, name: '15节', label: '电压15' }, { checked: false, name: '16节', label: '电压16' },
        { checked: false, name: '压差' }
      ],
      selectedCellVoltages: [],
      cellVoltagesIndeterminate: true,

      timeArr: [], // 时间选择
      curvePickingStart: null,
      curveSliceLoading: false,
      startTime: '',
      endTime: '',
      dischargeTimeRange: [], // 放电时间范围
      dischargePickingStart: null,
      dischargeSliceLoading: false,
      dischargeStartTime: '',
      dischargeEndTime: '',
      chargeTimeRange: [], // 充电时间范围
      chargePickingStart: null,
      chargeSliceLoading: false,
      chargeStartTime: '',
      chargeEndTime: '',

      allDischargeRecords: [],
      dischargeRecords: [],
      total1: 0,
      page1: 1,
      limit1: 10,

      allChargeRecords: [],
      chargeRecords: [],
      total2: 0,
      page2: 1,
      limit2: 10,

      timeTypeList: [ // 时间类型间隔列表
        { key: 'second', name: '秒' },
        { key: 'minute', name: '分钟' },
        { key: 'hour', name: '小时' },
        { key: 'day', name: '天' }
      ],
      timeType: 'second', // 时间间隔类型
      timeLength: 5, // 时间间隔长度

      pointChartData: {
        title: {
          text: '' // 曲线图标题名称
        },
        valueLabelMode: 'adaptive', // 与其他曲线统一按可视宽度等间隔显示数值
        legend: {
          data: [''] // 曲线标题
        },
        xAxis: { // 横轴
          data: ['0']
        },
        yAxis: [ // 纵轴
          { name: '' }
        ],
        showSeriesLable: 'true', // 是否显示曲线上的文字，为字符串'true'时显示
        series: [{ // 纵坐标数据
          name: '',
          data: [{
            name: '',
            value: 0,
            color: 'rgba(122,247,190)', // 线和文字颜色
            limitValue: 1, // 限值
            fixed: 2 // 显示数据小数点保留位数
          }],
          color: 'rgba(122,247,190)', // 线和文字颜色
          limitValue: 1, // 限值
          fixed: 2 // 显示数据小数点保留位数
        }]
      },

      floatChargeStateCode: 0, // value 为 0 时，代表浮充
      dischargeStateCode: 1, // value 为 1 时，代表放电
      chargeStateCode: 2, // value 为 2 时，代表充电
      batteryStatePoint: {}, // 电池状态状态信息
      batteryStatePoints: [], // 浮充、充电、放电独立状态数据点
      chartBatteryStateIntervals: [], // 当前曲线时间范围内的电池状态区间
      selectedSliceInterval: null, // 当前点击的充放电切片区间，作为曲线颜色的唯一依据
      currentBatteryStateCode: null, // 当前实时电池状态
      currentBatteryStateTimeStmap: null, // 当前实时电池状态采集时间
      historyEpoch: 0,
      historySequences: { curve: 0, charge: 0, discharge: 0 },
      intervalRefreshTimer: null,
      historyDisposed: false
    }
  },
  watch: {
    realDataList() {
      this.init()
    },
    timeLength: 'scheduleCurveRefresh',
    timeType: 'scheduleCurveRefresh'
  },
  mounted() {
    this.isAdmin = this.userInfo.isadmin === 1

    const currentTimestampSec = Math.floor(Date.now() / 1000)
    const nowDay = formatDateString(currentTimestampSec, 'timeStmapReset')
    const startDate = formatDateString(currentTimestampSec - 7 * 24 * 3600, 'timeStmapReset')
    this.dischargeTimeRange = [startDate, nowDay]
    this.chargeTimeRange = [startDate, nowDay]
    this.init()
  },
  beforeUnmount() {
    this.historyDisposed = true
    this.historyEpoch++
    clearTimeout(this.intervalRefreshTimer)
  },
  methods: {
    getHistoryIntervalSeconds() {
      const value = Number(this.timeLength)
      const unit = HISTORY_INTERVAL_SECONDS[this.timeType]
      const seconds = value * unit
      if (!Number.isFinite(seconds) || seconds <= 0 || !Number.isInteger(seconds)) {
        throw new Error('查询间隔必须为大于零的整秒，请检查间隔长度和单位')
      }
      return seconds
    },
    beginHistoryRequest(area = 'curve') {
      return { area, epoch: this.historyEpoch, sequence: ++this.historySequences[area] }
    },
    isCurrentHistoryRequest(request) {
      return !this.historyDisposed && request.epoch === this.historyEpoch &&
        request.sequence === this.historySequences[request.area]
    },
    reportHistoryError(error, request) {
      if (this.isCurrentHistoryRequest(request)) {
        this.$message.error(error.message || '历史数据查询失败，请稍后重试')
      }
    },
    async queryHistory(params) {
      const res = await getdata(params)
      if (Number(res?.code) !== 200) throw new Error(res?.msg || '历史数据查询失败')
      const data = res.return || {}
      if (data.error) throw new Error(data.error)
      if ((data.times != null && !Array.isArray(data.times)) ||
          (data.values != null && !Array.isArray(data.values)) ||
          (data.times?.length || 0) !== (data.values?.length || 0)) {
        throw new Error('历史数据返回格式异常')
      }
      return data
    },
    scheduleCurveRefresh() {
      // 合并连续输入，先使旧请求失效，避免较慢的旧间隔覆盖新曲线。
      this.historySequences.curve++
      clearTimeout(this.intervalRefreshTimer)
      this.intervalRefreshTimer = setTimeout(() => {
        if (this.historyDisposed) return
        const selected = this.dischargeRecords.find(item => item.selected) || this.chargeRecords.find(item => item.selected)
        if (selected && this.selectedSliceInterval) {
          this.selectSliceRecord(selected, this.selectedSliceInterval.code)
        } else {
          this.timeChange('')
        }
      }, 250)
    },
    async init() {
      const epoch = ++this.historyEpoch
      await this.getRealTimeDataMsg() // 获取实时数据
      if (this.historyDisposed || epoch !== this.historyEpoch) return
      await this.timeChange('')
      if (this.historyDisposed || epoch !== this.historyEpoch) return
      await this.timeChange('放电切片快速选择')
      if (this.historyDisposed || epoch !== this.historyEpoch) return
      await this.timeChange('充电切片快速选择')
    },
    async getRealTimeDataMsg() { // 获取实时数据
      const list = Array.isArray(this.realDataList) ? this.realDataList : []
      this.buildSelectLists(list)
      this.batteryStatePoint = {}
      this.batteryStatePoints = this.resolveBatteryStatePoints(list)
      const currentStateMsg = this.resolveCurrentBatteryStateMsg(this.batteryStatePoints)
      this.currentBatteryStateCode = currentStateMsg.code
      this.currentBatteryStateTimeStmap = currentStateMsg.timeStmap
      list.forEach(item => {
        const pointName = item.labelName || item.name || ''
        item.name = pointName
        if (pointName === '电池状态') {
          this.batteryStatePoint = item
        }
      })
    },
    buildSelectLists(list) {
      const previousBatteryParameterOptions = new Map(this.batteryParameterOptions.map(item => [item.name, item]))
      const previousCellVoltageOptions = new Map(this.cellVoltageOptions.map(item => [item.name, item]))
      const temperaturePoints = new Map()
      const voltagePoints = new Map()

      const getPointName = item => item.labelName || item.name || ''
      list.forEach(item => {
        const pointName = getPointName(item)
        const temperatureMatch = pointName.match(/温度(\d+)$/)
        const voltageMatch = pointName.match(/电压(\d+)$/)
        if (temperatureMatch) {
          temperaturePoints.set(Number(temperatureMatch[1]), item)
        }
        if (voltageMatch) {
          voltagePoints.set(Number(voltageMatch[1]), item)
        }
      })

      const findPoint = aliases => {
        const normalizedAliases = aliases.map(alias => alias.toLowerCase())
        return list.find(item => {
          const pointName = getPointName(item).toLowerCase()
          return normalizedAliases.some(alias => pointName.indexOf(alias) !== -1)
        })
      }
      const findExactPoint = pointName => {
        return list.find(item => getPointName(item).trim() === pointName)
      }
      const createOption = (name, defaultChecked, previousOptions, point, label) => {
        const previousOption = previousOptions.get(name)
        const option = {
          checked: previousOption ? previousOption.checked : defaultChecked,
          name
        }
        if (label) option.label = label
        if (point) {
          point.title = name
          option.labelDetails = point
        }
        return option
      }

      const batteryParameterOptions = [
        createOption('组电压', true, previousBatteryParameterOptions, findPoint(['组电压'])),
        createOption('组电流', true, previousBatteryParameterOptions, findPoint(['组电流']))
      ]
      Array.from(temperaturePoints.keys()).sort((a, b) => a - b).forEach(index => {
        batteryParameterOptions.push(createOption(`温度${index}`, false, previousBatteryParameterOptions, temperaturePoints.get(index)))
      })
      const temperatureDifferencePoint = resolveBatteryPoint(list, {
        dataIdSuffix: TEMPERATURE_DIFFERENCE_POINT_DATA_ID,
        gatewayId: this.formData.equipmentCode
      })
      const totalChargePoint = resolveBatteryPoint(list, {
        dataIdSuffix: TOTAL_CHARGE_CAPACITY_POINT_DATA_ID,
        gatewayId: this.formData.equipmentCode
      })
      const totalDischargePoint = resolveBatteryPoint(list, {
        dataIdSuffix: TOTAL_DISCHARGE_CAPACITY_POINT_DATA_ID,
        gatewayId: this.formData.equipmentCode
      })
      batteryParameterOptions.push(
        createOption('环境温度', false, previousBatteryParameterOptions, findExactPoint('环境温度')),
        createOption('MOS温度', false, previousBatteryParameterOptions, findPoint(['MOS温度'])),
        createOption('SOC', false, previousBatteryParameterOptions, findExactPoint('SOC'), 'soc'),
        createOption('SOH', false, previousBatteryParameterOptions, findPoint(['SOH']), 'soh'),
        createOption('充电量', false, previousBatteryParameterOptions, totalChargePoint, '充电电量'),
        createOption('放电量', false, previousBatteryParameterOptions, totalDischargePoint, '放电电量'),
        createOption('温差', false, previousBatteryParameterOptions, temperatureDifferencePoint)
      )

      const cellVoltageOptions = []
      Array.from(voltagePoints.keys()).sort((a, b) => a - b).forEach(index => {
        cellVoltageOptions.push(createOption(`${index}节`, false, previousCellVoltageOptions, voltagePoints.get(index), `电压${index}`))
      })
      const pressurePoint = resolveBatteryPoint(list, {
        dataIdSuffix: PRESSURE_POINT_DATA_ID,
        gatewayId: this.formData.equipmentCode
      })
      cellVoltageOptions.push(createOption('压差', false, previousCellVoltageOptions, pressurePoint))

      this.batteryParameterOptions = batteryParameterOptions
      this.cellVoltageOptions = cellVoltageOptions
    },
    isPointActive(val) {
      return val === 1 || val === '1' || Number(val) === 1
    },
    getBatteryStateDefinitions() {
      return [
        { suffix: '浮充', code: this.floatChargeStateCode },
        { suffix: '充电', code: this.chargeStateCode },
        { suffix: '放电', code: this.dischargeStateCode }
      ]
    },
    resolveBatteryStatePoints(list) {
      const dataList = Array.isArray(list) ? list : []
      const statePoints = []
      this.getBatteryStateDefinitions().forEach(stateDef => {
        dataList.forEach(labelDetails => {
          const pointName = labelDetails.labelName || labelDetails.name || ''
          if (pointName.endsWith(stateDef.suffix)) {
            statePoints.push({ ...stateDef, labelDetails })
          }
        })
      })
      return statePoints
    },
    getPointTimeStmap(item) {
      let timeStmap = Number(item.realTimeTimestamp)
      if (Number.isFinite(timeStmap) && timeStmap > 0) {
        return timeStmap > 1e12 ? timeStmap / 1000 : timeStmap
      }
      if (item.realTimeDate) {
        timeStmap = dateConvStamp(format0800CSTDateString(item.realTimeDate)) / 1000
        if (Number.isFinite(timeStmap)) {
          return timeStmap
        }
      }
      return null
    },
    resolveCurrentBatteryStateMsg(statePoints) {
      const pointList = Array.isArray(statePoints) ? statePoints : []
      const stateDefs = this.getBatteryStateDefinitions()
      for (const stateDef of stateDefs) {
        const activePoints = pointList.filter(item => item.code === stateDef.code && this.isPointActive(item.labelDetails.realTimeVal))
        if (activePoints.length) {
          const timeStmaps = activePoints.map(item => this.getPointTimeStmap(item.labelDetails)).filter(Number.isFinite)
          return {
            code: stateDef.code,
            timeStmap: timeStmaps.length ? Math.max(...timeStmaps) : null
          }
        }
      }
      return { code: null, timeStmap: null }
    },
    getBatteryStateColor(value) {
      if (value === null || value === undefined || value === '') {
        return null
      }
      switch (Number(value)) {
        case this.floatChargeStateCode: return '#339900'
        case this.dischargeStateCode: return '#ff0000'
        case this.chargeStateCode: return '#ffff00'
        default: return null
      }
    },
    normalizeBatteryStateCode(value) {
      if (value === null || value === undefined || value === '') {
        return null
      }
      const stateCode = Number(value)
      return this.getBatteryStateDefinitions().some(item => item.code === stateCode) ? stateCode : null
    },
    buildBatteryStateIntervals(times, values, options = {}) {
      const currentStateCode = this.normalizeBatteryStateCode(options.currentStateCode)
      const currentStateTimeStmap = options.currentStateTimeStmap === null || options.currentStateTimeStmap === undefined || options.currentStateTimeStmap === ''
        ? NaN
        : Number(options.currentStateTimeStmap)
      const rangeStartTimeStmap = Number(options.rangeStartTimeStmap)
      const rangeEndTimeStmap = Number(options.rangeEndTimeStmap)
      const currentStateIsInRange = currentStateCode !== null && Number.isFinite(currentStateTimeStmap) &&
        (!Number.isFinite(rangeStartTimeStmap) || currentStateTimeStmap >= rangeStartTimeStmap) &&
        (!Number.isFinite(rangeEndTimeStmap) || currentStateTimeStmap <= rangeEndTimeStmap)
      const samples = []
      times.forEach((timeItem, index) => {
        const time = format0800CSTDateString(timeItem)
        const timeStmap = dateConvStamp(time) / 1000
        if (Number.isFinite(timeStmap)) {
          samples.push({
            time,
            timeStmap,
            code: this.normalizeBatteryStateCode(values[index])
          })
        }
      })
      if (currentStateIsInRange) {
        samples.push({
          time: getTime(currentStateTimeStmap),
          timeStmap: currentStateTimeStmap,
          code: currentStateCode
        })
      }
      samples.sort((a, b) => a.timeStmap - b.timeStmap)

      const intervals = []
      let currentInterval = null
      const closeCurrentInterval = isLastInterval => {
        const isOngoing = Boolean(isLastInterval && currentStateIsInRange && currentInterval && currentInterval.code === currentStateCode)
        if (currentInterval && (currentInterval.startTimeStmap !== currentInterval.endTimeStmap || isOngoing)) {
          currentInterval.isOngoing = isOngoing
          intervals.push(currentInterval)
        }
        currentInterval = null
      }
      samples.forEach(sample => {
        if (!currentInterval || currentInterval.code !== sample.code) {
          closeCurrentInterval(false)
          if (sample.code !== null) {
            currentInterval = {
              code: sample.code,
              startDate: sample.time,
              endDate: sample.time,
              startTimeStmap: sample.timeStmap,
              endTimeStmap: sample.timeStmap
            }
          }
        } else {
          currentInterval.endDate = sample.time
          currentInterval.endTimeStmap = sample.timeStmap
        }
      })
      closeCurrentInterval(true)
      return intervals
    },
    getBatteryStateColorByTime(timeStmap) {
      const intervals = this.chartBatteryStateIntervals
      let left = 0
      let right = intervals.length - 1
      while (left <= right) {
        const middle = Math.floor((left + right) / 2)
        const interval = intervals[middle]
        if (timeStmap < interval.startTimeStmap) {
          right = middle - 1
        } else if (timeStmap > interval.endTimeStmap) {
          left = middle + 1
        } else {
          return this.getBatteryStateColor(interval.code)
        }
      }
      return null
    },
    exportExcel() {
      const chartData = this.pointChartData
      import('@/vendor/exportToExcel').then(excel => {
        const tHeader = ['时间']
        chartData.series.forEach(item => {
          tHeader.push(item.name)
        })
        const data = this.formatJson(chartData)
        excel.export_json_to_excel({
          header: tHeader,
          data,
          filename: 'table-list'
        })
      })
    },
    formatJson(chartData) {
      const xAxisData = chartData.xAxis.data
      const arr = [] // 声明一维数组
      for (let x = 0; x < xAxisData.length; x++) {
        arr[x] = [] // 声明二维
        for (let y = 0; y <= chartData.series.length; y++) {
          arr[x][y] = ''
        }
      }
      chartData.xAxis.data.forEach((item, index) => {
        if (arr[index]) {
          arr[index][0] = item.value
        }
      })
      chartData.series.forEach((item, index) => {
        item.data.forEach((dataItem, i) => {
          if (arr[i]) {
            arr[i][index + 1] = dataItem.value
          }
        })
      })
      return arr
    },
    getMsgList1(e) {
      this.page1 = e.page
      this.limit1 = e.limit
      this.updateChargeDischargeRecords()
    },
    getMsgList2(e) {
      this.page2 = e.page
      this.limit2 = e.limit
      this.updateChargeDischargeRecords()
    },
    async clickTable1(item) {
      await this.selectSliceRecord(item, this.dischargeStateCode)
    },
    async clickTable2(item) {
      await this.selectSliceRecord(item, this.chargeStateCode)
    },
    async selectSliceRecord(item, stateCode) {
      const request = this.beginHistoryRequest()
      this.curveSliceLoading = true
      try {
        const timeLength = this.getHistoryIntervalSeconds()
        this.dischargeRecords.forEach(tabItem => {
          tabItem.selected = false
        })
        this.chargeRecords.forEach(tabItem => {
          tabItem.selected = false
        })
        item.selected = true
        this.timeArr = [item.startDate, item.endDate]
        this.startTime = dateConvStamp(item.startDate) / 1000
        this.endTime = dateConvStamp(item.endDate) / 1000
        this.pointChartDataInit()
        this.selectedSliceInterval = {
          code: stateCode,
          startDate: item.startDate,
          endDate: item.endDate,
          startTimeStmap: this.startTime,
          endTimeStmap: this.endTime,
          isOngoing: Boolean(item.isOngoing)
        }
        this.chartBatteryStateIntervals = [this.selectedSliceInterval]
        const xAxisData = await this.getChartXAxisData({
          startTime: this.startTime,
          endTime: this.endTime,
          timeLength
        })
        if (!this.isCurrentHistoryRequest(request)) return
        this.pointChartData.xAxis.data = xAxisData
        await this.setChartData('电池状态', request) // 设置曲线数据
      } catch (error) {
        this.reportHistoryError(error, request)
      } finally {
        if (this.isCurrentHistoryRequest(request)) this.curveSliceLoading = false
      }
    },
    handleBatteryParameterSelectionChange(batteryParameterOptions) {
      this.batteryParameterOptions = batteryParameterOptions
      this.setChartData() // 设置曲线数据
    },
    handleCellVoltageSelectionChange(cellVoltageOptions) {
      this.cellVoltageOptions = cellVoltageOptions
      this.setChartData() // 设置曲线数据
    },
    async setChartData(referencePointName, request = this.beginHistoryRequest()) { // 设置曲线数据
      this.curveSliceLoading = true
      try {
        const timeLength = this.getHistoryIntervalSeconds()
        const selectList = []
        this.batteryParameterOptions.forEach(item => {
          if (item.checked && item.labelDetails) {
            selectList.push(item.labelDetails)
          }
        })
        this.cellVoltageOptions.forEach(item => {
          if (item.checked && item.labelDetails) {
            selectList.push(item.labelDetails)
          }
        })
        if (referencePointName !== '电池状态') { // 电池状态不是基准数据名称，则获取基准数据点历史数据
          if (this.selectedSliceInterval) {
            this.chartBatteryStateIntervals = [this.selectedSliceInterval]
          } else {
            await this.loadBatteryStateAndReferenceHistory('', request) // 获取电池状态历史数据 和 获取基准数据点历史数据
          }
        }
        if (!this.isCurrentHistoryRequest(request)) return
        const xAxisData = this.pointChartData.xAxis.data || []
        if (!xAxisData.length) {
          this.pointChartDataInit()
          return
        }

        const chartSeries = [] // 曲线需要展示的数据点
        this.pointChartData.series.forEach(serieItem => {
          let hasPoint = false // 被选择的数据点中是否在曲线图中存在
          selectList.forEach(item => {
            if (item.title === serieItem.name) {
              hasPoint = true
            }
          })
          if (hasPoint) {
            chartSeries.push(serieItem)
          }
        })
        for (let i = 0; i < selectList.length; i++) {
          const item = selectList[i]
          let hasPoint = false // 曲线中是否已存在数据点
          chartSeries.forEach(serieItem => {
            if (item.title === serieItem.name) {
              hasPoint = true
            }
          })
          if (!hasPoint) {
            const data = await this.queryHistory({
              Apikey: this.formData.apiKey,
              Function: 'historydatapasstime',
              Pagenum: 1,
              // 曲线与基准横轴使用相同上限，避免默认 24 小时范围后半段被截断。
              Pagesize: HISTORY_QUERY_PAGE_SIZE,
              Datestart: this.startTime,
              Dateend: this.endTime,
              labelID: item.labelId,
              timeType: 'second',
              timeLength
            })
            if (!this.isCurrentHistoryRequest(request)) return
            const seriesdata = []
            if (data) {
              if (data.times) {
                const times = data.times.slice().reverse() // 不修改接口原始数组
                const valueTime = [] // 时间戳数组
                times.forEach(timeItem => {
                  valueTime.push(dateConvStamp(format0800CSTDateString(timeItem)) / 1000)
                })
                const values = (data.values || []).slice().reverse()
                const valueByTime = new Map()
                valueTime.forEach((timeStmap, index) => {
                  valueByTime.set(timeStmap, values[index])
                })
                xAxisData.forEach(xAxisItem => {
                  const timeStmap = xAxisItem.timeStmap
                  const hasValue = valueByTime.has(timeStmap)
                  const stateColor = hasValue ? this.getBatteryStateColorByTime(timeStmap) : null
                  seriesdata.push({
                    // 状态只决定配色，缺少状态记录不能抹掉已经采集到的真实值。
                    value: hasValue ? valueByTime.get(timeStmap) : null,
                    itemStyle: hasValue ? { color: stateColor || DEFAULT_HISTORY_COLOR } : undefined
                  })
                })
              }
            }
            chartSeries.push({
              name: item.title,
              data: seriesdata,
              color: 'rgba(122,247,190)', // 线和文字颜色
              fixed: 2 // 显示数据小数点保留位数
            })
          }
        }
        if (chartSeries.length === 0) {
          this.pointChartData.series = [{
            name: '',
            data: new Array(xAxisData.length).fill(null),
            color: 'rgba(122,247,190)',
            limitValue: 1,
            fixed: 2
          }]
        } else {
          this.pointChartData.series = chartSeries
        }
      } catch (error) {
        if (this.isCurrentHistoryRequest(request)) this.pointChartDataInit()
        this.reportHistoryError(error, request)
      } finally {
        if (this.isCurrentHistoryRequest(request)) this.curveSliceLoading = false
      }
    },
    pointChartDataInit() {
      this.pointChartData.xAxis.data = []
      this.pointChartData.series = [{
        name: '',
        data: [null],
        color: 'rgba(122,247,190)',
        limitValue: 1,
        fixed: 2
      }]
    },
    async timeChange(type) {
      const request = this.beginHistoryRequest(type === '放电切片快速选择' ? 'discharge' : type === '充电切片快速选择' ? 'charge' : 'curve')
      let startTime = ''
      let endTime = ''
      if (type === '') {
        this.selectedSliceInterval = null
        this.dischargeRecords.forEach(tabItem => {
          tabItem.selected = false
        })
        this.chargeRecords.forEach(tabItem => {
          tabItem.selected = false
        })
        if (this.timeArr && this.timeArr.length !== 0) {
          this.timeArr = this.setSelectTime(this.timeArr)
          this.timeArr = this.limitRangeToOneMonth(this.timeArr)
          startTime = dateConvStamp(formatDateString(this.timeArr[0], 'formatDate')) / 1000
          endTime = dateConvStamp(formatDateString(this.timeArr[1], 'formatDate')) / 1000
        }
      } else if (type === '放电切片快速选择') {
        if (this.dischargeTimeRange && this.dischargeTimeRange.length !== 0) {
          this.dischargeTimeRange = this.setSelectTime(this.dischargeTimeRange)
          this.dischargeTimeRange = this.limitRangeToOneMonth(this.dischargeTimeRange)
          startTime = dateConvStamp(formatDateString(this.dischargeTimeRange[0], 'formatDate')) / 1000
          endTime = dateConvStamp(formatDateString(this.dischargeTimeRange[1], 'formatDate')) / 1000
        }
      } else if (type === '充电切片快速选择') {
        if (this.chargeTimeRange && this.chargeTimeRange.length !== 0) {
          this.chargeTimeRange = this.setSelectTime(this.chargeTimeRange)
          this.chargeTimeRange = this.limitRangeToOneMonth(this.chargeTimeRange)
          startTime = dateConvStamp(formatDateString(this.chargeTimeRange[0], 'formatDate')) / 1000
          endTime = dateConvStamp(formatDateString(this.chargeTimeRange[1], 'formatDate')) / 1000
        }
      }
      const nowDate = getTime() // 当前时间
      const nowDay = nowDate.slice(0, 11) + '23:59:59' // 今天最后一秒时间
      const startDate = nowDate.slice(0, 11) + '00:00:01'
      if (startTime === '') {
        startTime = dateConvStamp(startDate) / 1000
        endTime = dateConvStamp(nowDay) / 1000
        switch (type) {
          case '': {
            const nowTimestampSec = Math.floor(Date.now() / 1000)
            const endDate = getTime(nowTimestampSec)
            const startDate24h = getTime(nowTimestampSec - 24 * 3600)
            startTime = nowTimestampSec - 24 * 3600
            endTime = nowTimestampSec
            this.timeArr = [startDate24h, endDate]
          }
            break
          case '放电切片快速选择': {
            const nowTimestampSec = Math.floor(Date.now() / 1000)
            const endDate = getTime(nowTimestampSec)
            const startDate7d = getTime(nowTimestampSec - 7 * 24 * 3600)
            startTime = nowTimestampSec - 7 * 24 * 3600
            endTime = nowTimestampSec
            this.dischargeTimeRange = [startDate7d, endDate]
          }
            break
          case '充电切片快速选择': {
            const nowTimestampSec = Math.floor(Date.now() / 1000)
            const endDate = getTime(nowTimestampSec)
            const startDate7d = getTime(nowTimestampSec - 7 * 24 * 3600)
            startTime = nowTimestampSec - 7 * 24 * 3600
            endTime = nowTimestampSec
            this.chargeTimeRange = [startDate7d, endDate]
          }
            break
        }
      }
      if (type === '') {
        this.startTime = startTime
        this.endTime = endTime
        this.pointChartDataInit()
        this.curveSliceLoading = true
        try {
          await this.setChartData(undefined, request) // 设置曲线数据
        } finally {
          if (this.isCurrentHistoryRequest(request)) this.curveSliceLoading = false
        }
      } else if (type === '放电切片快速选择') {
        this.dischargeStartTime = startTime
        this.dischargeEndTime = endTime
        this.dischargeSliceLoading = true
        try {
          await this.loadBatteryStateAndReferenceHistory(type, request) // 获取电池状态历史数据 和 获取基准数据点历史数据
        } catch (error) {
          if (this.isCurrentHistoryRequest(request)) this.setDischargeSlices({ intervals: [] })
          this.reportHistoryError(error, request)
        } finally {
          if (this.isCurrentHistoryRequest(request)) this.dischargeSliceLoading = false
        }
      } else if (type === '充电切片快速选择') {
        this.chargeStartTime = startTime
        this.chargeEndTime = endTime
        this.chargeSliceLoading = true
        try {
          await this.loadBatteryStateAndReferenceHistory(type, request) // 获取电池状态历史数据 和 获取基准数据点历史数据
        } catch (error) {
          if (this.isCurrentHistoryRequest(request)) this.setChargeSlices({ intervals: [] })
          this.reportHistoryError(error, request)
        } finally {
          if (this.isCurrentHistoryRequest(request)) this.chargeSliceLoading = false
        }
      }
    },
    setSelectTime(selectTimeArr) {
      const endDate = formatDateString(selectTimeArr[1], 'formatDate')
      if (endDate.slice(11) === '00:00:00') {
        selectTimeArr[1] = endDate.slice(0, 10) + ' 23:59:59'
      }
      return selectTimeArr
    },
    isDischargeDateDisabled(date) {
      return this.isDateBeyondOneMonth(date, this.dischargePickingStart || (this.dischargeTimeRange && this.dischargeTimeRange[0]))
    },
    isChargeDateDisabled(date) {
      return this.isDateBeyondOneMonth(date, this.chargePickingStart || (this.chargeTimeRange && this.chargeTimeRange[0]))
    },
    handleDischargeCalendarChange(range) {
      this.dischargePickingStart = range && range[0] ? range[0] : null
    },
    handleChargeCalendarChange(range) {
      this.chargePickingStart = range && range[0] ? range[0] : null
    },
    curveDisabledDate(date) {
      return this.isDateBeyondOneMonth(date, this.curvePickingStart || (this.timeArr && this.timeArr[0]))
    },
    curveCalendarChange(range) {
      this.curvePickingStart = range && range[0] ? range[0] : null
    },
    isDateBeyondOneMonth(date, startValue) {
      if (!startValue) return false
      const maxEnd = this.getOneMonthEnd(startValue)
      return Boolean(maxEnd && date.getTime() > maxEnd.getTime())
    },
    getOneMonthEnd(startValue) {
      const startDate = new Date(startValue)
      if (Number.isNaN(startDate.getTime())) return null
      const targetMonth = startDate.getMonth() + 1
      const lastDay = new Date(startDate.getFullYear(), targetMonth + 1, 0).getDate()
      return new Date(
        startDate.getFullYear(),
        targetMonth,
        Math.min(startDate.getDate(), lastDay),
        startDate.getHours(),
        startDate.getMinutes(),
        startDate.getSeconds(),
        startDate.getMilliseconds()
      )
    },
    limitRangeToOneMonth(range) {
      if (!range || !range[0] || !range[1]) return range
      const maxEnd = this.getOneMonthEnd(range[0])
      const endDate = new Date(range[1])
      if (maxEnd && endDate.getTime() > maxEnd.getTime()) {
        range[1] = getTime(maxEnd.getTime())
      }
      return range
    },
    async loadBatteryStateAndReferenceHistory(type, request) { // 获取电池状态历史数据 和 获取基准数据点历史数据
      let startTime = this.startTime
      let endTime = this.endTime
      // 快速切片仍按原版一秒识别状态，曲线与横轴使用相同的实际查询间隔。
      let timeLength = type ? 1 : this.getHistoryIntervalSeconds()
      switch (type) {
        case '放电切片快速选择':
          startTime = this.dischargeStartTime
          endTime = this.dischargeEndTime
          timeLength = 1
          break
        case '充电切片快速选择':
          startTime = this.chargeStartTime
          endTime = this.chargeEndTime
          timeLength = 1
          break
      }

      const batteryStateResponse = await this.getBatteryStateHisMsg({
        startTime, endTime, timeLength
      })
      if (!this.isCurrentHistoryRequest(request)) return
      const batteryStateTimes = batteryStateResponse.batteryStateTimes
      const batteryStateValues = batteryStateResponse.batteryStateValues
      const stateIntervals = this.buildBatteryStateIntervals(batteryStateTimes, batteryStateValues, {
        currentStateCode: this.currentBatteryStateCode,
        currentStateTimeStmap: this.currentBatteryStateTimeStmap,
        rangeStartTimeStmap: startTime,
        rangeEndTimeStmap: endTime
      })
      this.batteryStatePoint.times = batteryStateResponse.timeStmapArr
      this.batteryStatePoint.values = batteryStateValues

      if (type === '放电切片快速选择') {
        this.setDischargeSlices({ intervals: stateIntervals })
        return
      }
      if (type === '充电切片快速选择') {
        this.setChargeSlices({ intervals: stateIntervals })
        return
      }
      this.chartBatteryStateIntervals = stateIntervals

      const xAxisData = await this.getChartXAxisData({
        startTime,
        endTime,
        timeLength
      })
      if (this.isCurrentHistoryRequest(request)) this.pointChartData.xAxis.data = xAxisData
    },
    async getChartXAxisData(params) {
      const { startTime, endTime, timeLength } = params
      let referencePoint = {}
      this.batteryParameterOptions.forEach(item => {
        if (item.name === '组电压' && item.labelDetails) {
          referencePoint = item.labelDetails
        }
      })
      // 个别机型没有组电压标签，仍可用已选中的真实测点作为时间轴基准。
      if (!referencePoint.labelId) {
        const fallback = [...this.batteryParameterOptions, ...this.cellVoltageOptions]
          .find(item => item.checked && item.labelDetails?.labelId)
        referencePoint = fallback?.labelDetails || {}
      }
      if (!referencePoint.labelId) {
        return []
      }

      const referenceHistory = await this.queryHistory({
        Apikey: this.formData.apiKey,
        Function: 'historydatapasstime',
        Pagenum: 1,
        Pagesize: HISTORY_QUERY_PAGE_SIZE,
        Datestart: startTime,
        Dateend: endTime,
        labelID: referencePoint.labelId,
        timeType: 'second',
        timeLength
      })
      const xAxisData = []
      if (referenceHistory && referenceHistory.times) {
        const referenceTimes = referenceHistory.times.slice().reverse()
        referenceTimes.forEach(timeItem => {
          xAxisData.push({
            value: format0800CSTDateString(timeItem),
            timeStmap: dateConvStamp(format0800CSTDateString(timeItem)) / 1000
          })
        })
      }

      return xAxisData
    },
    async getBatteryStateHisMsg(params) {
      if (this.batteryStatePoint.labelId) {
        const stateMsg = await this.loadBatteryStateHistory(params)
        if (stateMsg.batteryStateTimes.length) {
          return stateMsg
        }
      }
      return this.getIndependentBatteryStateHisMsg(params)
    },
    async getIndependentBatteryStateHisMsg(params) {
      const { startTime, endTime, timeLength = this.getHistoryIntervalSeconds() } = params
      const statePoints = this.batteryStatePoints.filter(item => item.labelDetails && item.labelDetails.labelId)
      if (!statePoints.length) {
        return {
          batteryStateTimes: [],
          batteryStateValues: [],
          timeStmapArr: [],
          batteryStateXAxisData: []
        }
      }

      const stateHistories = await Promise.all(statePoints.map(async statePoint => {
        const data = await this.queryHistory({
          Apikey: this.formData.apiKey,
          Function: 'historydatapasstime',
          Pagenum: 1,
          Pagesize: HISTORY_QUERY_PAGE_SIZE,
          Datestart: startTime,
          Dateend: endTime,
          labelID: statePoint.labelDetails.labelId,
          timeType: 'second',
          timeLength
        })
        const times = Array.isArray(data.times) ? data.times : []
        const values = Array.isArray(data.values) ? data.values : []
        const samples = []
        times.forEach((timeItem, index) => {
          const time = format0800CSTDateString(timeItem)
          const timeStmap = dateConvStamp(time) / 1000
          if (Number.isFinite(timeStmap)) {
            samples.push({
              time,
              timeStmap,
              active: this.isPointActive(values[index])
            })
          }
        })
        samples.sort((a, b) => a.timeStmap - b.timeStmap)
        return {
          pointKey: String(statePoint.labelDetails.labelId),
          code: statePoint.code,
          samples
        }
      }))

      const updatesByTime = new Map()
      stateHistories.forEach(stateHistory => {
        stateHistory.samples.forEach(sample => {
          if (!updatesByTime.has(sample.timeStmap)) {
            updatesByTime.set(sample.timeStmap, {
              time: sample.time,
              states: []
            })
          }
          updatesByTime.get(sample.timeStmap).states.push({
            pointKey: stateHistory.pointKey,
            code: stateHistory.code,
            active: sample.active
          })
        })
      })

      const activeStates = new Map()
      const stateDefs = this.getBatteryStateDefinitions()
      const timeStmapArr = Array.from(updatesByTime.keys()).sort((a, b) => a - b)
      const batteryStateTimes = []
      const batteryStateValues = []
      const batteryStateXAxisData = []
      timeStmapArr.forEach(timeStmap => {
        const timeMsg = updatesByTime.get(timeStmap)
        timeMsg.states.forEach(state => {
          activeStates.set(state.pointKey, state)
        })
        const currentState = stateDefs.find(stateDef => {
          return Array.from(activeStates.values()).some(state => state.code === stateDef.code && state.active)
        })
        batteryStateTimes.push(timeMsg.time)
        batteryStateValues.push(currentState ? currentState.code : null)
        batteryStateXAxisData.push({
          value: timeMsg.time,
          timeStmap
        })
      })

      return {
        batteryStateTimes,
        batteryStateValues,
        timeStmapArr,
        batteryStateXAxisData
      }
    },
    async loadBatteryStateHistory(params) { // 获取电池状态历史数据
      const { startTime, endTime, timeLength = this.getHistoryIntervalSeconds() } = params
      const data = await this.queryHistory({
        Apikey: this.formData.apiKey,
        Function: 'historydatapasstime',
        Pagenum: 1,
        Pagesize: HISTORY_QUERY_PAGE_SIZE,
        Datestart: startTime,
        Dateend: endTime,
        labelID: this.batteryStatePoint.labelId,
        timeType: 'second',
        timeLength
      })
      const timeStmapArr = []
      const batteryStateXAxisData = []
      let times = []
      let values = []
      if (data) {
        if (data.times) {
          times = data.times.slice().reverse() // 颠倒数组顺序
          values = (data.values || []).slice().reverse()
          times.forEach(timeItem => {
            timeStmapArr.push(dateConvStamp(format0800CSTDateString(timeItem)) / 1000)
            batteryStateXAxisData.push({
              value: format0800CSTDateString(timeItem),
              timeStmap: dateConvStamp(format0800CSTDateString(timeItem)) / 1000
            })
          })
        }
      }
      return {
        batteryStateTimes: times,
        batteryStateValues: values,
        timeStmapArr,
        batteryStateXAxisData
      }
    },
    setDischargeSlices(params) { // 设置放电切片
      const allDischargeRecords = params.intervals.filter(item => item.code === this.dischargeStateCode).map(item => ({
        startDate: item.startDate,
        endDate: item.endDate,
        dateArr: item.isOngoing ? item.startDate + ' ~ 进行中' : item.startDate + ' ~ ' + item.endDate,
        isOngoing: Boolean(item.isOngoing),
        selected: false
      }))
      this.allDischargeRecords = allDischargeRecords
      this.total1 = allDischargeRecords.length
      this.updateChargeDischargeRecords()
    },
    setChargeSlices(params) { // 设置充电切片
      const allChargeRecords = params.intervals.filter(item => item.code === this.chargeStateCode).map(item => ({
        startDate: item.startDate,
        endDate: item.endDate,
        dateArr: item.isOngoing ? item.startDate + ' ~ 进行中' : item.startDate + ' ~ ' + item.endDate,
        isOngoing: Boolean(item.isOngoing),
        selected: false
      }))
      this.allChargeRecords = allChargeRecords
      this.total2 = allChargeRecords.length
      this.updateChargeDischargeRecords()
    },
    updateChargeDischargeRecords() { // 设置充放电展示的数据
      const dischargeRecords = this.allDischargeRecords || []
      const chargeRecords = this.allChargeRecords || []
      this.dischargeRecords = dischargeRecords.slice((this.page1 - 1) * this.limit1, this.page1 * this.limit1)
      this.chargeRecords = chargeRecords.slice((this.page2 - 1) * this.limit2, this.page2 * this.limit2)
    }
  }
}
</script>

<style lang="stylus" scoped>
boxShadow = 0 0 3px 2px rgba(0,109,255, 0.4)
.hisDataCss {
  position relative
  width 100%
  min-width 0
  // 按详情正文实际宽度适配，弹窗和浏览器缩放都使用相同的布局规则。
  container-type inline-size
  container-name battery-history
  .historyNotesCss {
    position absolute
    top 5px
    right 10px
  }
  .titleCss {
    padding 0.2vw 0 0 1vw
    min-height 1.5vw
    line-height 1.5
    font-size clamp(16px, 1.2vw, 24px)
    font-weight bold
  }
  .parameterSelectionContentCss {
    margin 0.5vw 0.2%
    width 99.6%
    min-height 8.8vw
    height auto
    padding-bottom 0.5vw
    box-sizing border-box
    box-shadow boxShadow
    border-radius 0.2vw
    .parameterSelectionCss {
      position relative
      display flex
      align-items flex-start
      margin 0.5vw 0 0 5%
      padding 0 1%
      width 90%
      min-height 3vw
      height auto
      font-size clamp(14px, 1vw, 20px)
      box-sizing border-box
      border-radius 0.2vw
      border 1px solid rgba(25, 186, 139, 0.17)
      &::before {
        position absolute
        top 0
        left 0
        content ''
        width 10px
        height 10px
        border-top 2px solid #02a6b5
        border-left 2px solid #02a6b5
        border-radius 20%
      }
      &::after {
        position absolute
        top 0
        right 0
        content ''
        width 10px
        height 10px
        border-top 2px solid #02a6b5
        border-right 2px solid #02a6b5
        border-radius 20%
      }
      .parameterSelectionTitleCss {
        flex 0 0 unquote('max(10%, 110px)')
        padding 6px clamp(8px, 1vw, 20px) 0 0
        box-sizing border-box
        min-height 3vw
        line-height 1.6
        font-weight bold
        text-align right
      }
      .parameterSelectionOptionsCss {
        flex 1
        min-width 0
        min-height 3vw
        height auto
        padding 6px 0
      }
      .parameterSelectionFooterCss {
        position absolute
        left 0
        bottom 0
        width 100%
        &::before {
          position absolute
          bottom 0
          left 0
          content ''
          width 10px
          height 10px
          border-bottom 2px solid #02a6b5
          border-left 2px solid #02a6b5
          border-radius 20%
        }
        &::after {
          position absolute
          bottom 0
          right 0
          content ''
          width 10px
          height 10px
          border-bottom 2px solid #02a6b5
          border-right 2px solid #02a6b5
          border-radius 20%
        }
      }
    }
  }
  .sliceSelectionContentCss {
    margin 0.5vw 0.2%
    width 99.6%
    // 切片记录换行或分页增加高度时，外框随内容增长，避免压住变化曲线。
    display grid
    grid-template-columns repeat(2, minmax(0, 1fr))
    gap unquote('max(8px, 0.5vw)') 1%
    padding 0 1% unquote('max(8px, 0.5vw)')
    box-sizing border-box
    min-height 20vw
    box-shadow boxShadow
    border-radius 0.2vw
    > .titleCss {
      grid-column 1 / -1
      padding-left 0
    }
    .sliceSelectionCss {
      position relative
      min-width 0
      padding unquote('max(8px, 0.5vw)')
      box-sizing border-box
      container-type inline-size
      container-name battery-history-slice
      border-radius 0.2vw
      border 1px solid rgba(25, 186, 139, 0.17)
      &::before {
        position absolute
        top 0
        left 0
        content ''
        width 10px
        height 10px
        border-top 2px solid #02a6b5
        border-left 2px solid #02a6b5
        border-radius 20%
      }
      &::after {
        position absolute
        top 0
        right 0
        content ''
        width 10px
        height 10px
        border-top 2px solid #02a6b5
        border-right 2px solid #02a6b5
        border-radius 20%
      }
      .quickSelectionTitleCss {
        width 100%
        font-size clamp(14px, 1vw, 20px)
        line-height 1.6
        font-weight bold
      }
      .sliceHistoryContentCss {
        width 100%
        display flex
        align-items stretch
        .sliceHistoryCss {
          flex 0 0 44%
          min-width 0
          margin-right 1%
          min-height 14vw
          border-right 1px solid #ccc
          .sliceHistoryTitleCss {
            margin-top 0.5vw
            width 100%
            min-height unquote('max(24px, 1.5vw)')
            line-height 1.6
            font-size clamp(12px, 0.8vw, 16px)
            font-weight bold
          }
          .sliceHistoryDateContentCss {
            width 100%
            min-height 40px
            .sliceHistoryDateCss {
              width 99%
              min-width 0
            }
          }
          .dischargeCountCss {
            width 100%
            line-height 1.6
            font-size clamp(12px, 0.7vw, 16px)
            .dischargeCountValueCss {
              padding 0 0.2vw
            }
          }
        }
        .dischargeRecordsContentCss {
          margin-top -0.5vw
          flex 1
          min-width 0

          .dischargeRecordsTitleCss {
            width 100%
            min-height unquote('max(24px, 1.2vw)')
            line-height 1.6
            font-size clamp(12px, 0.8vw, 16px)
            font-weight bold
          }
          .dischargeRecordsCss {
            width 100%
            .dischargeRecordsTabCss {
              width 100%
              min-width 0
              background-color #fff
              .dischargeRecordsListCss {
                padding-left 0.5vw
                width 100%
                box-sizing border-box
                .dischargeRecordItemCss {
                  width 100%
                  // 保留完整切片起止时间；长时间文字换行时同步撑高记录行。
                  min-height unquote('max(24px, 1.2vw)')
                  padding 3px 4px
                  box-sizing border-box
                  line-height 1.5
                  overflow-wrap anywhere
                  text-align center
                  font-size clamp(12px, 0.7vw, 16px)
                  border-bottom 1px solid #EBEEF5
                }
                .dischargeRecordSelectedCss {
                  background-color rgba(0,109,255, 0.4)
                }
              }
              .emptyDataCss {
                padding-top 0.5vw
                line-height 1.6
                text-align center
              }
              .pagCss {
                margin-top 0.5vw
                max-width 100%
                overflow-x auto
              }
            }
          }
        }
      }
      .sliceSelectionFooterCss {
        position absolute
        left 0
        bottom 0
        width 100%
        &::before {
          position absolute
          bottom 0
          left 0
          content ''
          width 10px
          height 10px
          border-bottom 2px solid #02a6b5
          border-left 2px solid #02a6b5
          border-radius 20%
        }
        &::after {
          position absolute
          bottom 0
          right 0
          content ''
          width 10px
          height 10px
          border-bottom 2px solid #02a6b5
          border-right 2px solid #02a6b5
          border-radius 20%
        }
      }
    }
  }
  .trendCurveContentCss {
    margin 0.5vw 0.2%
    width 99.6%
    min-height 14.5vw
    padding-bottom unquote('max(8px, 0.5vw)')
    box-sizing border-box
    box-shadow boxShadow
    border-radius 0.2vw
    .trendCurveHeaderCss {
      width 100%
      // 筛选项按可用宽度换行，每一行的高度由日期框和按钮决定。
      display flex
      flex-wrap wrap
      align-items center
      gap 6px clamp(8px, 0.5vw, 16px)
      padding 6px 1%
      box-sizing border-box
      min-height 2vw
      .trendCurveTitleCss {
        flex 0 0 auto
        line-height 1.5
        font-size clamp(16px, 1.2vw, 24px)
        font-weight bold
      }
      .trendCurveLegendCss {
        display flex
        align-items center
        flex 0 0 auto
        margin 0 clamp(4px, 1vw, 20px)
        .statusIconCss {
          flex-shrink 0
          margin-right clamp(4px, 0.3vw, 8px)
          width clamp(10px, 1vw, 20px)
          height clamp(10px, 1vw, 20px)
          border-radius 50%
        }
        .statusIcon1Css {
          background-color #339900
        }
        .statusIcon2Css {
          background-color #ff0000
        }
        .statusIcon3Css {
          background-color #ffff00
        }
        .statusNameCss {
          margin-right clamp(8px, 1vw, 20px)
          line-height 1.6
          font-size clamp(12px, 0.8vw, 16px)
        }
      }
      .queryIntervalDateCss {
        flex 0 1 340px
        min-width 0
        max-width 100%
      }
      .queryIntervalCss {
        display flex
        flex-wrap wrap
        align-items center
        gap 4px
        min-width 0
        .queryIntervalTitleCss {
          line-height 1.6
          font-size clamp(12px, 0.8vw, 16px)
        }
        .inputContentCss {
          width 80px
          min-width 0
        }
      }
      .exportCss {
        flex 0 0 auto
        .importCss {
          color red
          line-height 32px
        }
      }
    }
    .multipleYAxesContentCss {
      position relative
      margin 0.2vw 0 0 1%
      padding-bottom 0.5vw
      width 98%
      height 11.5vw
      // 为图例、坐标轴和缩放条预留可读空间，低分辨率下通过正文滚动查看。
      min-height 260px
      box-sizing border-box
      border-radius 0.2vw
      border 1px solid rgba(25, 186, 139, 0.17)
      &::before {
        position absolute
        top 0
        left 0
        content ''
        width 10px
        height 10px
        border-top 2px solid #02a6b5
        border-left 2px solid #02a6b5
        border-radius 20%
      }
      &::after {
        position absolute
        top 0
        right 0
        content ''
        width 10px
        height 10px
        border-top 2px solid #02a6b5
        border-right 2px solid #02a6b5
        border-radius 20%
      }
      .smultipleYAxesFooterCss {
        position absolute
        left 0
        bottom 0
        width 100%
        &::before {
          position absolute
          bottom 0
          left 0
          content ''
          width 10px
          height 10px
          border-bottom 2px solid #02a6b5
          border-left 2px solid #02a6b5
          border-radius 20%
        }
        &::after {
          position absolute
          bottom 0
          right 0
          content ''
          width 10px
          height 10px
          border-bottom 2px solid #02a6b5
          border-right 2px solid #02a6b5
          border-radius 20%
        }
      }
    }
  }
  :deep(.el-date-editor) {
    width 100%
    max-width 100%
    min-width 0
    box-sizing border-box
  }
  // 参数复选框原来随 vw 缩到不足 10px，仅在本页设置可读字号及点击区域。
  :deep(.customCheckboxCss) {
    font-size clamp(12px, 0.7vw, 16px)
    line-height 1.4
    min-height clamp(14px, 1vw, 20px)
    padding-left clamp(14px, 1vw, 20px)
    margin-bottom unquote('max(6px, 0.5vw)')
  }
  :deep(.checkmarkCss) {
    width clamp(14px, 1vw, 20px)
    height clamp(14px, 1vw, 20px)
  }
  :deep(.customCheckboxCss .checkmarkCss:after) {
    left 36%
    top 10%
    width 26%
    height 52%
    border-width 0 unquote('max(1px, 0.1vw)') unquote('max(1px, 0.1vw)') 0
  }
  :deep(.customCheckboxCss input:indeterminate ~ .checkmarkCss:before) {
    top 41%
    left 15%
    width 62%
    height unquote('max(1px, 0.1vw)')
  }
}

// 先让单张切片卡片内部纵向排列，再在详情更窄时将充、放电卡片上下排列。
@container battery-history-slice (max-width: 630px) {
  .hisDataCss .sliceSelectionContentCss .sliceSelectionCss .sliceHistoryContentCss {
    flex-direction column
    gap 8px
    .sliceHistoryCss {
      flex-basis auto
      min-height 0
      margin-right 0
      padding-bottom 8px
      border-right 0
      border-bottom 1px solid #ccc
    }
    .dischargeRecordsContentCss {
      margin-top 0
    }
  }
}
@container battery-history (max-width: 980px) {
  .hisDataCss .sliceSelectionContentCss {
    grid-template-columns minmax(0, 1fr)
  }
}
@container battery-history (max-width: 460px) {
  .hisDataCss .parameterSelectionContentCss .parameterSelectionCss {
    flex-direction column
    .parameterSelectionTitleCss {
      flex-basis auto
      min-height 0
      padding-right 0
      text-align left
    }
    .parameterSelectionOptionsCss {
      width 100%
    }
  }
}

// 不支持容器查询的旧浏览器使用窗口断点，仍允许内容撑高和工具栏换行。
@supports not (container-type: inline-size) {
  @media (max-width: 1440px) {
    .hisDataCss .sliceSelectionContentCss .sliceSelectionCss .sliceHistoryContentCss {
      flex-direction column
      gap 8px
      .sliceHistoryCss {
        flex-basis auto
        min-height 0
        margin-right 0
        padding-bottom 8px
        border-right 0
        border-bottom 1px solid #ccc
      }
      .dischargeRecordsContentCss {
        margin-top 0
      }
    }
  }
  @media (max-width: 1100px) {
    .hisDataCss .sliceSelectionContentCss {
      grid-template-columns minmax(0, 1fr)
    }
  }
  @media (max-width: 600px) {
    .hisDataCss .parameterSelectionContentCss .parameterSelectionCss {
      flex-direction column
      .parameterSelectionTitleCss {
        flex-basis auto
        min-height 0
        padding-right 0
        text-align left
      }
      .parameterSelectionOptionsCss {
        width 100%
      }
    }
  }
}
</style>

<style lang="stylus">
.sliceHistoryDateCss {
  .el-date-editor {
    width 100%
  }
}

</style>
