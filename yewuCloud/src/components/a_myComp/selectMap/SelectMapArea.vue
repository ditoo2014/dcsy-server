<template>
  <div v-loading="mapLoading" class="mapCss">
    <div id="container" class="containerCss" />

    <div ref="mapInfoWindowRef" class="mapInfoWindowContentCss">
      <div class="lngAndLatCss">
        <div class="lngAndLatContentCss">
          <el-input v-model="params.lng" placeholder="请输入经度" />
        </div>
        <div class="lngAndLatContentCss">
          <el-input v-model="params.lat" placeholder="请输入纬度" />
        </div>
        <div class="lngAndLatButtonCss">
          <el-button @click="searchAddrMsg">查询</el-button>
        </div>
      </div>
      <div class="inputAndButCss">
        <div class="inputCss">
          <el-input v-model="searchAddr" placeholder="请输入地理位置信息" @keyup.enter="searchSite" />
        </div>
        <div class="buttonCss">
          <el-button @click="searchSite">确认地址</el-button>
        </div>
      </div>
      <div id="panel" class="panelCss" />
    </div>
    <div class="footerCss">
      <el-button type="primary" @click="mapInfo">确定</el-button>
    </div>
  </div>
</template>

<script>
import { markRaw } from 'vue'
import { loadBusinessMap } from '@/utils/amap-loader'

export default {
  props: {
    mapMsg: {
      type: Object,
      default: () => {
        return {}
      }
    }
  },
  data() {
    return {
      mapKey: null, // 地图标志
      mapCenter: [104.06, 30.45], // 地图中心坐标
      mapLoading: false,
      mapDisposed: false,
      mapInfoWindow: null, // 信息窗体

      searchAddr: '',
      marker: null, // 标记点
      params: {
        lng: '', // 经度
        lat: '', // 纬度
        provinceName: '', // 省份名称
        cityName: '', // 城市名称
        resultName: '' // 详细地址
      }
    }
  },
  mounted() {
    this.createMarker()
  },
  beforeUnmount() {
    this.mapDisposed = true
    this.mapInfoWindow?.close()
    this.mapKey?.destroy()
    this.mapKey = null
  },
  methods: {
    async createMarker() {
      if (this.mapMsg.msg) {
        this.params = { ...this.mapMsg.msg }
        this.mapCenter[0] = this.params.lng
        this.mapCenter[1] = this.params.lat
      }
      this.mapLoading = true
      try {
        await loadBusinessMap()
      } catch (error) {
        if (!this.mapDisposed) {
          this.mapLoading = false
          this.$message.error(error.message)
        }
        return
      }
      if (this.mapDisposed) return

      const opts = {
        subdistrict: 0,
        extensions: 'all',
        level: 'district'
        /*
          地区级别 查询地址：https://lbs.amap.com/demo/javascript-api/example/district-search/draw-district-boundaries
          district: 区
          city: 市
          province: 省
        */
      }
      // 利用行政区查询获取边界构建mask路径
      // 也可以直接通过经纬度构建mask路径
      const district = new AMap.DistrictSearch(opts)
      district.search('成都', (status, result) => {
        if (this.mapDisposed) return
        if (status !== 'complete' || !result?.districtList?.length) {
          this.mapLoading = false
          this.$message.error('地图区域加载失败，请稍后重试')
          return
        }
        const bounds = result.districtList[0].boundaries || []
        const mask = []
        for (let i = 0; i < bounds.length; i++) {
          mask.push([bounds[i]])
        }
        const map = new AMap.Map('container', {
          // mask, // 地图掩膜，用于将查询以外部分隐藏
          center: this.mapCenter,
          disableSocket: true,
          // viewMode: '3D', // 开启3D视图,默认为关闭
          terrain: false, // 开启地形图
          showLabel: true, // 是否显示文字
          labelzIndex: 130,
          pitch: 0, // 3D视图倾斜度
          zoom: 9,
          mapStyle: 'amap://styles/blue', // 背景颜色: https://blog.csdn.net/weixin_41767649/article/details/99838052
          layers: [
            new AMap.TileLayer.RoadNet({
              // rejectMapMask:true
            }),
            new AMap.TileLayer.Satellite() // 地图掩膜
          ]
        })

        for (let i = 0; i < bounds.length; i++) { // 添加描边
          new AMap.Polyline({
            path: bounds[i],
            strokeColor: '#99ffff',
            strokeWeight: 4,
            map: map
          })
        }

        map.on('click', (ev) => {
          const lnglat = ev.lnglat
          this.getAddrMsg([lnglat.lng, lnglat.lat])
        })

        this.mapKey = markRaw(map)
        this.mapLoading = false
      })

      const staticContent = this.$refs.mapInfoWindowRef.cloneNode(true)
      this.mapInfoWindow = new AMap.InfoWindow({
        isCustom: true,
        closeWhenClickMap: false,
        content: staticContent, // 交出去的是克隆体
        offset: new AMap.Pixel(0, -28)
      })
    },
    searchAddrMsg() {
      if (!this.ensureMapReady()) return
      const lng = Number(this.params.lng)
      const lat = Number(this.params.lat)
      if (this.params.lng === '' || this.params.lat === '' || !Number.isFinite(lng) || !Number.isFinite(lat) || Math.abs(lng) > 180 || Math.abs(lat) > 90) {
        this.$message.error('请输入有效的经纬度')
        return
      }
      const This = this
      const params = this.params
      this.mapKey.plugin('AMap.Geocoder', function() {
        const geocoder = new AMap.Geocoder({
          // city 指定进行编码查询的城市，支持传入城市名、adcode 和 citycode
          city: ''
        })
        const lnglat = [params.lng, params.lat]
        geocoder.getAddress(lnglat, (status, result) => {
          if (This.mapDisposed) return
          if (status === 'complete' && result?.regeocode) {
            params.provinceName = result.regeocode.addressComponent.province // 省份名称
            params.cityName = result.regeocode.addressComponent.city || params.provinceName // 直辖市可能没有城市字段
            params.resultName = result.regeocode.formattedAddress // 详细地址
            This.searchAddr = params.resultName
            This.addMarker(lnglat)
          } else {
            This.$message.error('地址查询失败，请检查经纬度后重试')
          }
        })
      })
    },
    searchSite() {
      if (!this.ensureMapReady()) return
      const that = this
      const placeSearch = new AMap.PlaceSearch({ // 构造地点查询类
        pageSize: 5,
        pageIndex: 1,
        city: '', // 城市
        map: that.mapKey,
        panel: 'panel' // 查询组件的id
      })
      placeSearch.search(that.searchAddr, (status, result) => { // 关键字查询
        if (this.mapDisposed) return
        if (status !== 'complete' || !result?.poiList?.pois?.length) {
          this.$message.error('未查询到该地址，请修改关键词后重试')
          return
        }
        if (result.info === 'OK') {
          const pois = result.poiList.pois
          if (pois.length !== 0) {
            const poi = pois[0]
            const location = poi.location
            let resultName = ''
            if (poi.adname === poi.name) {
              if (poi.name === poi.address) {
                resultName = poi.address
              } else {
                resultName = `${poi.name}，${poi.address}`
              }
            } else {
              if (poi.name === poi.address) {
                resultName = `${poi.adname}，${poi.name}`
              } else {
                resultName = `${poi.adname}，${poi.name}，${poi.address}`
              }
            }
            this.params = {
              lng: location.lng, // 经度
              lat: location.lat, // 纬度
              provinceName: poi.pname, // 省份名称
              cityName: poi.cityname, // 城市名称
              resultName // 详细地址
            }
          }
        }
        placeSearch.on('markerClick', (e) => {
          if (this.mapDisposed) return
          const poi = e.data
          const location = poi.location
          let resultName = ''
          if (poi.adname === poi.name) {
            if (poi.name === poi.address) {
              resultName = poi.address
            } else {
              resultName = `${poi.name}，${poi.address}`
            }
          } else {
            if (poi.name === poi.address) {
              resultName = `${poi.adname}，${poi.name}`
            } else {
              resultName = `${poi.adname}，${poi.name}，${poi.address}`
            }
          }
          this.params = {
            lng: location.lng, // 经度
            lat: location.lat, // 纬度
            provinceName: poi.pname, // 省份名称
            cityName: poi.cityname, // 城市名称
            resultName // 详细地址
          }
        })
        placeSearch.on('listElementClick', (e) => { // 点击panel列表的点标记
          if (this.mapDisposed) return
          const poi = e.data
          const location = poi.location
          let resultName = ''
          if (poi.adname === poi.name) {
            if (poi.name === poi.address) {
              resultName = poi.address
            } else {
              resultName = `${poi.name}，${poi.address}`
            }
          } else {
            if (poi.name === poi.address) {
              resultName = `${poi.adname}，${poi.name}`
            } else {
              resultName = `${poi.adname}，${poi.name}，${poi.address}`
            }
          }
          this.params = {
            lng: location.lng, // 经度
            lat: location.lat, // 纬度
            provinceName: poi.pname, // 省份名称
            cityName: poi.cityname, // 城市名称
            resultName // 详细地址
          }
        })
      })
    },
    getAddrMsg(lnglat) {
      if (!this.ensureMapReady()) return
      const geocoder = new AMap.Geocoder({})
      geocoder.getAddress(lnglat, (status, result) => {
        if (this.mapDisposed) return
        if (status !== 'complete' || !result?.regeocode) {
          this.$message.error('地址查询失败，请稍后重试')
          return
        }
        const regeocode = result.regeocode
        this.searchAddr = regeocode.formattedAddress
        this.params = {
          lng: lnglat[0], // 经度
          lat: lnglat[1], // 纬度
          provinceName: regeocode.addressComponent.province, // 省份名称
          cityName: regeocode.addressComponent.city || regeocode.addressComponent.province, // 城市名称
          resultName: regeocode.formattedAddress // 详细地址
        }
        this.addMarker(lnglat)
      })
    },
    addMarker(lnglat) { // 添加标记点
      if (!this.mapKey || this.mapDisposed) return
      this.clearMarker()
      this.marker = new AMap.Marker({
        position: lnglat,
        title: this.searchAddr
      })
      this.marker.setMap(this.mapKey)
    },
    clearMarker() { // 清除 marker
      if (this.marker) {
        this.marker.setMap(null)
        this.marker = null
      }
    },
    ensureMapReady() {
      // 地图脚本异步加载，加载完成前点击查询不应触发空对象异常。
      if (this.mapKey && !this.mapDisposed) return true
      if (!this.mapDisposed) this.$message.warning('地图尚未加载完成，请稍后重试')
      return false
    },
    mapInfo() { // 确定选择地图
      if (this.params.lng === '') {
        this.$message({
          message: '请点击地图选择具体地址',
          type: 'error',
          duration: 1000
        })
      } else {
        this.$emit('map', this.params)
      }
    }
  }
}
</script>

<style lang="stylus" scoped>
.mapCss {
  position relative
  width 100%
  height 100%
  .containerCss {
    width 100%
    height 100%
  }
  .mapInfoWindowContentCss {
    position absolute
    width 350px
    height 40%
    top 5%
    right 1%
    .lngAndLatCss {
      width 100%
      height 40px
      .lngAndLatContentCss {
        float left
        width 135px
      }
      .lngAndLatButtonCss {
        float left
        margin-left 5px
      }
    }
    .inputAndButCss {
      width 100%
      height 40px
      .inputCss {
        float left
        margin-right 1%
        width 70%
      }
      .buttonCss {
        float left
      }
    }

    .panelCss {
      width 100%
    }
  }
  .footerCss {
    position absolute
    right 20px
    bottom 20px
  }
}
</style>
