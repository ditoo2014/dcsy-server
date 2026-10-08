<template>

  <div class="detailMapCss" :style="{height,width}">

    <div :id="mapContainerId" :style="{height,width}" />

    <div class="searchCss">

      <el-input v-model="searchMsg" placeholder="请搜索..." @change="searchMsgChaneg" />

    </div>

  </div>

</template>

<script>

import { markRaw } from 'vue'
import { loadBusinessMap } from '@/utils/amap-loader'

let detailMapInstanceCounter = 0

export default {

  props: {

    width: {

      type: String,

      default: '500px'

    },

    height: {

      type: String,

      default: '500px'

    },

    addrName: {

      type: String,

      default: '成都'

    },

    markerList: {

      type: Array,

      default: () => {
        return []
      }

    },

    amapWindowInfo: {

      type: [Object, Boolean],

      default: () => {
        return false
      }

    }

  },

  data() {
    detailMapInstanceCounter += 1

    return {

      mapContainerId: `detail-map-container-${detailMapInstanceCounter}`,

      mapCenter: [104.06, 30.45],

      mapKey: null,
      mapDisposed: false,
      mapRequestId: 0,

      searchMsg: '',

      allMarkers: [],

      initTimer: null

    }
  },

  watch: {

    width() {
      this.resizeMap()
    },

    height() {
      this.resizeMap()
    },

    addrName() {
      this.recreateMap()
    },

    markerList: {
      handler() {
        this.replaceMarkers()
      }
    }

  },

  mounted() {
    this.bindWindowHandlers()

    this.initTimer = setTimeout(() => {
      this.initTimer = null

      this.createMarker()
    }, 1000)

    this.__resizeHandler = () => this.resizeMap()

    window.addEventListener('resize', this.__resizeHandler)
  },

  beforeUnmount() {
    this.mapDisposed = true
    window.removeEventListener('resize', this.__resizeHandler)

    if (this.initTimer) {
      clearTimeout(this.initTimer)

      this.initTimer = null
    }

    this.unbindWindowHandlers()

    if (this.mapKey) {
      this.mapKey.destroy()

      this.mapKey = null
    }
  },

  methods: {

    bindWindowHandlers() {
      window.goToDetail = this.goToDetail

      window.closeInfoWindow = this.closeAllInfoWindows
    },

    unbindWindowHandlers() {
      if (window.goToDetail === this.goToDetail) {
        delete window.goToDetail
      }

      if (window.closeInfoWindow === this.closeAllInfoWindows) {
        delete window.closeInfoWindow
      }
    },

    closeAllInfoWindows() {
      if (!this.mapKey) return

      this.mapKey.getAllOverlays('marker').forEach(m => {
        if (m.infoWindow) m.infoWindow.close()
      })
    },

    resizeMap() {
      if (this.mapKey && typeof this.mapKey.resize === 'function') {
        this.mapKey.resize()
      }
    },

    recreateMap() {
      this.mapRequestId += 1
      if (this.initTimer) {
        clearTimeout(this.initTimer)

        this.initTimer = null
      }

      if (this.mapKey) {
        this.mapKey.destroy()

        this.mapKey = null
      }

      this.initTimer = setTimeout(() => {
        this.initTimer = null

        this.createMarker()
      }, 300)
    },

    async createMarker() {
      // 快速切换地区时，旧查询的回调不能重新创建旧地图。
      const requestId = ++this.mapRequestId
      try {
        await loadBusinessMap()
      } catch (error) {
        if (!this.mapDisposed && requestId === this.mapRequestId) this.$message.error(error.message)
        return
      }
      if (this.mapDisposed || requestId !== this.mapRequestId) return
      const opts = {

        subdistrict: 0,

        extensions: 'all',

        level: 'district' // 地区级别 查询地址：https://lbs.amap.com/demo/javascript-api/example/district-search/draw-district-boundaries

      }

      // 利用行政区查询获取边界构建mask路径

      // 也可以直接通过经纬度构建mask路径

      const district = new AMap.DistrictSearch(opts)

      district.search(this.addrName, (status, result) => {
        if (this.mapDisposed || requestId !== this.mapRequestId) return
        if (status === 'complete' && result?.districtList?.length) {
          const bounds = result.districtList[0].boundaries || []

          const center = result.districtList[0].center

          const map = new AMap.Map(this.mapContainerId, {

            center: [center.lng, center.lat], // 设置中心点

            disableSocket: true,

            terrain: false, // 开启地形图

            showLabel: false, // 是否显示文字

            labelzIndex: 130,

            pitch: 0 // 3D视图倾斜度

          })

          /* 1. 把边界画成线 */

          bounds.forEach(path => {
            new AMap.Polyline({

              path,

              strokeColor: '#99ffff',

              strokeWeight: 4,

              map

            })
          })

          /* 3. 自适应视野 */

          map.setFitView()

          this.mapKey = markRaw(map)

          this.replaceMarkers(map)
        } else {
          this.$message.error('地图区域加载失败，请检查地区名称后重试')
        }
      })
    },

    replaceMarkers(map = this.mapKey) {
      if (!map) return

      const openedMarker = this.allMarkers.find(marker => {
        return marker.infoWindow && typeof marker.infoWindow.getIsOpen === 'function' && marker.infoWindow.getIsOpen()
      })
      const openedMarkerLabel = openedMarker && openedMarker.getLabel()
        ? openedMarker.getLabel().content
        : ''
      this.closeAllInfoWindows()
      if (this.allMarkers.length > 0) {
        map.remove(this.allMarkers)
      }

      const allMarkers = []
      this.markerList.forEach(markerMsg => {
        const longitude = markerMsg.position[0]
        const latitude = markerMsg.position[1]
        if (!longitude || !latitude) return

        const icon = markerMsg.icon
        const offset = markerMsg.label.offset
        const content = markerMsg.label.content
        const infoWindowDom = markerMsg.infoWindowDom
        const getInfoWindowDom = markerMsg.getInfoWindowDom
        const marker = new AMap.Marker({
          position: new AMap.LngLat(longitude, latitude),
          icon,
          label: {
            offset: new AMap.Pixel(offset[0], offset[1]),
            content
          }
        })
        marker.infoWindow = null
        const openInfoWindow = () => {
          if (!marker.infoWindow) {
            marker.infoWindow = new AMap.InfoWindow({
              isCustom: true,
              content: typeof getInfoWindowDom === 'function' ? getInfoWindowDom() : infoWindowDom,
              offset: new AMap.Pixel(0, -45)
            })
          }
          marker.infoWindow.open(map, marker.getPosition())
        }
        marker.on('click', openInfoWindow)
        map.add(marker)
        allMarkers.push(marker)
        if (openedMarkerLabel && content === openedMarkerLabel) {
          openInfoWindow()
        }
      })
      this.allMarkers = markRaw(allMarkers)
      if ((this.searchMsg || '').trim()) {
        this.searchMsgChaneg()
      }
    },

    searchMsgChaneg() { // 搜索框内容变化时触发
      if (!this.mapKey || this.mapDisposed) return
      const kw = (this.searchMsg || '').trim().toLowerCase()

      const visibleMarkers = [] // 收集本次要显示的 marker

      this.allMarkers.forEach(m => {
        const labelTxt = (m.getLabel() && m.getLabel().content) || ''

        const hit = !kw || labelTxt.toLowerCase().includes(kw)

        hit ? m.show() : m.hide()

        if (hit) visibleMarkers.push(m) // 只把可见的留下来
      })

      /* 关键：让地图重新聚焦到可见 marker */

      if (visibleMarkers.length) {
        this.mapKey.setFitView(visibleMarkers, false, [20, 20, 20, 20], 12) // 第2、3参数可微调边距
      }
    },

    goToDetail(id) { // 进入详情页
      this.$emit('goToDetail', id)
    }

  }

}

</script>

<style lang="stylus" scoped>

.detailMapCss {

  position relative

  .searchCss {

    position absolute

    top 20px

    left 130px

    z-index 1

  }

}

</style>

