<template>
  <div :id="mapContainerId" v-loading="mapLoading" class="mapCss" />
</template>

<script>
import { markRaw } from 'vue'
import { loadBusinessMap } from '@/utils/amap-loader'
import { createContainerResizeWatcher } from '@/utils/container-resize.js'

let mapDrivingInstanceCounter = 0

// 电池流转位置地图组件
export default {
  props: {
    mapMsg: {
      type: Object,
      default: () => ({})
    }
  },
  data() {
    mapDrivingInstanceCounter += 1
    return {
      mapLoading: false,
      mapDisposed: false,
      mapContainerId: `map-driving-container-${mapDrivingInstanceCounter}`,
      map: null,
      markers: [],
      polyline: null,
      infoWindows: [],
      resizeTimer: null
    }
  },
  watch: {
    mapMsg: {
      deep: true,
      handler() {
        this.drawPositions()
      }
    }
  },
  mounted() {
    // 流转地图跟随弹窗实际大小，不再依赖固定延时判断容器是否就绪。
    this.__mapSizeWatcher = createContainerResizeWatcher(
      () => this.$el,
      () => {
        if (!this.map || typeof this.map.resize !== 'function') return false
        this.map.resize()
        return true
      }
    )
    this.__mapSizeWatcher.start()
    this.$nextTick(() => this.initMap())
  },
  activated() {
    this.__mapSizeWatcher && this.__mapSizeWatcher.start()
  },
  deactivated() {
    this.__mapSizeWatcher && this.__mapSizeWatcher.stop()
  },
  beforeUnmount() {
    this.mapDisposed = true
    this.__mapSizeWatcher && this.__mapSizeWatcher.stop()
    if (this.resizeTimer) {
      clearTimeout(this.resizeTimer)
      this.resizeTimer = null
    }
    this.clearOverlays()
    if (this.map) {
      this.map.destroy()
      this.map = null
    }
  },
  methods: {
    async initMap() {
      this.mapLoading = true
      try {
        await loadBusinessMap()
        if (this.mapDisposed) return
        this.map = markRaw(new AMap.Map(this.mapContainerId, {
          center: [104.06, 30.45],
          zoom: 12,
          disableSocket: true,
          terrain: false,
          showLabel: true,
          labelzIndex: 130,
          pitch: 0
        }))
        this.__mapSizeWatcher && this.__mapSizeWatcher.schedule()
        this.drawPositions()
        this.resizeTimer = setTimeout(() => {
          this.resizeTimer = null
          if (this.map) {
            this.__mapSizeWatcher && this.__mapSizeWatcher.schedule()
            this.drawPositions()
          }
        }, 300)
      } catch (error) {
        if (!this.mapDisposed) this.$message.error(error.message)
      } finally {
        if (!this.mapDisposed) this.mapLoading = false
      }
    },
    escapeHtml(value) {
      return String(value == null ? '' : value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;')
    },
    pointStyle(pointType) {
      if (pointType === 'start') {
        return { color: '#00c853', border: '#008f3a', name: '起点' }
      }
      if (pointType === 'end') {
        return { color: '#f44336', border: '#c62828', name: '终点' }
      }
      return { color: '#ffffff', border: '#606266', name: '中间点' }
    },
    buildInfoWindowContent(item) {
      const address = item.locate || item.storeroomAddress || '无'
      return `<div style="min-width:260px;line-height:24px;padding:4px 2px;">
        <div style="font-weight:600;margin-bottom:4px;">${this.escapeHtml(item.pointType === 'end' ? '最后出现位置' : '流转位置')}</div>
        <div>位置：${this.escapeHtml(address)}</div>
        <div>库房/站点：${this.escapeHtml(item.storeroomAddress || '无')}</div>
        <div>状态：${this.escapeHtml(item.operateTypeName || '未知状态')}</div>
        <div>操作时间：${this.escapeHtml(item.createTime || '无')}</div>
        <div>责任人：${this.escapeHtml(item.commander || '无')}</div>
        <div>电话：${this.escapeHtml(item.phone || '无')}</div>
      </div>`
    },
    clearOverlays() {
      if (!this.map) return
      this.infoWindows.forEach(infoWindow => infoWindow.close())
      this.infoWindows = []
      if (this.markers.length > 0) {
        this.map.remove(this.markers)
        this.markers = []
      }
      if (this.polyline) {
        this.map.remove(this.polyline)
        this.polyline = null
      }
    },
    drawPositions() {
      if (!this.map) return
      this.clearOverlays()
      const pointList = Array.isArray(this.mapMsg && this.mapMsg.pointList)
        ? this.mapMsg.pointList
        : []
      if (pointList.length === 0) return

      const validPoints = pointList.filter(item => {
        const longitude = Number.parseFloat(item.longitude)
        const latitude = Number.parseFloat(item.latitude)
        return Number.isFinite(longitude) && Number.isFinite(latitude)
      })
      if (validPoints.length === 0) return

      const path = validPoints.map(item => [
        Number.parseFloat(item.longitude),
        Number.parseFloat(item.latitude)
      ])
      if (path.length >= 2) {
        this.polyline = markRaw(new AMap.Polyline({
          path,
          strokeColor: '#409eff',
          strokeWeight: 5,
          strokeOpacity: 0.9,
          showDir: true,
          zIndex: 100
        }))
        this.map.add(this.polyline)
      }

      validPoints.forEach(item => {
        try {
          const position = new AMap.LngLat(
            Number.parseFloat(item.longitude),
            Number.parseFloat(item.latitude)
          )
          const style = this.pointStyle(item.pointType)
          const address = item.locate || item.storeroomAddress || '无'
          const marker = markRaw(new AMap.Marker({
            position,
            content: `<div title="${this.escapeHtml(style.name)}" style="position:relative;width:28px;height:34px;">
              <div style="position:absolute;left:4px;top:3px;width:20px;height:20px;border:3px solid ${style.border};border-radius:50% 50% 50% 0;background:${style.color};box-sizing:border-box;box-shadow:-2px 2px 7px rgba(0,0,0,.4);transform:rotate(-45deg);">
                <div style="position:absolute;left:5px;top:5px;width:4px;height:4px;border-radius:50%;background:${item.pointType === 'middle' ? '#606266' : '#fff'};"></div>
              </div>
            </div>`,
            anchor: 'bottom-center',
            title: address,
            zIndex: item.pointType === 'end' ? 200 : 150,
            label: {
              direction: 'top',
              offset: new AMap.Pixel(0, -5),
              content: `<div style="white-space:nowrap;padding:4px 8px;border:1px solid ${style.border};border-radius:4px;background:#fff;color:#303133;box-shadow:0 2px 8px rgba(0,0,0,.18);">${this.escapeHtml(address)}</div>`
            }
          }))
          const infoWindow = markRaw(new AMap.InfoWindow({
            content: this.buildInfoWindowContent(item),
            offset: new AMap.Pixel(0, -34),
            autoMove: false
          }))
          marker.on('click', () => {
            this.infoWindows.forEach(window => window.close())
            infoWindow.open(this.map, marker.getPosition())
          })
          marker.setMap(this.map)
          this.markers.push(marker)
          this.infoWindows.push(infoWindow)
        } catch (error) {
          console.error('[MapDriving] 创建流转位置标记失败', item, error)
        }
      })

      if (this.markers.length === 0) return
      if (this.markers.length === 1) {
        const position = this.markers[0].getPosition()
        this.map.setZoomAndCenter(15, [position.lng, position.lat])
      } else {
        const overlays = this.polyline ? [this.polyline, ...this.markers] : this.markers
        this.map.setFitView(overlays, false, [60, 60, 60, 60], 16)
      }

      const finalIndex = this.markers.length - 1
      this.infoWindows[finalIndex].open(this.map, this.markers[finalIndex].getPosition())
    }
  }
}
</script>

<style lang="stylus" scoped>
.mapCss {
  position relative
  width 100%
  height 100%
}
</style>
