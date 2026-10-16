<template>
  <div
    ref="screenWrapperRef"
    class="screen-wrapper"
    :class="{
      'screen-wrapper--scroll': screenNeedsScroll,
      'screen-wrapper--dialog-open': showDialog
    }"
  >
    <div class="screen-scale-box" :style="scaleBoxStyle">
      <div ref="dashboardRef" class="jscCss" :style="screenStyle">
        <div class="bigPingCss">
          <div class="bingPingBackgroundCss">
            <img src="./imgs/大屏背景1.png">
          </div>
          <div class="topBannerCss">
            <img src="./imgs/顶部背景.png">
            <div class="headerCss">
              <div class="titleCss">锂硫电池组溯源平台</div>
            </div>
          </div>
          <div class="msgBorderContentOneCss">
            <div class="imgBackgroundCss"><img src="./imgs/电池统计.png"></div>
            <div class="msgBorderTitleCss msgBorderTitleDctjCss" @click="goDetailPage('电池统计')">电池统计</div>
            <div class="gfxtContentCss">
              <div v-for="(item, index) in lifecycleSummaryItems" :key="index" class="gfxtDetailCss">
                <div class="gfxtImgCss">
                  <img :src="item.imgUrl">
                </div>
                <div class="gfxtDetailTitleAndMsgCss">
                  <div class="gfxtDetailTitleMsgCss">{{ item.title }}</div>
                  <div class="gfxtDetailMsgCss">{{ item.msg }}</div>
                </div>
              </div>
            </div>
          </div>
          <div class="msgBorderLeftImgCss">
            <img src="./imgs/三角形左.png">
          </div>
          <div class="msgBorderContentFourCss">
            <div class="imgBackgroundCss"><img src="./imgs/告警统计.png"></div>
            <div class="msgBorderTitleCss msgBorderTitleGjtjCss" @click="goDetailPage('告警统计')">告警统计</div>
            <div class="msgBorderContentFourAndFiveCss">
              <DpLineChart
                id="dpLineChart1Id"
                width="98%"
                height="100%"
                :chart-data="dpLineChart1ChartData"
              />
            </div>
          </div>
          <div class="msgBorderLeftBotoomImgCss">
            <img src="./imgs/三角形右.png">
          </div>
          <div class="msgBorderContentFiveCss">
            <div class="imgBackgroundCss"><img src="./imgs/预警统计.png"></div>
            <div class="msgBorderRightTitleCss msgBorderTitleSsgjCss" @click="goDetailPage('实时告警')">
              实时告警（<span class="realtimeAlarmCountCss">{{ alarmCount }}</span>条）
            </div>
            <div class="tableCss">
              <dv-scroll-board
                ref="realtimeAlarmBoardRef"
                :config="realtimeAlarmBoardConfig"
                style="width:100%;height:100%"
              />
            </div>
          </div>

          <div class="msgBorderContentTwoCss msgBorderContentRightCss">
            <div class="imgBackgroundCss"><img src="./imgs/实时告警.png"></div>
            <div class="msgBorderRightTitleCss msgBorderTitleDcjkdtjCss" @click="goDetailPage('电池健康度统计')">电池健康度统计</div>
            <div class="msgBorderContentSevenContentCss">
              <div class="BasicPieCss">
                <BasicPie
                  v-if="healthStatisticsStatus === 'ready'"
                  id="dcjkdtj"
                  width="100%"
                  height="100%"
                  :chart-data="batteryHealthChartData"
                />
                <div v-else class="healthStatisticsPlaceholderCss">
                  {{ healthStatisticsStatus === 'loading' ? '正在加载数据' : '暂无数据' }}
                </div>
              </div>
            </div>
          </div>
          <div class="msgBorderRightImgCss">
            <img src="./imgs/三角形右.png">
          </div>
          <div class="msgBorderContentSixCss msgBorderContentRightCss">
            <div class="imgBackgroundCss"><img src="./imgs/实时预警.png"></div>
            <div class="msgBorderTitleCss msgBorderTitleYjtjCss" @click="goDetailPage('预警统计')">预警统计</div>
            <div class="msgBorderContentFourAndFiveCss">
              <DpLineChart
                id="dpLineChart2Id"
                width="98%"
                height="100%"
                :chart-data="dpLineChart2ChartData"
              />
            </div>
          </div>
          <div class="msgBorderRightBottomImgCss">
            <img src="./imgs/三角形左.png">
          </div>
          <div class="msgBorderContentSevenCss msgBorderContentRightCss">
            <div class="imgBackgroundCss"><img src="./imgs/电池健康度统计.png"></div>
            <div class="msgBorderRightTitleCss msgBorderTitleSsyjCss" @click="goDetailPage('实时预警')">
              实时预警（<span class="realtimeWarningCountCss">{{ warningCount }}</span>条）
            </div>
            <div class="msgBorderContentSixChartContentCss">
              <div class="tableCss">
                <dv-scroll-board
                  ref="realtimeWarningBoardRef"
                  :config="realtimeWarningBoardConfig"
                  style="width:100%;height:100%"
                />
              </div>
            </div>
          </div>

          <div class="mapBannerCss">
            <div class="pngAndMsgCss">
              <div class="pngAndMsgDataCss">
                <div class="msgTitleCss">站址数量</div>
                <div class="msgCss">
                  <div class="imgBackgroundCss"><img src="./imgs/顶部面板框2.png"></div>
                  <div class="msgNumCss">{{ siteCount }}</div>
                </div>
              </div>
            </div>
            <div class="pngAndMsgCss">
              <div class="pngAndMsgDataCss">
                <div class="msgTitleCss">入网电池数量</div>
                <div class="msgCss">
                  <div class="imgBackgroundCss"><img src="./imgs/顶部面板框2.png"></div>
                  <div class="msgNumCss">{{ onlineBatteryCount }}</div>
                </div>
              </div>
            </div>
            <div class="pngAndMsgCss">
              <div class="pngAndMsgDataCss">
                <div class="msgTitleCss">告警数量</div>
                <div class="msgCss">
                  <div class="imgBackgroundCss"><img src="./imgs/顶部面板框1.png"></div>
                  <div class="msgNumCss">{{ alarmCount }}</div>
                </div>
              </div>
            </div>
            <div class="pngAndMsgCss">
              <div class="pngAndMsgDataCss">
                <div class="msgTitleCss">预警数量</div>
                <div class="msgCss">
                  <div class="imgBackgroundCss"><img src="./imgs/顶部面板框1.png"></div>
                  <div class="msgNumCss">{{ warningCount }}</div>
                </div>
              </div>
            </div>
          </div>

          <div class="mapContentCss">
            <div class="amap-page-container">
              <MapComplex
                id="map1"
                width="100%"
                height="100%"
                :area-addr-msg-list="areaAddrMsgList"
                :corp-tree-list="treeList"
                :queryable-corp-keys="queryableCorpKeys"
                @level-change="levelChange"
                @go-to-detail="goToDetail"
              />
            </div>
          </div>

          <div class="mapBackCss">
            <div class="cicle3" />
            <div class="cicle4" />
            <div class="cicle5" />
            <div class="cicle6" />
            <div class="cicle7" />
          </div>
          <div ref="rainContentRef" class="rainContentCss">
            <canvas ref="rainCanvasRef" class="rainCss" />
          </div>

          <div class="msgBorderContentThreeCss">
            <div class="imgBackgroundCss"><img src="./imgs/充放电统计.png"></div>
            <div class="msgBorderThreeTitleContentCss">
              <div class="msgBorderThreeTitleContentChildCss">
                <div class="msgBorderThreeTitleLeftImgCss">
                  <div class="imgBackgroundCss"><img src="./imgs/充放电统计装饰.png"></div>
                </div>
                <div class="msgBorderThreeTitleCss" @click="goDetailPage('电池健康度统计')">充放电统计</div>
                <div class="msgBorderThreeTitleRightImgCss">
                  <div class="imgBackgroundCss"><img src="./imgs/充放电统计装饰.png"></div>
                </div>
              </div>
            </div>
            <div v-if="showChart" class="msgBorderChartCss">
              <div class="multipleYAxesCss">
                <MultipleYAxes
                  id="EchartsBasicBar1"
                  width="100%"
                  height="288px"
                  :chart-data="basicBarData1"
                />
              </div>
            </div>
          </div>
        </div>

        <div class="jkztjcCss">
          <!-- 中间弹出 -->
          <el-dialog
            v-if="dialogTeleportReady"
            v-model="showDialog"
            append-to=".screen-wrapper"
            class="jscDcDetailDialog"
            modal-class="jscDcDetailOverlay"
            title="电池详情"
            width="90%"
            destroy-on-close
          >
            <BatteryDetail
              v-if="showDialog"
              :form-msg="equipDetailMsg"
              @click-close="showDialog = false"
            />
          </el-dialog>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { defineAsyncComponent, markRaw } from 'vue'
import { Message } from '@/utils/element-shim'
import lodash from 'lodash'
import BasicPie from '@/components/a_myComp/echarts/BasicPie.vue'
import MultipleYAxes from '@/components/a_myComp/echarts/MultipleYAxes.vue'
import MapComplex from '@/components/a_myComp/mapComplex/MapComplex.vue'
import { findBestCorpByRegionName, resolveSiteRegionName } from '@/components/a_myComp/mapComplex/index.js'
import DpLineChart from '@/components/a_myComp/echarts/DpLineChart.vue'
import { findSiteListExactMatch } from '@/api/siteMana.js'
import { findEquipmentListExactMatch, getEquipAllPoint } from '@/api/equipMana.js'
import { getapilistdata } from '@/api/moniList.js'
import { corpList } from '@/api/user.js'
import { getCountbycorp, getDailyAlarmStatistics, getDailyWarningStatistics, getChargeDischargeDailyStatistics } from '@/api/dashboard.js'
import { findAlarmRecord } from '@/api/trigger.js'
import healthImage from './imgs/健康度图标.png'
import coldHeatImage from './imgs/冷热总量.png'
import storageDischargeImage from './imgs/储能放电量.png'
import dailyHeatImage from './imgs/日热量.png'
import heatEfficiencyImage from './imgs/热效率.png'
import dailyColdImage from './imgs/日冷量.png'
import coldEfficiencyImage from './imgs/冷效率.png'
import flowOneImage from './imgs/表1流量.png'
import flowTwoImage from './imgs/表2流量.png'
import dataDecorationImage from './imgs/储能系统数据装饰.png'
import inboundImage from './imgs/入库数量.png'
import outboundImage from './imgs/出库数量.png'
import stationEntryImage from './imgs/上站入网.png'
import stationExitImage from './imgs/退网下站.png'
import pendingInspectionImage from './imgs/检测待用.png'
import discardedImage from './imgs/检测淘汰.png'
import mapAlarmImage from './map.png'
import alarmImage from './imgs/告警3.png'
import warningImage from './imgs/预警3.png'
import { getTime, formatDateString, dateConvStamp, getRecentlyDate, setNumerOrStrNum } from '@/utils/utils.js'
import { parseUserInfo } from '@/utils/safe-json.js'
import { buildQueryableCorpKeys } from '@/utils/corp-scope.js'
import { linearGradientColor } from '@/utils/echarts-gradient.js'
import {
  isDashboardApiFullscreen,
  looksLikeBrowserF11Fullscreen,
  onFullscreenChange,
  toggleDashboardFullscreen
} from '@/utils/fullscreen.js'

const DESIGN_WIDTH = 1920
const DESIGN_HEIGHT = 1080
const USER_ZOOM_MIN = 0.5
const USER_ZOOM_MAX = 3
const USER_ZOOM_STEP = 0.05
const DATA_REFRESH_INTERVAL = 5 * 60 * 1000
const SITE_PAGE_SIZE = 10000
const BatteryDetail = defineAsyncComponent(() => import('@/views/batteryTraceability/batteryHealth/BatteryDetail.vue'))

export default {
  // 保留改名前的组件名称，使既有路由缓存继续生效。
  name: 'jsc',
  components: {
    BasicPie,
    MultipleYAxes,
    MapComplex,
    DpLineChart,
    BatteryDetail
  },
  props: {
    carouselIndex: {
      type: Number,
      default: 0
    }
  },
  data() {
    return {
      userInfo: parseUserInfo({}), // 用户信息

      weatherInfo: {}, // 天气信息
      weatherImage: '', // 天气图片

      pngAndMsg1List: [
        { url: healthImage, title: '光伏发电总量', unit: '(kWh)', msg: '暂无' },
        { url: coldHeatImage, title: '冷总量/热总量', unit: '(kw)', msg: '暂无' },
        { url: storageDischargeImage, title: '储能总充/放电量', unit: '(kWh)', msg: '暂无' },
        { url: storageDischargeImage, title: '充电桩总用电量', unit: '(kWh)', msg: '暂无' }
      ],

      alarmSummaryItems: [
        { title: '总压低保护告警', msg: '3', imgUrl: dailyHeatImage },
        { title: '单体低保护告警', msg: '0', imgUrl: heatEfficiencyImage },
        { title: '高温告警', msg: '2', imgUrl: dailyColdImage },
        { title: '低温告警', msg: '1', imgUrl: coldEfficiencyImage },
        { title: '过流告警', msg: '0', imgUrl: flowOneImage },
        { title: '短路告警', msg: '2', imgUrl: flowTwoImage }
      ],
      chartMsg1: {
        show: false,
        name: '暖通实时功率',
        chartId: 'chartId1',
        title: '实时功率（kWh)',
        max: 100,
        series: [{ // 纵坐标数据
          data: [{ value: 0, name: '' }]
        }]
      },

      warningSummaryItems: [
        { title: '总预警数', msg: '0', imgUrl: dataDecorationImage },
        { title: 'SOC预警', msg: '0', imgUrl: dataDecorationImage },
        { title: 'SOH预警', msg: '0', imgUrl: dataDecorationImage }
      ],
      chartMsg2: {
        show: false,
        name: '发电功率',
        chartId: 'chartId2',
        title: '发电功率（kWh)',
        max: 100,
        series: [{ // 纵坐标数据
          data: [{ value: 0, name: '' }]
        }]
      },

      lifecycleSummaryItems: [
        { title: '入库数量', msg: '0', imgUrl: inboundImage },
        { title: '出库数量', msg: '0', imgUrl: outboundImage },
        { title: '上站入网', msg: '0', imgUrl: stationEntryImage },
        { title: '退网下站', msg: '0', imgUrl: stationExitImage },
        { title: '检测待用', msg: '0', imgUrl: pendingInspectionImage },
        { title: '检测淘汰', msg: '0', imgUrl: discardedImage }
      ],

      indicatorPoolApiKey: '66592e3b99a25818b505e748', // 指标关联数据池
      indicatorPoolLabels: [], // 指标关联数据池标签列表

      showChart: false,

      photovoltaicPoolApiKey: '665984e18f754a80ae8fe2ab', // 光伏发电数据池
      photovoltaicPoolLabels: [], // 光伏发电数据池标签列表
      msgBorderChart1: {
        colors: ['#2198FC'],
        legendDataNames: ['逆变器实时功率'],
        xAxisData: [],
        yAxisNames: ['kw'],
        series: [{
          name: '逆变器实时功率',
          type: 'line',
          data: []
        }]
      },
      msgBorderChart1ChangeKey: 0,

      energyStoragePoolApiKey: '665984d48f754a80ae8fe2a8', // 储能监控数据池
      energyStoragePoolLabels: [], // 储能监控数据池标签列表
      msgBorderChart2: {
        colors: ['#F9C37F'],
        legendDataNames: ['电池功率'],
        xAxisData: [],
        yAxisNames: ['kw'],
        series: [{
          name: '电池功率',
          type: 'line',
          data: []
        }]
      },
      msgBorderChart2ChangeKey: 0,

      dpLineChart1ChartData: {
        animation: false,
        legend: {
          data: ['总告警数', '放电过流', '充电高温', '放电高温'] // 曲线标题
        },
        xAxis: { // 横轴
          data: []
        },
        yAxis: [ // 纵轴
          { name: '' }
        ],
        showSeriesLable: 'true', // 是否显示曲线上的文字，为字符串'true'时显示
        series: [
          {
            name: '总告警数',
            type: 'line',
            smooth: true,
            symbol: 'circle',
            symbolSize: 5,
            showSymbol: false,
            lineStyle: {
              normal: {
                color: 'rgba(34,196,255, 1)',
                width: 2
              }
            },
            areaStyle: {
              normal: {
                color: linearGradientColor([{
                  offset: 0,
                  color: 'rgba(34,196,255,.8)'
                }, {
                  offset: 0.8,
                  color: 'rgba(34,196,255,.1)'
                }]),
                shadowColor: 'rgba(0, 0, 0, 0.1)'
              }
            },
            itemStyle: {
              normal: {
                color: 'rgba(34,196,255, 1)',
                borderColor: 'rgba(34,196,255,.1)',
                borderWidth: 12
              }
            },
            data: []
            // data: [300, 100, 300, 400, 300, 400, 300]
          }, {
            name: '放电过流',
            type: 'line',
            smooth: true,
            symbol: 'circle',
            symbolSize: 5,
            showSymbol: false,
            lineStyle: {
              normal: {
                color: 'rgba(255,170,255, 1)',
                width: 2
              }
            },
            areaStyle: {
              normal: {
                color: linearGradientColor([{
                  offset: 0,
                  color: 'rgba(255,170,255,.8)'
                }, {
                  offset: 0.8,
                  color: 'rgba(255,170,255,.1)'
                }]),
                shadowColor: 'rgba(0, 0, 0, 0.1)'
              }
            },
            itemStyle: {
              normal: {
                color: 'rgba(255,170,255, 1)',
                borderColor: 'rgba(255,170,255,.1)',
                borderWidth: 12
              }
            },
            data: []
            // data: [400, 150, 200, 300, 100, 300, 200]
          }, {
            name: '充电高温',
            type: 'line',
            smooth: true,
            symbol: 'circle',
            symbolSize: 5,
            showSymbol: false,
            lineStyle: {
              normal: {
                color: 'rgba(37,239,61, 1)',
                width: 2
              }
            },
            areaStyle: {
              normal: {
                color: linearGradientColor([{
                  offset: 0,
                  color: 'rgba(37,239,61,.8)'
                }, {
                  offset: 0.8,
                  color: 'rgba(37,239,61,.1)'
                }]),
                shadowColor: 'rgba(0, 0, 0, 0.1)'
              }
            },
            itemStyle: {
              normal: {
                color: 'rgba(37,239,61, 1)',
                borderColor: 'rgba(37,239,61,.1)',
                borderWidth: 12
              }
            },
            data: []
            // data: [500, 200, 100, 200, 400, 200, 100]
          }, {
            name: '放电高温',
            type: 'line',
            smooth: true,
            symbol: 'circle',
            symbolSize: 5,
            showSymbol: false,
            lineStyle: {
              normal: {
                color: 'rgba(255,190,92, 1)',
                width: 2
              }
            },
            areaStyle: {
              normal: {
                color: linearGradientColor([{
                  offset: 0,
                  color: 'rgba(255,190,92,.8)'
                }, {
                  offset: 0.8,
                  color: 'rgba(255,190,92,.1)'
                }]),
                shadowColor: 'rgba(0, 0, 0, 0.1)'
              }
            },
            itemStyle: {
              normal: {
                color: 'rgba(255,190,92, 1)',
                borderColor: 'rgba(255,190,92,.1)',
                borderWidth: 12
              }
            },
            data: []
          }
        ]
      },

      dpLineChart2ChartData: {
        animation: false,
        legend: {
          data: ['总预警数', 'SOC预警', 'SOH预警']
        },
        xAxis: {
          data: []
        },
        yAxis: [
          { name: '' }
        ],
        showSeriesLable: 'true',
        series: [
          {
            name: '总预警数',
            type: 'line',
            smooth: true,
            symbol: 'circle',
            symbolSize: 5,
            showSymbol: false,
            lineStyle: {
              normal: {
                color: 'rgba(244,234,42, 1)',
                width: 2
              }
            },
            areaStyle: {
              normal: {
                color: linearGradientColor([{
                  offset: 0,
                  color: 'rgba(244,234,42,.8)'
                }, {
                  offset: 0.8,
                  color: 'rgba(244,234,42,.1)'
                }]),
                shadowColor: 'rgba(0, 0, 0, 0.1)'
              }
            },
            itemStyle: {
              normal: {
                color: 'rgba(244,234,42, 1)',
                borderColor: 'rgba(244,234,42,.1)',
                borderWidth: 12
              }
            },
            data: []
          }, {
            name: 'SOC预警',
            type: 'line',
            smooth: true,
            symbol: 'circle',
            symbolSize: 5,
            showSymbol: false,
            lineStyle: {
              normal: {
                color: 'rgba(34,196,255, 1)',
                width: 2
              }
            },
            areaStyle: {
              normal: {
                color: linearGradientColor([{
                  offset: 0,
                  color: 'rgba(34,196,255,.8)'
                }, {
                  offset: 0.8,
                  color: 'rgba(34,196,255,.1)'
                }]),
                shadowColor: 'rgba(0, 0, 0, 0.1)'
              }
            },
            itemStyle: {
              normal: {
                color: 'rgba(34,196,255, 1)',
                borderColor: 'rgba(34,196,255,.1)',
                borderWidth: 12
              }
            },
            data: []
          }, {
            name: 'SOH预警',
            type: 'line',
            smooth: true,
            symbol: 'circle',
            symbolSize: 5,
            showSymbol: false,
            lineStyle: {
              normal: {
                color: 'rgba(255,170,255, 1)',
                width: 2
              }
            },
            areaStyle: {
              normal: {
                color: linearGradientColor([{
                  offset: 0,
                  color: 'rgba(255,170,255,.8)'
                }, {
                  offset: 0.8,
                  color: 'rgba(255,170,255,.1)'
                }]),
                shadowColor: 'rgba(0, 0, 0, 0.1)'
              }
            },
            itemStyle: {
              normal: {
                color: 'rgba(255,170,255, 1)',
                borderColor: 'rgba(255,170,255,.1)',
                borderWidth: 12
              }
            },
            data: []
          }
        ]
      },

      realtimeAlarmRecords: [], // 实时告警数据列表
      realtimeWarningRecords: [], // 实时预警数据列表

      realtimeAlarmBoardConfig: { // 实时告警
        headerBGC: '#006990', // 表头背景色
        rowNum: 5,
        header: ['告警名称', '电池ID', '触发时间'],
        data: [],
        align: ['center', 'center', 'center'],
        columnWidth: [120, 190, 120],
        waitTime: 3000 // 轮播间隔(ms)
      },

      realtimeWarningBoardConfig: { // 实时预警
        headerBGC: '#006990', // 表头背景色
        rowNum: 5,
        header: ['预警名称', '电池ID', '触发时间'],
        data: [],
        align: ['center', 'center', 'center'],
        columnWidth: [120, 190, 120],
        waitTime: 3000 // 轮播间隔(ms)
      },

      healthStatisticsStatus: 'loading',
      batteryHealthChartData: {
        animation: false,
        tooltip: {
          trigger: 'item',
          formatter: '{b} : {d}%' // {a}（系列名称），{b}（类目值），{c}（数值）； 注意：换行使用 '\n'
        },
        colors: ['#90EE90', '#25D6FF', '#C3E43B', '#ffaaff'],
        series: [{
          type: 'pie',
          radius: ['30%', '70%'],
          center: ['50%', '50%'],
          selectedMode: 'single',
          padAngle: 5,
          itemStyle: {
            borderRadius: 5
          },
          label: {
            show: true, // 显示标签
            overflow: 'none', // 标签溢出不隐藏
            formatter: '{b}'
          },
          data: [
            {
              value: 0,
              name: '健康度优',
              position: 'inner', label: {
                fontSize: 14,
                color: '#90EE90', formatter: [
                  '{b}({c}%)' // {a}（系列名称），{b}（类目值），{c}（数值）； 注意：换行使用 '\n'
                ].join('\n')
              }
            },
            { value: 0, name: '健康度良',
              position: 'inner', label: {
                fontSize: 14,
                color: '#25D6FF', formatter: [
                  '{b}({c}%)' // {a}（系列名称），{b}（类目值），{c}（数值）； 注意：换行使用 '\n'
                ].join('\n')
              }
            },
            { value: 0, name: '健康度中',
              position: 'inner', label: {
                fontSize: 14,
                color: '#C3E43B', formatter: [
                  '{b}({c}%)' // {a}（系列名称），{b}（类目值），{c}（数值）； 注意：换行使用 '\n'
                ].join('\n')
              }
            },
            { value: 0, name: '健康度差',
              position: 'inner', label: {
                fontSize: 14,
                color: '#ffaaff', formatter: [
                  '{b}({c}%)' // {a}（系列名称），{b}（类目值），{c}（数值）； 注意：换行使用 '\n'
                ].join('\n')
              }
            }
          ]
        }]
      },

      basicBarData1: {
        animation: false,
        colors: ['#22c4ff', '#ffaaff'],
        legend: {
          show: true,
          data: ['累计放电量', '累计充电量'] // 曲线标题
        },
        grid: {
          top: 50,
          left: '12%',
          right: '10%',
          bottom: 60,
          textStyle: {
            color: '#fff'
          }
        },
        xAxis: { // 横轴
          data: []
        },
        yAxis: [{ // 纵轴
          type: 'value',
          name: '电量',
          position: 'left',
          alignTicks: false,
          axisLine: {
            show: true,
            onZero: false,
            lineStyle: {
              color: '#fff'
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
              show: false,
              position: 'top',
              color: '#fff'
            },
            yAxisIndex: 0,
            data: []
          },
          {
            name: '累计充电量',
            type: 'bar',
            label: {
              show: false,
              position: 'top',
              color: '#fff'
            },
            yAxisIndex: 0,
            data: []
          }
        ]
      },

      startTime: '', // 开始时间
      endTime: '', // 结束时间
      dataRefreshTimer: null,
      dataRefreshEnabled: false,
      dashboardRefreshRunning: false,
      dashboardRefreshPending: false,
      dashboardRefreshVersion: 0,
      lastSuccessfulRefreshAt: 0,
      visibilityChangeHandler: null,

      siteCount: 0, // 站址数量
      onlineBatteryCount: 0, // 入网电池数量
      alarmCount: 0, // 告警数量
      warningCount: 0, // 预警数量

      treeList: [], // 部门列表
      queryableCorpKeys: [], // 当前用户可查询的部门范围
      siteList: [], // 客户列表
      equipList: [], // 全部设备列表
      equipmentById: markRaw(new Map()),
      equipmentBySiteId: markRaw(new Map()),

      areaAddrMsgList: [], // 区域位置信息列表

      equipDetailMsg: {}, // 设备详情信息
      showDialog: false,
      dialogTeleportReady: false,

      selectCropId: null, // 选择的部门id

      screenStyle: {
        transform: 'scale(1)',
        transformOrigin: 'left top',
        position: 'absolute',
        left: '0px',
        top: '0px'
      },
      scaleBoxStyle: {
        width: `${DESIGN_WIDTH}px`,
        height: `${DESIGN_HEIGHT}px`
      },
      isScreenFullscreen: false,
      userZoom: 1,
      screenNeedsScroll: false,
      zoomCenterRatioX: null,
      screenResizeHandler: null,
      screenScaleFrameId: null,
      screenResizeObserver: null,
      shellResizeObserver: null,
      removeFullscreenListener: null,
      f11KeydownHandler: null,
      wheelZoomHandler: null,
      browserF11TipShown: false,
      lastScreenScale: null,
      screenScaleForceUntil: 0,
      fullscreenScaleTimers: [],

      rainAnimId: null,
      rainResizeHandler: null,
      rainDropTimers: [],
      rainBgTimer: null,
      rainIdleCallbackId: null,
      rainStopped: false
    }
  },
  watch: {
    carouselIndex(val) {
      if (val === 0) {
        this.initAllMsg()
      } else {
        this.stopRainBg()
        this.stopDataRefresh()
      }
    }
  },
  created() {
    this.setxAxisData()
  },
  beforeUnmount() {
    this.stopRainBg()
    this.stopScreenResize()
    this.stopDataRefresh()
  },
  activated() {
    this.lastScreenScale = null
    if (this.carouselIndex === 0) {
      const shouldRefresh = !this.dataRefreshEnabled &&
        Date.now() - this.lastSuccessfulRefreshAt >= DATA_REFRESH_INTERVAL
      this.startDataRefresh()
      if (shouldRefresh) {
        this.refreshDashboardData()
      }
    }
    this.$nextTick(() => {
      this.startScreenResize()
      this.queueScreenScale()
    })
  },
  deactivated() {
    this.stopScreenResize()
    this.stopDataRefresh()
  },
  mounted() {
    this.dialogTeleportReady = true
    if (this.carouselIndex === 0) {
      this.initAllMsg()
    }
    this.showChart = true
    this.startScreenResize()
    this.syncScreenFullscreen()
    this.$nextTick(() => {
      this.queueScreenScale()
    })
  },
  methods: {
    initAllMsg() {
      this.rainStopped = false
      this.startDataRefresh()
      this.init()
      if (this.rainBgTimer) {
        clearTimeout(this.rainBgTimer)
      }
      // 接口异常时也会兜底启动；正常情况下由地图首批数据完成后在空闲时启动。
      this.rainBgTimer = setTimeout(() => {
        this.rainBgTimer = null
        if (this.rainStopped) return
        this.queueRainBg()
      }, 5000)
    },
    startDataRefresh() {
      if (this.dataRefreshEnabled && this.dataRefreshTimer) return
      this.stopDataRefresh()
      this.dataRefreshEnabled = true
      this.dataRefreshTimer = setInterval(() => {
        if (this.carouselIndex === 0) {
          this.refreshDashboardData()
        }
      }, DATA_REFRESH_INTERVAL)
      this.visibilityChangeHandler = () => {
        if (
          document.visibilityState === 'visible' &&
          this.carouselIndex === 0 &&
          Date.now() - this.lastSuccessfulRefreshAt >= DATA_REFRESH_INTERVAL
        ) {
          this.refreshDashboardData()
        }
      }
      document.addEventListener('visibilitychange', this.visibilityChangeHandler)
    },
    stopDataRefresh() {
      this.dataRefreshEnabled = false
      this.dashboardRefreshPending = false
      this.dashboardRefreshVersion++
      if (this.dataRefreshTimer) {
        clearInterval(this.dataRefreshTimer)
        this.dataRefreshTimer = null
      }
      if (this.visibilityChangeHandler) {
        document.removeEventListener('visibilitychange', this.visibilityChangeHandler)
        this.visibilityChangeHandler = null
      }
    },
    stopRainBg() {
      this.rainStopped = true
      if (this.rainBgTimer) {
        clearTimeout(this.rainBgTimer)
        this.rainBgTimer = null
      }
      if (this.rainIdleCallbackId !== null) {
        if (typeof window.cancelIdleCallback === 'function') {
          window.cancelIdleCallback(this.rainIdleCallbackId)
        } else {
          clearTimeout(this.rainIdleCallbackId)
        }
        this.rainIdleCallbackId = null
      }
      if (this.rainAnimId) {
        cancelAnimationFrame(this.rainAnimId)
        this.rainAnimId = null
      }
      if (this.rainResizeHandler) {
        window.removeEventListener('resize', this.rainResizeHandler)
        this.rainResizeHandler = null
      }
      this.rainDropTimers.forEach(id => clearTimeout(id))
      this.rainDropTimers = []
    },
    queueRainBg() {
      if (this.rainStopped || this.rainAnimId || this.rainIdleCallbackId !== null) return
      if (this.rainBgTimer) {
        clearTimeout(this.rainBgTimer)
        this.rainBgTimer = null
      }
      const start = () => {
        this.rainIdleCallbackId = null
        if (!this.rainStopped && !this.rainAnimId) this.rainBg()
      }
      if (typeof window.requestIdleCallback === 'function') {
        this.rainIdleCallbackId = window.requestIdleCallback(start, { timeout: 2000 })
      } else {
        this.rainIdleCallbackId = setTimeout(start, 0)
      }
    },
    queueScreenScale(force = false) {
      if (force) this.lastScreenScale = null
      if (this.screenScaleFrameId !== null) return
      this.screenScaleFrameId = requestAnimationFrame(() => {
        this.screenScaleFrameId = null
        this.applyScreenScale()
      })
    },
    startScreenResize() {
      this.stopScreenResize()
      this.browserF11TipShown = false
      this.userZoom = 1
      this.screenResizeHandler = lodash.debounce(() => {
        this.queueScreenScale()
        this.checkBrowserF11Mismatch()
      }, 100)
      window.addEventListener('resize', this.screenResizeHandler)
      const wrapper = this.$refs.screenWrapperRef
      if (wrapper && typeof ResizeObserver !== 'undefined') {
        this.screenResizeObserver = new ResizeObserver(() => {
          this.queueScreenScale()
        })
        this.screenResizeObserver.observe(wrapper)
      }
      const shell = document.querySelector('.layout-shell--jsc')
      if (shell && typeof ResizeObserver !== 'undefined') {
        this.shellResizeObserver = new ResizeObserver(() => {
          this.queueScreenScale(true)
        })
        this.shellResizeObserver.observe(shell)
      }
      this.removeFullscreenListener = onFullscreenChange(() => {
        const wasFullscreen = this.isScreenFullscreen
        this.syncScreenFullscreen()
        if (wasFullscreen && !this.isScreenFullscreen) {
          const wrapper = this.$refs.screenWrapperRef
          if (wrapper) {
            wrapper.scrollTop = 0
            wrapper.scrollLeft = 0
          }
        }
        this.scheduleScreenScaleAfterLayout()
      })
      this.f11KeydownHandler = (e) => {
        if (e.key !== 'F11' && e.keyCode !== 122) {
          return
        }
        e.preventDefault()
        e.stopPropagation()
        toggleDashboardFullscreen()
      }
      window.addEventListener('keydown', this.f11KeydownHandler, true)
      this.wheelZoomHandler = (e) => {
        if (!e.ctrlKey) {
          return
        }
        e.preventDefault()
        e.stopPropagation()
        const delta = e.deltaY > 0 ? -USER_ZOOM_STEP : USER_ZOOM_STEP
        const nextZoom = Math.min(USER_ZOOM_MAX, Math.max(USER_ZOOM_MIN, this.userZoom + delta))
        if (nextZoom === this.userZoom) {
          return
        }
        const wrapper = this.$refs.screenWrapperRef
        if (wrapper) {
          const { width: visibleWidth, height: visibleHeight } = this.getContentViewportSize()
          const oldLayoutScale = this.getLayoutScale(visibleWidth, visibleHeight)
          const oldScaledWidth = DESIGN_WIDTH * oldLayoutScale * this.userZoom
          if (oldScaledWidth > 0) {
            this.zoomCenterRatioX = (wrapper.scrollLeft + wrapper.clientWidth / 2) / oldScaledWidth
          }
        }
        this.userZoom = nextZoom
        this.queueScreenScale(true)
      }
      if (wrapper) {
        wrapper.addEventListener('wheel', this.wheelZoomHandler, { capture: true, passive: false })
      }
      this.syncScreenFullscreen()
    },
    stopScreenResize() {
      const wrapper = this.$refs.screenWrapperRef
      if (wrapper && this.wheelZoomHandler) {
        wrapper.removeEventListener('wheel', this.wheelZoomHandler, { capture: true })
      }
      this.wheelZoomHandler = null
      this.userZoom = 1
      this.screenNeedsScroll = false
      this.zoomCenterRatioX = null
      if (this.f11KeydownHandler) {
        window.removeEventListener('keydown', this.f11KeydownHandler, true)
        this.f11KeydownHandler = null
      }
      this.browserF11TipShown = false
      this.clearFullscreenScaleTimers()
      if (this.screenScaleFrameId !== null) {
        cancelAnimationFrame(this.screenScaleFrameId)
        this.screenScaleFrameId = null
      }
      if (this.removeFullscreenListener) {
        this.removeFullscreenListener()
        this.removeFullscreenListener = null
      }
      if (this.screenResizeObserver) {
        this.screenResizeObserver.disconnect()
        this.screenResizeObserver = null
      }
      if (this.shellResizeObserver) {
        this.shellResizeObserver.disconnect()
        this.shellResizeObserver = null
      }
      if (this.screenResizeHandler) {
        window.removeEventListener('resize', this.screenResizeHandler)
        this.screenResizeHandler.cancel?.()
        this.screenResizeHandler = null
      }
    },
    syncScreenFullscreen() {
      this.isScreenFullscreen = isDashboardApiFullscreen()
    },
    clearFullscreenScaleTimers() {
      this.fullscreenScaleTimers.forEach(id => clearTimeout(id))
      this.fullscreenScaleTimers = []
    },
    // 全屏切换后布局需多帧才稳定，延迟多次重算避免发布环境算错尺寸
    scheduleScreenScaleAfterLayout() {
      this.clearFullscreenScaleTimers()
      this.lastScreenScale = null
      this.screenScaleForceUntil = Date.now() + 800
      const run = () => {
        this.queueScreenScale(true)
      }
      this.$nextTick(() => {
        run()
        requestAnimationFrame(() => {
          run()
          requestAnimationFrame(run)
        })
      })
      this.fullscreenScaleTimers = [
        setTimeout(run, 50),
        setTimeout(run, 150),
        setTimeout(run, 300),
        setTimeout(run, 500),
        setTimeout(run, 800)
      ]
    },
    checkBrowserF11Mismatch() {
      if (isDashboardApiFullscreen()) {
        return
      }
      if (!looksLikeBrowserF11Fullscreen()) {
        return
      }
      if (this.browserF11TipShown) {
        return
      }
      this.browserF11TipShown = true
      Message.warning('当前为浏览器全屏，请使用顶部全屏按钮或点击 Banner「驾驶舱」进入驾驶舱全屏')
    },
    // 非全屏：Banner 下方区域；全屏：.screen-wrapper 占满全屏视口
    getBannerHeight() {
      const bannerEl = document.querySelector('.app-wrapper--jsc .bannerCss')
      if (bannerEl && bannerEl.offsetHeight && bannerEl.offsetParent !== null) {
        return bannerEl.offsetHeight
      }
      return 0
    },
    getContentViewportSize() {
      const wrapper = this.$refs.screenWrapperRef
      const bannerHeight = this.getBannerHeight()
      const fallbackHeight = Math.max(0, window.innerHeight - bannerHeight)
      if (wrapper && wrapper.clientWidth > 0) {
        let height = wrapper.clientHeight
        // 退出全屏后 wrapper 可能仍短暂保持全屏高度，非全屏时用 Banner 下方高度兜底
        if (!this.isScreenFullscreen && fallbackHeight > 0 && height > fallbackHeight + 2) {
          height = fallbackHeight
        }
        if (height > 0) {
          return {
            width: wrapper.clientWidth,
            height
          }
        }
      }
      return {
        width: window.innerWidth,
        height: fallbackHeight || window.innerHeight
      }
    },
    // 全屏以宽度为基准；非全屏宽高取 min，保证完整显示设计稿
    getLayoutScale(visibleWidth, visibleHeight) {
      const widthScale = visibleWidth / DESIGN_WIDTH
      if (this.isScreenFullscreen) {
        return widthScale
      }
      return Math.min(widthScale, visibleHeight / DESIGN_HEIGHT)
    },
    // 水平以可视区垂直中线居中；Ctrl+滚轮叠加 userZoom
    applyScreenScale() {
      const { width: visibleWidth, height: visibleHeight } = this.getContentViewportSize()
      if (!visibleWidth || !visibleHeight) {
        return
      }
      const layoutScale = this.getLayoutScale(visibleWidth, visibleHeight)
      const scale = layoutScale * this.userZoom
      const scaledWidth = DESIGN_WIDTH * scale
      const scaledHeight = DESIGN_HEIGHT * scale
      const offsetX = (visibleWidth - scaledWidth) / 2
      const scaleKey = `${this.isScreenFullscreen ? 'fs' : 'nf'}_${visibleWidth.toFixed(1)}_${visibleHeight.toFixed(1)}_${this.userZoom.toFixed(4)}_${layoutScale.toFixed(4)}`
      const forceRecalc = Date.now() < this.screenScaleForceUntil
      if (!forceRecalc && this.lastScreenScale === scaleKey) {
        return
      }
      this.lastScreenScale = scaleKey

      const overflowX = scaledWidth > visibleWidth
      const overflowY = scaledHeight > visibleHeight
      this.screenNeedsScroll = !this.isScreenFullscreen || this.userZoom !== 1 || overflowX || overflowY

      this.scaleBoxStyle = {
        width: overflowX ? `${scaledWidth}px` : `${visibleWidth}px`,
        height: `${scaledHeight}px`,
        marginTop: '0'
      }
      this.screenStyle = {
        transform: `scale(${scale})`,
        transformOrigin: 'left top',
        position: 'absolute',
        left: overflowX ? '0px' : `${offsetX}px`,
        top: '0px'
      }

      this.$nextTick(() => {
        this.syncScrollPosition(visibleWidth, visibleHeight, scaledWidth, scaledHeight, overflowX, overflowY)
      })
    },
    syncScrollPosition(visibleWidth, visibleHeight, scaledWidth, scaledHeight, overflowX) {
      const wrapper = this.$refs.screenWrapperRef
      if (!wrapper) {
        return
      }
      if (overflowX) {
        if (this.zoomCenterRatioX != null) {
          wrapper.scrollLeft = this.zoomCenterRatioX * scaledWidth - wrapper.clientWidth / 2
          this.zoomCenterRatioX = null
        } else {
          wrapper.scrollLeft = (scaledWidth - visibleWidth) / 2
        }
      } else {
        wrapper.scrollLeft = 0
        this.zoomCenterRatioX = null
      }
      wrapper.scrollTop = 0
    },
    async init() {
      await this.refreshDashboardData()
    },
    setxAxisData() { // 设置x轴数据
      const nowDate = getTime()
      const recentDates = getRecentlyDate(7)
      const xAxisDate = recentDates.map(date => date.slice(5, 10))
      const startDate = recentDates[0].slice(0, 10) + ' 00:00:00'
      this.startTime = dateConvStamp(startDate) / 1000
      this.endTime = dateConvStamp(nowDate) / 1000

      const updateXAxis = chartData => {
        const current = chartData.xAxis.data || []
        if (current.length !== xAxisDate.length || current.some((item, index) => item !== xAxisDate[index])) {
          chartData.xAxis.data = xAxisDate
        }
      }
      updateXAxis(this.dpLineChart1ChartData)
      updateXAxis(this.dpLineChart2ChartData)
      updateXAxis(this.basicBarData1)
    },
    isDashboardRefreshCurrent(refreshVersion) {
      return refreshVersion === undefined || refreshVersion === this.dashboardRefreshVersion
    },
    async refreshDashboardData() {
      if (this.dashboardRefreshRunning) {
        this.dashboardRefreshPending = true
        this.dashboardRefreshVersion++
        return
      }

      const refreshVersion = ++this.dashboardRefreshVersion
      this.dashboardRefreshRunning = true
      this.dashboardRefreshPending = false
      this.setxAxisData()
      const equipmentReadyPromise = this.getEquipList(refreshVersion).then(() => true, () => false)
      try {
        await this.getCorpList(refreshVersion)
        if (!this.isDashboardRefreshCurrent(refreshVersion)) return

        let siteReady = false
        let warningReady = false
        let mapIncludesLatestWarning = false
        const updateMap = () => {
          if (!this.isDashboardRefreshCurrent(refreshVersion)) return
          this.setAreaAddrNumList()
          this.queueRainBg()
        }

        // 站点、设备、统计互不阻塞；站点查询和设备查询可同时进行。
        const sitePromise = this.getSiteList(
          this.selectCropId !== null,
          refreshVersion,
          equipmentReadyPromise
        ).then(() => {
          siteReady = true
          updateMap()
          mapIncludesLatestWarning = warningReady
        })
        const warningPromise = equipmentReadyPromise
          .then(() => this.getWarningMsg(this.selectCropId !== null, refreshVersion))
          .then(() => {
            warningReady = true
            if (siteReady && !mapIncludesLatestWarning) {
              updateMap()
              mapIncludesLatestWarning = true
            }
          })
        const dataPromise = this.getDataMsg(refreshVersion)
        // SOH实时数据量较大，等地图首批数据完成并让出主线程后再加载。
        const healthPromise = sitePromise
          .catch(() => undefined)
          .then(() => this.waitForBrowserIdle())
          .then(() => {
            if (!this.isDashboardRefreshCurrent(refreshVersion)) return
            return this.loadBatteryHealthStatistics(refreshVersion)
          })

        await Promise.allSettled([
          sitePromise,
          warningPromise,
          dataPromise,
          healthPromise
        ])
        if (!this.isDashboardRefreshCurrent(refreshVersion)) return

        this.lastSuccessfulRefreshAt = Date.now()
      } catch {
        // 静默刷新失败时保留当前画面，等待下一轮刷新。
      } finally {
        this.dashboardRefreshRunning = false
        if (this.dashboardRefreshPending && this.dataRefreshEnabled && this.carouselIndex === 0) {
          this.dashboardRefreshPending = false
          this.refreshDashboardData()
        }
      }
    },
    async getDataMsg(refreshVersion) { // 获取数据信息
      await Promise.allSettled([
        this.loadEquipmentCounts(refreshVersion),
        this.loadChargeDischargeStatistics(refreshVersion),
        this.loadAlarmStatistics(refreshVersion),
        this.loadWarningStatistics(refreshVersion)
      ])
    },
    waitForBrowserIdle() {
      return new Promise(resolve => {
        if (typeof window.requestIdleCallback === 'function') {
          window.requestIdleCallback(resolve, { timeout: 500 })
        } else {
          setTimeout(resolve, 0)
        }
      })
    },
    async getCorpList(refreshVersion) { // 获取部门列表
      const treeList = []
      const res = await corpList() // 列表（用户部门树）
      const list = res && res.return && res.return.data
      if (!Array.isArray(list)) throw new Error('部门列表数据格式异常')
      list.forEach(item => {
        treeList.push({ ...item, name: item.corpName })
      })
      if (!this.isDashboardRefreshCurrent(refreshVersion)) return
      this.treeList = treeList
      this.queryableCorpKeys = buildQueryableCorpKeys(treeList, this.userInfo)
    },
    parseSiteListResponse(res) {
      if (!res || Number(res.code) !== 200) {
        throw new Error((res && res.msg) || '站址列表查询失败')
      }
      const payload = res.return || {}
      const list = Array.isArray(payload.data) ? payload.data : []
      const responseTotal = Number(payload.total)
      const total = Number.isFinite(responseTotal) && responseTotal >= 0 ? responseTotal : list.length
      return { list, total }
    },
    async findAllSiteList(searchMsg) {
      const firstRes = await findSiteListExactMatch({
        ...searchMsg,
        pageNum: 1,
        pageSize: SITE_PAGE_SIZE
      })
      const firstPage = this.parseSiteListResponse(firstRes)
      const pageCount = Math.ceil(firstPage.total / SITE_PAGE_SIZE)
      if (pageCount <= 1) return firstPage.list

      const pageRequests = []
      for (let pageNum = 2; pageNum <= pageCount; pageNum++) {
        pageRequests.push(findSiteListExactMatch({
          ...searchMsg,
          pageNum,
          pageSize: SITE_PAGE_SIZE
        }))
      }
      const pageResponses = await Promise.all(pageRequests)
      return pageResponses.reduce((list, res) => {
        return list.concat(this.parseSiteListResponse(res).list)
      }, firstPage.list)
    },
    async getSiteList(isSearchZdsl, refreshVersion, equipmentReadyPromise) { // 获取站址列表
      const siteList = []
      const siteIds = new Set()
      const searchMsg = {
        corpId: null
      }
      const corpList = this.getSearchCorpList() // 获取查询的部门列表
      if (isSearchZdsl) {
        if (corpList.length === 0) {
          if (!this.isDashboardRefreshCurrent(refreshVersion)) return
          this.siteList = []
          this.siteCount = 0
          return
        }
        searchMsg.corpId = corpList[0]
      }
      const siteRequest = this.findAllSiteList(searchMsg)
      const list = await siteRequest
      if (equipmentReadyPromise) await equipmentReadyPromise
      list.forEach(item => {
        const siteId = item.id == null ? null : String(item.id)
        if (siteId !== null && siteIds.has(siteId)) {
          return
        }
        if (siteId !== null) {
          siteIds.add(siteId)
        }
        const treeItem = this.treeList.find(tree => String(item.corpId) === String(tree.id))
        const corpName = treeItem ? treeItem.name : ''
        const detailMsg = item.siteDetails || {}
        const equipList = this.equipmentBySiteId.get(String(item.id)) || []
        const regionName = resolveSiteRegionName({
          corpName,
          siteDetails: detailMsg
        })
        const includeWithoutCorp = !isSearchZdsl && Number(this.userInfo.isadmin) === 1
        if (!corpName && !includeWithoutCorp) return
        siteList.push({
          ...item,
          corpName,
          addr: detailMsg.addr,
          cityName: detailMsg.cityName,
          latitude: detailMsg.latitude,
          longitude: detailMsg.longitude,
          provinceName: detailMsg.provinceName,
          regionName,
          equipList
        })
      })
      if (!this.isDashboardRefreshCurrent(refreshVersion)) return
      this.siteList = markRaw(siteList)
      this.siteCount = siteList.length
    },
    async getEquipList(refreshVersion) { // 获取设备列表
      const equipList = []
      const equipmentById = new Map()
      const equipmentBySiteId = new Map()
      const res = await findEquipmentListExactMatch({
        pageNum: 1,
        pageSize: 10000
      })
      const list = res && res.return && res.return.data
      if (!Array.isArray(list)) throw new Error('设备列表数据格式异常')
      list.forEach(item => {
        if (item.positionStatus === 3) {
          equipList.push(item)
          equipmentById.set(String(item.id), item)
          const siteId = item.belongToSiteId == null ? '' : String(item.belongToSiteId)
          if (siteId) {
            if (!equipmentBySiteId.has(siteId)) equipmentBySiteId.set(siteId, [])
            equipmentBySiteId.get(siteId).push(item)
          }
        }
      })
      if (!this.isDashboardRefreshCurrent(refreshVersion)) return
      this.equipList = markRaw(equipList)
      this.equipmentById = markRaw(equipmentById)
      this.equipmentBySiteId = markRaw(equipmentBySiteId)
    },
    setAreaAddrNumList() { // 设置区域位置数量列表
      const areaMap = new Map()
      const alarmEquipmentIds = new Set(this.realtimeAlarmRecords.map(item => String(item.equipmentId)))
      const warningEquipmentIds = new Set(this.realtimeWarningRecords.map(item => String(item.equipmentId)))
      this.siteList.forEach(item => {
        const regionName = item.regionName || resolveSiteRegionName({
          corpName: item.corpName,
          siteDetails: {
            cityName: item.cityName,
            provinceName: item.provinceName
          }
        })
        const equipList = item.equipList.map(equipItem => {
          return {
            id: equipItem.id,
            equipmentName: equipItem.equipmentName,
            hasAlarm: alarmEquipmentIds.has(String(equipItem.id)),
            hasWarning: warningEquipmentIds.has(String(equipItem.id))
          }
        })
        const hasAlarm = equipList.some(equipItem => equipItem.hasAlarm || equipItem.hasWarning)
        const icon = hasAlarm ? mapAlarmImage : null
        const areaKey = item.corpId == null || Number(item.corpId) === 0 || !item.corpName
          ? regionName
          : String(item.corpId)
        const point = { // 标点信息列表
          pointName: item.siteName,
          icon,
          longitude: parseFloat(item.longitude),
          latitude: parseFloat(item.latitude),
          equipList
        }
        const infoWindowData = {
          siteName: point.pointName,
          addr: item.addr,
          equipList: point.equipList,
          longitude: point.longitude,
          latitude: point.latitude
        }
        point.getInfoWindowDom = () => this.setAmapInfoWindow(infoWindowData)

        let areaItem = areaMap.get(areaKey)
        if (!areaItem) {
          areaItem = {
            areaKey,
            moreAreaName: regionName, // 区域名称(用于地图内部匹配)
            pointList: []
          }
          areaMap.set(areaKey, areaItem)
        }
        areaItem.pointList.push(point)
      })
      this.areaAddrMsgList = markRaw(Array.from(areaMap.values()))
    },
    getSearchCorpList() { // 获取查询的部门列表
      let corpList = lodash.cloneDeep(this.userInfo.corpIds) // 部门id数组
      if (this.userInfo.isadmin === 1) {
        this.treeList.forEach(item => {
          if (item.level === 0) {
            corpList.push(item.id)
          }
        })
      }
      if (this.selectCropId || this.selectCropId === 0) {
        corpList = [this.selectCropId]
      } else if (this.selectCropId === false) {
        corpList = []
      }
      return corpList
    },
    findCorpIdByRegionName(regionName) { // 根据地图区域名匹配部门id
      const matchedCorp = findBestCorpByRegionName(
        this.treeList,
        regionName,
        this.queryableCorpKeys
      )
      return matchedCorp ? matchedCorp.id : false
    },
    async loadEquipmentCounts(refreshVersion) { // 获取设备数量
      const corpList = this.getSearchCorpList() // 获取查询的部门列表

      let inboundCount = 0 // 入库数量
      let outboundCount = 0 // 出库数量
      let stationEntryCount = 0 // 上站入网数量
      let stationExitCount = 0 // 退网下站数量
      let inspectionReadyCount = 0 // 检测待用数量
      let retiredCount = 0 // 检测淘汰数量
      if (corpList.length === 0) {
        if (!this.isDashboardRefreshCurrent(refreshVersion)) return
        this.lifecycleSummaryItems.forEach(item => {
          switch (item.title) {
            case '入库数量':
              item.msg = inboundCount
              break
            case '出库数量':
              item.msg = outboundCount
              break
            case '上站入网':
              item.msg = stationEntryCount
              break
            case '退网下站':
              item.msg = stationExitCount
              break
            case '检测待用':
              item.msg = inspectionReadyCount
              break
            case '检测淘汰':
              item.msg = retiredCount
              break
          }
        })
        this.onlineBatteryCount = stationEntryCount // 入网电池数量
        return
      }

      const countRequests = [1, 2, 3, 4, 5, 6].map(positionStatus => getCountbycorp({
        filterFunc: 'BinC', // ConE: 关注设备， BinC: 绑定部门
        positionStatus,
        corpList
      }))
      const countResponses = await Promise.all(countRequests)
      const counts = countResponses.map(res => {
        const countList = res && res.return && res.return.count
        if (!Array.isArray(countList)) throw new Error('设备数量统计数据格式异常')
        return countList.reduce((total, value) => total + Number(value || 0), 0)
      })
      inboundCount = counts[0]
      outboundCount = counts[1]
      stationEntryCount = counts[2]
      stationExitCount = counts[3]
      inspectionReadyCount = counts[4]
      retiredCount = counts[5]

      if (!this.isDashboardRefreshCurrent(refreshVersion)) return

      this.lifecycleSummaryItems.forEach(item => {
        switch (item.title) {
          case '入库数量':
            item.msg = inboundCount
            break
          case '出库数量':
            item.msg = outboundCount
            break
          case '上站入网':
            item.msg = stationEntryCount
            break
          case '退网下站':
            item.msg = stationExitCount
            break
          case '检测待用':
            item.msg = inspectionReadyCount
            break
          case '检测淘汰':
            item.msg = retiredCount
            break
        }
      })
      this.onlineBatteryCount = stationEntryCount // 入网电池数量
    },
    async loadChargeDischargeStatistics(refreshVersion) { // 获取充放电统计数据
      const showXAxisDate = this.basicBarData1.xAxis.data
      const corpList = this.getSearchCorpList() // 获取查询的部门列表
      const seriesData2 = new Array(showXAxisDate.length).fill(0)
      const seriesData3 = new Array(showXAxisDate.length).fill(0)
      const res = corpList.length > 0
        ? await getChargeDischargeDailyStatistics({
          startTime: this.startTime,
          endTime: this.endTime,
          corpList
        })
        : { return: { data: [] }}
      const dataList = res && res.return && res.return.data
      if (!Array.isArray(dataList)) throw new Error('充放电统计数据格式异常')
      const allDataList = []
      dataList.forEach(item => {
        const marks = item.mark || []
        marks.forEach((time, index) => {
          const markDate = time.slice(5, 10)
          allDataList.push({
            statisticName: item.code,
            markDate,
            statisticValue: item.value[index]
          })
        })
      })
      allDataList.forEach(item => {
        showXAxisDate.forEach((xAxisItem, index) => {
          if (item.markDate === xAxisItem) {
            switch (item.statisticName) {
              case '累计放电量':
                seriesData2[index] += parseFloat(item.statisticValue)
                break
              case '累计充电量':
                seriesData3[index] += parseFloat(item.statisticValue)
                break
            }
          }
        })
      })

      for (let i = 0; i < seriesData2.length; i++) {
        seriesData2[i] = seriesData2[i] ? seriesData2[i].toFixed(1) : 0
        seriesData3[i] = seriesData3[i] ? seriesData3[i].toFixed(1) : 0
      }
      if (!this.isDashboardRefreshCurrent(refreshVersion)) return
      this.basicBarData1.series.forEach(item => {
        switch (item.name) {
          case '累计放电量': item.data = seriesData2; break
          case '累计充电量': item.data = seriesData3; break
        }
      })
    },
    async loadAlarmStatistics(refreshVersion) { // 获取告警统计数据
      const corpList = this.getSearchCorpList() // 获取查询的部门列表

      const showXAxisDate = this.dpLineChart1ChartData.xAxis.data
      const totalAlarmDatas = new Array(showXAxisDate.length).fill(0)
      const dischargeOvercurrentDatas = new Array(showXAxisDate.length).fill(0)
      const chargeHighTemperatureDatas = new Array(showXAxisDate.length).fill(0)
      const dischargeHighTemperatureDatas = new Array(showXAxisDate.length).fill(0)

      const res = await getDailyAlarmStatistics({
        startTime: this.startTime,
        endTime: this.endTime,
        corpList
      })
      const dataList = res && res.return && res.return.data
      if (!Array.isArray(dataList)) throw new Error('告警统计数据格式异常')
      dataList.forEach(item => {
        const markDate = String(item.markDate || '').slice(5, 10)
        const xAxisIndex = showXAxisDate.indexOf(markDate)
        if (xAxisIndex < 0) return

        totalAlarmDatas[xAxisIndex] = item.totalAlarmCount || 0
        dischargeOvercurrentDatas[xAxisIndex] = item.dischargeOvercurrentCount || 0
        chargeHighTemperatureDatas[xAxisIndex] = item.chargeHighTemperatureCount || 0
        dischargeHighTemperatureDatas[xAxisIndex] = item.dischargeHighTemperatureCount || 0
      })
      if (!this.isDashboardRefreshCurrent(refreshVersion)) return
      this.dpLineChart1ChartData.series.forEach(item => {
        switch (item.name) {
          case '总告警数': item.data = totalAlarmDatas; break
          case '放电过流': item.data = dischargeOvercurrentDatas; break
          case '充电高温': item.data = chargeHighTemperatureDatas; break
          case '放电高温': item.data = dischargeHighTemperatureDatas; break
        }
      })
    },
    async loadWarningStatistics(refreshVersion) { // 获取预警统计数据
      const corpList = this.getSearchCorpList() // 获取查询的部门列表

      const showXAxisDate = this.dpLineChart2ChartData.xAxis.data
      const totalWarningDatas = new Array(showXAxisDate.length).fill(0)
      const socWarningDatas = new Array(showXAxisDate.length).fill(0)
      const sohWarningDatas = new Array(showXAxisDate.length).fill(0)

      const res = await getDailyWarningStatistics({
        startTime: this.startTime,
        endTime: this.endTime,
        corpList
      })
      const dataList = res && res.return && res.return.data
      if (!Array.isArray(dataList)) throw new Error('预警统计数据格式异常')
      dataList.forEach(item => {
        const markDate = String(item.markDate || '').slice(5, 10)
        const xAxisIndex = showXAxisDate.indexOf(markDate)
        if (xAxisIndex < 0) return

        totalWarningDatas[xAxisIndex] = item.totalWarningCount || 0
        socWarningDatas[xAxisIndex] = item.socWarningCount || 0
        sohWarningDatas[xAxisIndex] = item.sohWarningCount || 0
      })
      if (!this.isDashboardRefreshCurrent(refreshVersion)) return
      this.dpLineChart2ChartData.series.forEach(item => {
        switch (item.name) {
          case '总预警数': item.data = totalWarningDatas; break
          case 'SOC预警': item.data = socWarningDatas; break
          case 'SOH预警': item.data = sohWarningDatas; break
        }
      })
    },
    async loadBatteryHealthStatistics(refreshVersion) { // 获取电池健康度统计数据
      const batteryHealthSeriesData = this.batteryHealthChartData.series[0].data

      if (this.selectCropId === false) {
        if (!this.isDashboardRefreshCurrent(refreshVersion)) return
        this.healthStatisticsStatus = 'empty'
        return
      }

      const hadValidData = this.healthStatisticsStatus === 'ready'
      try {
        const corpId = this.selectCropId || this.selectCropId === 0 ? this.selectCropId : null
        // 与健康状态检测页使用同一实时取数链路，采集时间不作为过期条件。
        const res = await getEquipAllPoint({
          corpId,
          pageNum: 1,
          pageSize: 10000,
          positionStatus: 3
        })
        const list = res && res.return && Array.isArray(res.return.data)
          ? res.return.data
          : []
        list.forEach(item => {
          item.labelList = (item.dataDetails || [])
            .map(detail => detail.labelId)
            .filter(labelId => labelId !== undefined && labelId !== null && labelId !== '')
        })

        const realtimeResponse = list.length > 0
          ? await getapilistdata({ equipList: list })
          : { equipList: [] }
        const realtimeList = realtimeResponse && Array.isArray(realtimeResponse.equipList)
          ? realtimeResponse.equipList
          : []
        const levelCounts = {
          '健康度优': 0,
          '健康度良': 0,
          '健康度中': 0,
          '健康度差': 0
        }
        let validCount = 0
        realtimeList.forEach(item => {
          const sohDetail = (item.dataDetails || []).find(detail => {
            const labelName = String(detail.labelName || detail.name || '').trim().toUpperCase()
            return labelName === 'SOH' || labelName.includes('SOH')
          })
          if (!sohDetail || sohDetail.realTimeVal === '异常') return
          const soh = Number(sohDetail.realTimeVal)
          if (!Number.isFinite(soh) || soh < 0) return

          validCount++
          if (soh > 90) levelCounts['健康度优']++
          else if (soh >= 80) levelCounts['健康度良']++
          else if (soh >= 60) levelCounts['健康度中']++
          else levelCounts['健康度差']++
        })

        if (validCount === 0) {
          if (!hadValidData && this.isDashboardRefreshCurrent(refreshVersion)) {
            this.healthStatisticsStatus = 'empty'
          }
          return
        }
        if (!this.isDashboardRefreshCurrent(refreshVersion)) return
        batteryHealthSeriesData.forEach(item => {
          item.value = Number(((levelCounts[item.name] / validCount) * 100).toFixed(2))
        })
        this.healthStatisticsStatus = 'ready'
      } catch (error) {
        if (!hadValidData && this.isDashboardRefreshCurrent(refreshVersion)) {
          this.healthStatisticsStatus = 'empty'
        }
        throw error
      }
    },
    goDetailPage(name) { // 前往详细界面
      this.stopRainBg()
      let bannerId = ''
      let url = ``
      switch (name) {
        case '电池统计': bannerId = '12'; url = `/dczt`; break
        case '实时告警': bannerId = '13'; url = `/gjxxssgj`; break
        case '电池健康度统计': bannerId = '13'; url = `/jkztjc`; break
        case '告警统计': bannerId = '13'; url = `/gjxxlsgj`; break
        case '预警统计': bannerId = '13'; url = `/yjxxlsyj`; break
        case '实时预警': bannerId = '13'; url = `/yjxxssyj`; break
      }
      localStorage.setItem('bannerId', bannerId)
      localStorage.setItem('routerPath', url)
      setTimeout(() => {
        this.$router.push(url)
      }, 200)
    },
    async levelChange(e) { // 级别变化
      if (e.value === 0) {
        Message({
          message: `该地区未开通业务`,
          type: 'warning',
          duration: 3 * 1000
        })
        return
      }
      let selectCrop = ''
      switch (e.level) {
        case 'china': selectCrop = '中国'; break
        case 'province': selectCrop = e.areaName; break
        case 'city': selectCrop = e.areaName; break
      }

      this.selectCropId = this.findCorpIdByRegionName(selectCrop)
      await this.refreshDashboardData()
    },
    goToDetail(id) {
      this.equipDetailMsg = this.equipmentById.get(String(id)) || {}
      this.showDialog = true
    },
    setAmapInfoWindow(row) { // 信息窗体
      const alarmIcon = alarmImage
      const warningIcon = warningImage

      const dom = document.createElement('div')
      const contentHtmlStart = `<div class='Mo_banCss'>
                                  <div class='Mo_ban_boxCss'>
                                    <div class="Mo_ban_box_closeCss" onclick="closeInfoWindow()">X</div>
                                    <div class="moLineCss">站址名称：${row.siteName}</div>
                                    <div class="moLineMsgCss">电池组数量：${row.equipList.length}</div>
                                    <div class="moLineMsgCss">电池总负载：${row.equipList.length}A</div>
                                    <div class="moLineAddrCss">站址地址：${row.addr ? row.addr : ''}</div>`

      let contentHtml = ''
      row.equipList.forEach((equipItem, index) => {
        contentHtml += `<div class="moLineCss">
                                      <div class="moLine1Css">电池组${index + 1}：${equipItem.equipmentName}</div>`
        equipItem.hasAlarm ? contentHtml += `<div class="moLineIconCss"><img src="${alarmIcon}" /></div>` : null
        equipItem.hasWarning ? contentHtml += `<div class="moLineIconCss"><img src="${warningIcon}" /></div>` : null
        contentHtml += `<div id="Mo_ban_lookCss" onclick="goToDetail(${equipItem.id})">查看详情</div>
                                    </div>`
      })
      const contentHtmlEnd = `</div>
                                </div>`
      dom.innerHTML = contentHtmlStart + contentHtml + contentHtmlEnd
      return dom
    },
    // 中间背景雨
    rainBg() {
      const c = this.$refs.rainCanvasRef
      const container = this.$refs.rainContentRef
      if (!c || !container || this.rainStopped || this.rainAnimId) {
        return
      }
      const ctx = c.getContext('2d')
      if (!ctx) {
        return
      }
      let w = c.width = container.clientWidth
      let h = c.height = container.clientHeight
      if (!w || !h) {
        return
      }
      const vm = this
      // 设置canvas宽、高
      function random(min, max) {
        return Math.random() * (max - min) + min
      }

      function RainDrop() { }
      // 雨滴对象 这是绘制雨滴动画的关键
      RainDrop.prototype = {
        init: function() {
          this.x = random(0, w)// 雨滴的位置x
          this.y = h// 雨滴的位置y
          this.color = 'hsl(180, 100%, 50%)'// 雨滴颜色 长方形的填充色
          this.vy = random(4, 5)// 雨滴下落速度
          this.hit = 0// 下落的最大值
          this.size = 2// 长方形宽度
        },
        draw: function() {
          if (this.y > this.hit) {
            var linearGradient = ctx.createLinearGradient(this.x, this.y, this.x, this.y + this.size * 30)
            // 设置起始颜色
            linearGradient.addColorStop(0, '#14789c')
            // 设置终止颜色
            linearGradient.addColorStop(1, '#090723')
            // 设置填充样式
            ctx.fillStyle = linearGradient
            ctx.fillRect(this.x, this.y, this.size, this.size * 50)// 绘制长方形，通过多次叠加长方形，形成雨滴下落效果
          }
          this.update()// 更新位置
        },
        update: function() {
          if (this.y > this.hit) {
            this.y -= this.vy// 未达到底部，增加雨滴y坐标
          } else {
            this.init()
          }
        }
      }

      function resize() {
        if (vm.rainStopped || !container.isConnected) {
          return
        }
        w = c.width = container.clientWidth
        h = c.height = container.clientHeight
      }

      const rs = []
      for (let i = 0; i < 10; i++) {
        const timerId = setTimeout(function() {
          if (vm.rainStopped) {
            return
          }
          const r = new RainDrop()
          r.init()
          rs.push(r)
        }, i * 300)
        this.rainDropTimers.push(timerId)
      }

      function anim() {
        if (vm.rainStopped || !c.isConnected) {
          vm.rainAnimId = null
          return
        }
        ctx.clearRect(0, 0, w, h)
        for (let i = 0; i < rs.length; i++) {
          rs[i].draw()
        }
        vm.rainAnimId = requestAnimationFrame(anim)
      }

      if (this.rainResizeHandler) {
        window.removeEventListener('resize', this.rainResizeHandler)
      }
      this.rainResizeHandler = resize
      window.addEventListener('resize', this.rainResizeHandler)
      anim()
    },
    updateScrollBoardRows(refName, configName, rows) {
      const board = this.$refs[refName]
      if (this.lastSuccessfulRefreshAt > 0 && board && typeof board.updateRows === 'function') {
        board.updateRows(rows)
        return
      }
      this[configName] = {
        ...this[configName],
        data: rows
      }
    },
    async getWarningMsg(isSelectedCropId, refreshVersion) { // 获取告警预警信息
      if (isSelectedCropId && this.selectCropId === false) {
        if (!this.isDashboardRefreshCurrent(refreshVersion)) return
        this.updateScrollBoardRows('realtimeAlarmBoardRef', 'realtimeAlarmBoardConfig', [])
        this.updateScrollBoardRows('realtimeWarningBoardRef', 'realtimeWarningBoardConfig', [])
        this.realtimeAlarmRecords = []
        this.realtimeWarningRecords = []
        this.alarmCount = 0
        this.warningCount = 0
        return
      }

      let corpId = false
      if (isSelectedCropId) {
        corpId = this.selectCropId
      }
      const [alarmRes, warningRes] = await Promise.all([
        findAlarmRecord({
          pageNum: 1,
          pageSize: 10000,
          corpId,
          alarmType: '0', // 告警
          alarmStatus: 1 // 实时
        }),
        findAlarmRecord({
          pageNum: 1,
          pageSize: 10000,
          corpId,
          alarmType: '1', // 预警
          alarmStatus: 1 // 实时
        })
      ])
      const alarmData = alarmRes && alarmRes.return && alarmRes.return.data
      const warningData = warningRes && warningRes.return && warningRes.return.data
      if (!Array.isArray(alarmData) || !Array.isArray(warningData)) {
        throw new Error('实时告警预警数据格式异常')
      }

      const buildRealtimeRows = data => {
        const records = data.map(item => {
          return {
            ...item,
            cfsj: formatDateString(item.beginTime, 'timeStmapReset'),
            hfsj: formatDateString(item.endTime, 'timeStmapReset'),
            alarmValue: setNumerOrStrNum({
              value: item.alarmValue,
              fixed: 2
            }),
            equipmentCode: (this.equipmentById.get(String(item.equipmentId)) || {}).equipmentCode || item.equipmentCode || ''
          }
        })
        const rows = records.map(item => [
          `<div style="white-space:normal;word-break:break-all;" >${item.alarmCode}</div>`,
          `<div style="white-space:normal;word-break:break-all;" >${item.equipmentCode}</div>`,
          `<div style="white-space:normal;word-break:break-all;" >${item.cfsj.slice(0, 10)}</div>`
        ])
        return { records, rows }
      }
      const alarmResult = buildRealtimeRows(alarmData)
      const warningResult = buildRealtimeRows(warningData)
      const alarmTotal = alarmRes && alarmRes.return ? alarmRes.return.total : undefined
      const warningTotal = warningRes && warningRes.return ? warningRes.return.total : undefined
      const parsedAlarmTotal = Number(alarmTotal)
      const parsedWarningTotal = Number(warningTotal)

      if (!this.isDashboardRefreshCurrent(refreshVersion)) return
      this.updateScrollBoardRows('realtimeAlarmBoardRef', 'realtimeAlarmBoardConfig', alarmResult.rows)
      this.updateScrollBoardRows('realtimeWarningBoardRef', 'realtimeWarningBoardConfig', warningResult.rows)
      this.realtimeAlarmRecords = alarmResult.records // 实时告警数据列表
      this.alarmCount = alarmTotal !== undefined && alarmTotal !== null && Number.isFinite(parsedAlarmTotal)
        ? parsedAlarmTotal
        : alarmResult.records.length // 当前告警数量
      this.realtimeWarningRecords = warningResult.records // 实时预警数据列表
      this.warningCount = warningTotal !== undefined && warningTotal !== null && Number.isFinite(parsedWarningTotal)
        ? parsedWarningTotal
        : warningResult.records.length // 当前预警数量
    }
  }
}
</script>

<style lang="stylus" scoped>
@font-face {
  font-family 'FZXBSK_JSC'
  src url('../../../assets/ttf/FZXBSK-jsc.woff2') format('woff2')
  font-display swap
}
@font-face {
  font-family 'FZHTK_JSC'
  src url('../../../assets/ttf/FZHTK-jsc.woff2') format('woff2')
  font-display swap
}
.screen-wrapper {
  position relative
  width 100%
  height 100%
  overflow hidden
  background-color #020B41
}
.screen-wrapper--scroll {
  overflow auto
}
.screen-wrapper--dialog-open {
  overflow hidden
}
.screen-scale-box {
  position relative
}
.screen-wrapper:fullscreen,
.screen-wrapper:-webkit-full-screen {
  width 100vw !important
  height 100vh !important
}
@media (display-mode: fullscreen) {
  .screen-wrapper {
    width 100vw
    height 100vh
  }
}
.jscCss {
  width 1920px
  height 1080px
  overflow hidden
  will-change transform
  .bigPingCss {
    position relative
    width 100%
    height 100%
    background-color #020B41
    .topBannerCss {
      position absolute
      top 0
      left 0
      width 100%
      height 78.72px
      .headerCss {
        position absolute
        top 0
        left 0
        color #fff
        width 100%
        .titleCss {
          position absolute
          width 100%
          font-size 28.8px
          line-height 78.72px
          text-align center
          font-family FZXBSK_JSC // 方正小标宋简体
        }
      }
    }
    .bingPingBackgroundCss {
      position absolute
      top 0
      left 0
      width 100%
      height 100%
      pointer-events none
    }
    .imgCss {
      float left
      margin 18.09px 9.6px 0 9.6px
      width 56.51px
      height 48px
    }
    .imgBackgroundCss {
      position absolute
      top 0
      left 0
      width 100%
      height 100%
      pointer-events none
    }
    .msgBorderTitleCss {
      position relative
      width 100%
      text-align center
      color #25E0EF
      font-size 19.2px
      cursor pointer
      font-family FZHTK_JSC // 方正黑体
    }
    .msgBorderTitleDctjCss {
      line-height 38.4px
    }
    .msgBorderTitleGjtjCss {
      line-height 42.24px
    }
    .msgBorderTitleYjtjCss {
      line-height 44.16px
    }
    .msgBorderRightTitleCss {
      position relative
      width 100%
      text-align center
      color #25E0EF
      font-size 19.2px
      cursor pointer
      font-family FZHTK_JSC // 方正黑体
    }
    .msgBorderTitleSsgjCss {
      line-height 44.16px
    }
    .msgBorderTitleSsyjCss {
      line-height 42.24px
    }
    .realtimeAlarmCountCss,
    .realtimeWarningCountCss {
      font-variant-numeric tabular-nums
    }
    .realtimeAlarmCountCss {
      color #ff7a9e
    }
    .realtimeWarningCountCss {
      color #ffd84d
    }
    .msgBorderTitleDcjkdtjCss {
      line-height 44.16px
    }
    .msgBorderContentOneCss { // 电池统计
      position absolute
      top 55.68px
      left 0
      width 489.6px
      height 342.72px
      .gfxtContentCss {
        width 100%
        height calc(100% - 57.6px)
        color #fff
        .gfxtDetailCss {
          position relative
          float left
          padding 38.4px 0 0 28.8px
          width 32%
          height 42%
          .gfxtImgCss {
            float left
            width 57.6px
            height 64.9px
          }
          .gfxtDetailTitleAndMsgCss {
            float left
            padding-top 3.84px
            width calc(100% - 67.2px)
            .gfxtDetailTitleMsgCss {
              width 100%
              height 28.8px
              line-height 28.8px
              font-size 13.44px
              text-align center
            }
            .gfxtDetailMsgCss {
              width 100%
              height 28.8px
              line-height 28.8px
              text-align center
              font-size 23.04px
              font-family DS_DIGIT // 电子时钟专用字体
              color #69ECFF
            }
          }
        }
      }
    }
    .msgBorderLeftImgCss {
      position absolute
      top 379.07px
      left 470.4px
      width 19.2px
      height 38.4px
    }
    .msgBorderContentFourCss { // 告警统计
      position absolute
      top 399.36px
      left 0
      width 489.6px
      height 342.72px
    }
    .msgBorderLeftBotoomImgCss {
      position absolute
      top 717.12px
      left 3.84px
      width 19.2px
      height 38.4px
    }
    .msgBorderContentFiveCss { // 预警统计
      position absolute
      top 739.2px
      left 0
      width 489.6px
      height 342.72px
      .tableCss {
        position absolute
        top 48px
        left 3%
        width 94%
        height 264.96px
      }
    }
    .msgBorderContentRightCss {
      position absolute
      right 19.2px
      width 489.6px
    }
    .msgBorderContentTwoCss { // 实时告警
      position absolute
      top 55.68px
      height 342.72px
      .msgBorderContentSevenContentCss {
        position absolute
        top 76.8px
        left 24px
        width 441.6px
        height 249.6px
        .BasicPieCss {
          position absolute
          top -19.2px
          left 9.6px
          width 100%
          height 100%
          .healthStatisticsPlaceholderCss {
            display flex
            align-items center
            justify-content center
            width 100%
            height 100%
            color rgba(255, 255, 255, 0.58)
            font-size 16px
          }
        }
      }
    }
    .msgBorderRightImgCss {
      position absolute
      top 379.07px
      right 485.76px
      width 19.2px
      height 38.4px
    }
    .msgBorderContentSixCss { // 实时预警
      position absolute
      top 399.36px
      height 342.72px
    }
    .msgBorderRightBottomImgCss {
      position absolute
      top 719.04px
      right 19.2px
      width 19.2px
      height 38.4px
    }
    .msgBorderContentSevenCss {
      position absolute
      top 739.2px
      width 489.6px
      height 342.72px
      .msgBorderContentSixChartContentCss {
        position absolute
        top 46.08px
        left 3%
        width 94%
        height 268.8px
        .tableCss {
          width 100%
          height 100%
        }
      }
    }
    .msgBorderContentThreeCss {
      position absolute
      top 743.04px
      left 493.44px
      width 912px
      height 333.12px
      .msgBorderThreeTitleContentCss {
        position relative
        width 100%
        height 57.6px
        .msgBorderThreeTitleContentChildCss {
          position absolute
          top 50%
          left 50%
          transform translateX(-50%) translateY(-50%)
          .msgBorderThreeTitleLeftImgCss {
            position relative
            float left
            width 140.16px
            height 32.64px
            transform rotate(180deg)
          }
          .msgBorderThreeTitleCss {
            float left
            height 32.64px
            line-height 32.64px
            color #25E0EF
            font-size 19.2px
            font-family FZHTK_JSC // 方正黑体
          }
          .msgBorderThreeTitleRightImgCss {
            position relative
            float left
            width 140.16px
            height 32.64px
          }
        }

      }
      .msgBorderChartCss {
        position absolute
        top 76.8px
        left 0px
        width 887.04px
        height 316.8px
        .multipleYAxesCss {
          position absolute
          top -19.2px
          left 19.2px
          width 98%
        }
      }
    }
    .msgBorderContentFourAndFiveCss {
      position absolute
      top 48px
      left 19.2px
      width 470.4px
      height 270.72px
      z-index 2
      pointer-events auto
    }
    .mapBannerCss {
      position absolute
      top 76.8px
      left 497.28px
      width 906.24px
      .pngAndMsgCss {
        position relative
        float left
        margin 9.6px 0 0 38.4px
        width 178.56px
        .pngAndMsgDataCss {
          float left
          width 168.96px
          color #25E0EF
          .msgTitleCss {
            width 100%
            text-align center
            height 28.8px
            line-height 28.8px
            font-size 19.2px
            font-family FZHTK_JSC // 方正黑体
          }
          .msgCss {
            position relative
            width 100%
            height 49.92px
            line-height 49.92px
            text-align center
            .msgNumCss {
              width 100%
              font-size 34.56px
              font-family DS_DIGIT // 电子时钟专用字体
              color #f4ea2a
            }
          }
        }
      }
    }
    .mapContentCss {
      position absolute
      top 172.8px
      left 497.28px
      width 906.24px
      height 537.6px
      z-index 2
      .amap-page-container {
        width 100%
        height 100% /*写给不支持calc()的浏览器*/
        height -moz-calc(100% - 10px)
        height -webkit-calc(100% - 10px)
        height calc(100% - 10px)
      }
    }
    .mapBackCss {
      position absolute
      top -38.4px
      left 499.2px
      z-index 1
      width 902.4px
      height 902.4px
      pointer-events none
      .cicle3 {
        width 100%
        height 100%
        background url(./rotateImages/circle.png) no-repeat center
        background-size: 100%
        position absolute
        top 20%
        left 50%
        transform-style preserve-3d
        transform translateX(-50%) rotateX(75deg)
        animation rotate3 20s linear infinite
      }
      .cicle4 {
        width 328.7px
        height 328.7px
        position absolute
        top 55%
        left 50%
        transform-style preserve-3d
        background url(./rotateImages/53gqright.png) no-repeat center
        background-size 100%
        transform translateX(-50%) rotateX(75deg)
        animation rotate3 2s linear infinite
      }

      .cicle5 {
        width 328.7px
        height 328.7px
        position absolute
        top 57%
        left 50%
        transform-style preserve-3d
        background url(./rotateImages/53gqleft.png) no-repeat center
        background-size: 100%
        transform translateX(-50%) rotateX(75deg)
        animation rotate4 2s linear infinite
      }

      .cicle6 {
        width 263.04px
        height 263.04px
        position absolute
        top 65%
        left 50%
        transform-style preserve-3d
        background url(./rotateImages/535gqbottomright.png) no-repeat center
        background-size 100%
        transform translateX(-50%) rotateX(75deg)
        animation rotate3 2s linear infinite
      }

      .cicle7 {
        width 263.04px
        height 263.04px
        position  absolute
        top 67%
        left 50%
        transform-style preserve-3d
        background url(./rotateImages/53gqbottomleft.png) no-repeat center
        background-size: 100%
        transform translateX(-50%) rotateX(75deg)
        animation rotate4 2s linear infinite
      }
      // 旋转动画3
      @keyframes rotate3 {
        0% {
          transform translateX(-50%) rotateX(75deg) rotateZ(0)
        }

        100% {
          transform translateX(-50%) rotateX(75deg) rotateZ(360deg)
        }
      }
      // 旋转动画4
      @keyframes rotate4 {
        0% {
          transform translateX(-50%) rotateX(75deg) rotateZ(0)
        }

        100% {
          transform translateX(-50%) rotateX(75deg) rotateZ(-360deg)
        }
      }
    }
    .rainContentCss {
      position absolute
      top 172.8px
      left 497.28px
      width 906.24px
      height 537.6px
      z-index 1
      pointer-events none
    }
  }
}

img {
  width 100%
  height 100%
}
</style>

<style lang="stylus">
.jkztjcCss {
  height: 100%;
}

/* Keep the dialog outside the scaled dashboard and inside its fullscreen root. */
.jscDcDetailOverlay {
  position: absolute !important;
  inset: 0;
  overflow: hidden;

  .el-overlay-dialog {
    position: absolute;
    inset: 0;
    overflow: hidden;
  }
}

.jscDcDetailDialog {
  display: flex;
  flex-direction: column;
  height: 80vh;
  max-height: calc(100% - 32px);
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

.tableCss {
  height: 100%;
  box-sizing: border-box;

  .dv-scroll-board {
    width: 100%;
    height: 100%;
  }

  .dv-scroll-board .ceil {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>
