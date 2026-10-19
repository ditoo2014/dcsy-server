<template>
  <div :style="{height:height,width:width}">
    <el-button
      v-if="level !== 'china'"
      size="small"
      class="buttonCss"
      @click="goBackAddr"
    >
      返回上一级
    </el-button>
    <RegionMap
      v-if="level === 'china'"
      ref-key="chinaRef"
      :width="width"
      :height="height"
      :all-ego-data="allEgoData"
      :chart-data="chartData"
      @choose="enterProvince"
    />
    <RegionMap
      v-else-if="level === 'province'"
      ref-key="provinceRef"
      :width="width"
      :height="height"
      :all-ego-data="allEgoData"
      :ego-msg="egoMsg"
      :chart-data="chartData"
      @choose="enterProvince"
    />
    <DetailMap
      v-else-if="level === 'city'"
      :width="width"
      :height="height"
      :addr-name="addrName"
      :marker-list="markerList"
      :amap-window-info="amapWindowInfo"
      @back="goBackAddr"
      @go-to-detail="goToDetail"
    />
  </div>
</template>

<script>
import RegionMap from './RegionMap.vue'
import DetailMap from './DetailMap.vue'
import { chinaRegions, judgeRegionType, lastudgeRegionType, regionNameMatchScore, findBestCorpByRegionName } from './index.js'
import { expandCorpKeys } from '@/utils/corpScope.js'
import axios from 'axios'
import { loadPinYinInit } from '@/utils/utils.js'

export default {
  components: {
    RegionMap,
    DetailMap
  },
  props: {
    width: {
      type: String,
      default: '500px'
    },
    height: {
      type: String,
      default: '500px'
    },
    areaAddrMsgList: { // 区域位置信息列表
      type: Array,
      default: () => {
        return []
      }
    },
    corpTreeList: {
      type: Array,
      default: () => []
    },
    queryableCorpKeys: {
      type: Array,
      default: () => []
    }
  },
  data() {
    return {
      level: 'china', // 层级，china, province, city
      historyEgoMsgList: [], // 加载过的历史层级信息列表
      egoMsg: {},
      addrName: '',
      areaName: '',
      markerList: [], // 标注点列表
      amapWindowInfo: '', // 信息窗口内容

      allEgoData: {},
      chartData: {
        seriesData: []
      },
      areaRefreshTimer: null
    }
  },
  watch: {
    areaAddrMsgList: {
      handler() {
        this.scheduleMapDataRefresh()
      },
      immediate: true
    },
    corpTreeList: {
      handler() {
        this.scheduleMapDataRefresh()
      }
    },
    queryableCorpKeys: {
      handler() {
        this.scheduleMapDataRefresh()
      }
    }
  },
  mounted() {
    this.init()
  },
  beforeUnmount() {
    if (this.areaRefreshTimer) {
      clearTimeout(this.areaRefreshTimer)
      this.areaRefreshTimer = null
    }
  },
  methods: {
    async init() {
      const egoMsg = {
        areaName: '中国',
        pingYinName: 'china',
        adcode: 100000 // 中国的地区代码
      }
      this.egoMsg = egoMsg
      this.historyEgoMsgList.push(egoMsg)

      await this.getEgoData() // 获取地图数据
      this.setSeriesData()
      if (this.level === 'city') this.setMarkList()
    },
    scheduleMapDataRefresh() {
      if (this.areaRefreshTimer) {
        clearTimeout(this.areaRefreshTimer)
      }
      this.areaRefreshTimer = setTimeout(() => {
        this.areaRefreshTimer = null
        this.setSeriesData()
        if (this.level === 'city') this.setMarkList()
      }, 0)
    },
    async getEgoData() { // 获取地图数据
      let allEgoData = null
      const egoMsg = this.egoMsg
      let egoDataDetailListMsg = localStorage.getItem('egoDataDetailListMsg') // GeoJson详细数据
      if (!egoDataDetailListMsg) { // 如果没有缓存数据，则请求接口获取
        const egoDataRes = await this.getEgoCode(egoMsg) // 获取GeoJson数据
        const egoDataList = [{
          name: egoMsg.pingYinName,
          data: egoDataRes
        }]
        allEgoData = egoDataRes
        egoDataDetailListMsg = JSON.stringify(egoDataList)
      } else {
        const egoDataList = JSON.parse(egoDataDetailListMsg)
        egoDataList.forEach(item => {
          if (item.name === egoMsg.pingYinName) {
            allEgoData = item.data
          }
        })
        if (!allEgoData) { // 如果没有当前地区的GeoJson数据，则重新请求接口获取
          const egoDataRes = await this.getEgoCode(egoMsg) // 获取GeoJson数据
          egoDataList.push({
            name: egoMsg.pingYinName,
            data: egoDataRes
          })
          allEgoData = egoDataRes
        }
        egoDataDetailListMsg = JSON.stringify(egoDataList)
      }
      localStorage.setItem('egoDataDetailListMsg', egoDataDetailListMsg)
      this.allEgoData = await this.setEgoData(allEgoData, egoMsg) // 设置GeoJson数据
    },
    setEgoData(egoData, egoMsg) { // 设置GeoJson数据
      if (egoData) {
        if (egoMsg.areaName === '中国') {
          const features = []
          egoData.features.forEach(item => {
            const trueName = item.properties.name // 真实地理位置名称
            item.properties.areaName = trueName
            if (trueName !== '') {
              if (trueName === '海南省') {
                item.geometry.coordinates = [item.geometry.coordinates[0]]
              }
              let showName = trueName.replace('省', '')
              showName = showName.replace('市', '')
              showName = showName.replace('特别行政区', '')
              showName = showName.replace('自治区', '')
              showName = showName.replace('壮族', '')
              showName = showName.replace('维吾尔', '')
              showName = showName.replace('回族', '')
              item.properties.name = showName
              item.properties.pingYinName = loadPinYinInit.ConvertPinyin(showName) // 汉字转拼音
              this.mapLevel = item.properties.level // 当前展示板块级别
              features.push(item)
            }
          })
          egoData.features = features
        } else {
          const features = []
          egoData.features.forEach(item => {
            const trueName = item.properties.name // 真实地理位置名称
            item.properties.areaName = trueName
            if (trueName !== '' && trueName !== '三沙市') {
              item.properties.pingYinName = loadPinYinInit.ConvertPinyin(trueName) // 汉字转拼音
              this.mapLevel = item.properties.level // 当前展示板块级别
              features.push(item)
            }
          })
          egoData.features = features
        }
      }
      return egoData
    },
    async getEgoCode(egoMsg) { // 获取GeoJson数据
      let jsonMsg = `${egoMsg.adcode}_full`
      for (const key in chinaRegions) {
        const item = chinaRegions[key]
        if (key === '海南省') {
          item.children.forEach(cityItem => {
            if (cityItem === egoMsg.areaName) {
              jsonMsg = egoMsg.adcode
            }
          })
        }
      }
      // ego查询Url地址 https://datav.aliyun.com/portal/school/atlas/area_selector
      const protocol = window.location.protocol // 当前网页地址的协议类型
      const res = await axios({
        method: 'get',
        url: `${protocol}//geo.datav.aliyun.com/areas_v3/bound/${jsonMsg}.json`
      })
      // 返回的数据 properties 中的 adcode 代表地区代码
      const egoDataRes = res.data
      return egoDataRes
    },
    setSeriesData() { // 设置图表数据
      const seriesData = []
      const areaItemsByKey = new Map()
      const addressMatchedAreaItems = []
      this.areaAddrMsgList.forEach(areaItem => {
        if (!areaItem) return
        const areaKey = areaItem.areaKey == null ? '' : String(areaItem.areaKey)
        if (areaKey) areaItemsByKey.set(areaKey, areaItem)
        if (areaKey && !/^\d+$/.test(areaKey)) addressMatchedAreaItems.push(areaItem)
      })
      if (this.allEgoData.features) {
        this.allEgoData.features.forEach(item => {
          seriesData.push({
            areaName: item.properties.areaName, // 真实名称
            name: item.properties.name, // 显示的名称
            pingYinName: item.properties.pingYinName, // 显示的名称的拼音
            adcode: item.properties.adcode, // 地区代码
            value: 0,
            level: item.properties.level
          })
        })
      }
      if (this.egoMsg.areaName === '中国') { // 全国
        seriesData.push({
          name: '南海诸岛',
          value: 0
        })
      }
      seriesData.forEach(item => {
        if (!item.areaName) {
          return
        }

        const matchRegionNames = [item.areaName]
        if (this.egoMsg.areaName === '中国') {
          const province = chinaRegions[item.areaName]
          if (province) {
            matchRegionNames.push(...province.children)
          }
        }

        const matchedCorp = findBestCorpByRegionName(
          this.corpTreeList,
          item.areaName,
          this.queryableCorpKeys
        )
        const matchedCorpKeys = matchedCorp
          ? new Set(expandCorpKeys(this.corpTreeList, matchedCorp.id))
          : new Set()
        const matchedAreaItemSet = new Set()
        matchedCorpKeys.forEach(areaKey => {
          const areaItem = areaItemsByKey.get(String(areaKey))
          if (areaItem) matchedAreaItemSet.add(areaItem)
        })
        // 未分配部门的站点没有部门范围，只按站点地址归入区域。
        addressMatchedAreaItems.forEach(areaItem => {
          if (matchRegionNames.some(regionName => regionNameMatchScore(areaItem.moreAreaName, regionName) > 0)) {
            matchedAreaItemSet.add(areaItem)
          }
        })

        matchedAreaItemSet.forEach(areaItem => {
          item.value += areaItem.pointList.length
        })
      })
      this.chartData.seriesData = seriesData
    },
    async enterProvince(params) { // 选择省份或城市
      if (params.value === 0) {
        this.$emit('levelChange', {
          level: this.level,
          areaName: this.areaName,
          value: params.value
        })
      } else {
        this.level = ''
        const data = params.data
        this.addrName = params.name
        this.areaName = data.areaName
        this.egoMsg = {
          areaName: data.areaName,
          pingYinName: data.pingYinName,
          adcode: data.adcode // 中国的地区代码
        }
        this.historyEgoMsgList.push(this.egoMsg)
        const addrMsg = judgeRegionType(data.areaName) // 判断一个地区是省还是市
        if (!addrMsg.success) {
          this.historyEgoMsgList.pop()
          return
        }
        await this.getEgoData() // 获取地图数据
        setTimeout(() => {
          this.level = addrMsg.type === '省' ? 'province' : 'city'
          this.setSeriesData() // 设置图表数据
          if (this.level === 'city') this.setMarkList()
          this.$emit('levelChange', {
            level: this.level,
            areaName: this.areaName
          })
        }, 200)
      }
    },
    async goBackAddr() { // 返回上级地理位置
      this.level = ''
      const parentRegion = lastudgeRegionType(this.areaName)
      let level = 'china'
      if (parentRegion.areaName) {
        this.areaName = parentRegion.areaName
        level = parentRegion.data.type
      }
      this.egoMsg = this.historyEgoMsgList[this.historyEgoMsgList.length - 2]
      await this.getEgoData() // 获取地图数据
      setTimeout(() => {
        this.level = level
        this.setSeriesData() // 设置图表数据
        if (this.level === 'city') this.setMarkList()
        this.historyEgoMsgList.pop()
        this.$emit('levelChange', {
          level: this.level,
          areaName: this.areaName
        })
      }, 200)
    },
    setMarkList() { // 设置详细地图标点
      const markerList = []
      this.areaAddrMsgList.forEach(item => {
        item.pointList.forEach(point => {
          markerList.push({
            position: [point.longitude, point.latitude],
            icon: point.icon, // 地图标点图标
            label: { offset: [-10, 33], content: `<div>${point.pointName}</div>` },
            infoWindowDom: point.infoWindowDom,
            getInfoWindowDom: point.getInfoWindowDom
          })
        })
      })
      this.markerList = markerList
    },
    goToDetail(id) {
      this.$emit('goToDetail', id)
    }
  }
}
</script>

<style lang="stylus" scoped>
  .buttonCss {
    position absolute
    left 20px
    top 20px
    z-index 1
  }
</style>

<style lang="stylus">
.mapPopupCss {
  width: 530px;
  padding: 10px;
  background: rgba(255, 255, 255, 0.2);
  backdrop-filter: blur(10px);
  box-sizing: border-box;
  border: 1px solid #FFFFFF;
  display: flex;
  position: relative;
  z-index 9999
  .mapPopupBodyCss {
    position relative
    margin-left: 10px;
    width: 100%;
    height: 100%;
    font-size: 16px;
    font-weight: normal;
    line-height: 22px;
    letter-spacing: 0px;
    color: #000;
    .mapPopupCloseCss {
      position absolute
      top 0
      right 10px
      font-size 20px
    }
    .mapPopupRowCss {
      margin-top 10px
      width 100%
      height 30px
      .mapPopupBatteryCss {
        float left
        margin-right 10px
        line-height 30px
      }
      .mapPopupMessageCss {
        float left
        margin-right 10px
        line-height 20px
      }
      .mapPopupIconCss {
        float left
        margin-right 10px
        width 30px
        height 30px
      }
      #mapPopupDetailsCss {
        float left
        padding: 5px 10px;
        font-size: 15px;
        cursor: pointer;
        text-align: center;
        text-decoration: none;
        outline: none;
        color: #fff;
        background-color: rgb(16, 185, 214);
        border: none;
        border-radius: 27px;
      }
    }
  }
}
</style>
