<template>
  <div class="bannerCss">
    <div v-if="logoUrl !== ''" class="logoCss">
      <img :src="logoUrl">
    </div>
    <div v-if="logoName !== ''" class="logoNameCss">
      {{ logoName }}
    </div>
    <div class="groupOneCss">
      <img :src="groupOneImage">
    </div>
    <div class="bannerButContentCss">
      <div v-for="(item,index) in platformList" :key="index">
        <div v-if="item.isShow" class="numeContentCss" :style="{ width: platformWidth }">
          <div
            class="bannerButCss cursorPointer"
            :class="[item.index === activeIndex ? 'activeCss' : '']"
            @click="goRoute(item.index)"
          >
            {{ item.name }}
          </div>
          <div v-if="item.warningTotal" class="warningTotalCss">
            <el-tag type="danger" effect="dark" size="small">{{ item.warningTotal }}</el-tag>
          </div>
        </div>
      </div>
    </div>
    <div class="groupTwoCss">
      <img :src="groupTwoImage">
    </div>
    <!-- <div class="loginCjyCss cursorPointer" @click="logoCjy">登录采集云</div> -->
    <el-popover
      placement="bottom"
      width="150"
      trigger="click"
    >
      <div class="funcSelectCss cursorPointer">
        <div class="borderBottomCss" @click="modifyPas">修改密码</div>
        <div @click="logout">退出登录</div>
      </div>
      <template #reference>
        <div class="userOperationCss cursorPointer">
          <div class="userPortraitCss">
            <img :src="headImg">
          </div>
          <div class="userNameCss">{{ userName }}</div>
          <div class="selectIconCss" />
        </div>
      </template>
    </el-popover>
    <div class="screenfullCss cursorPointer" @click="screenfullClick">
      <img :src="screenfullImage">
    </div>
    <audio id="vd" loop>
      <source src="./warning.mp3" type="audio/mpeg">
      您的浏览器不支持 audio 元素。
    </audio>

  </div>
</template>

<script>
import config from '@/config.js'
import { PLATFORM } from '@/constants/platform'
import { mapGetters } from '@/store/pinia-helpers'
import screenfull from 'screenfull'
import { requestJscFullscreen, toggleJscFullscreen } from '@/utils/fullscreen.js'
import { getPageList, getUsername } from '@/utils/auth.js'
import { findCollectLiveTriggerTempData } from '@/api/jurisdCenter.js'
import { getBannerHomePath, resolveBannerIdFromPath } from '@/utils/banner-route.js'
import { beginBannerNav, endBannerNav, isBannerNavLocked } from '@/utils/banner-nav-lock.js'
import { expireSession } from '@/utils/session-expiration'
import groupOneImage from './imgs/group1.png'
import groupTwoImage from './imgs/group2.png'
import screenfullImage from './imgs/screenfull.png'
import dcsyLogoImage from './imgs/dcsyLogo.png'

let music = null // 是否正在播放告警

export default {
  props: {
    bannerId: {
      type: [Number, String],
      default: '0'
    }
  },
  data() {
    return {
      groupOneImage,
      groupTwoImage,
      screenfullImage,
      isDevelopment: import.meta.env.DEV, // 是否是开发者模式

      logoUrl: '', // logo图片
      logoName: '', // logo名称
      headImg: '', // 头像图片

      userName: getUsername(),

      activeIndex: '1',
      platformList: [
        { index: '1', name: '大屏中心', routeName: '', path: '', isShow: false },
        { index: '3', name: '监控平台', routeName: '', path: '', isShow: false },
        { index: '4', name: '告警中心', routeName: '', path: '', isShow: false, warningTotal: 0 },
        { index: '5', name: '运维中心', routeName: '', path: '', isShow: false },
        { index: '7', name: '报表中心', routeName: '', path: '', isShow: false },
        { index: '6', name: '系统管理', routeName: '', path: '', isShow: false },
        { index: '10', name: '图标库', routeName: '', path: '', isShow: false },
        { index: '0', name: '采集云平台', routeName: '', path: '', isShow: false },
        { index: '11', name: '驾驶舱', routeName: '', path: '', isShow: false },
        { index: '12', name: '全生命周期管理', routeName: '', path: '', isShow: false },
        { index: '13', name: '电池健康管理', routeName: '', path: '', isShow: false },
        { index: '14', name: '基础信息管理', routeName: '', path: '', isShow: false },
        { index: 'test', name: '测试', routeName: '', path: '', isShow: false }
      ],

      baoBiaoPageRoutes: [], // 报表中心页面路由
      jueCePageRoutes: [], // 监控中心页面路由
      jurisdCenterRoutes: [], // 告警中心页面路由
      dataManaPageRoutes: [], // 系统管理页面路由
      ywptPageRoutes: [], // 运维中心页面路由

      isShowYunwei: false, // 是否显示运维中心

      WarningTotal: 0 // 告警信息条数
    }
  },
  computed: {
    ...mapGetters([
      'avatar',
      'device',
      'permission_routes'
    ]),
    platformWidth() {
      let platformNum = 0
      this.platformList.forEach(item => {
        if (item.isShow) {
          platformNum++
        }
      })
      return platformNum > 0 ? `${100 / platformNum}%` : '100%'
    }
  },
  watch: {
    bannerId(val) {
      this.activeIndex = val
    }
  },
  mounted() {
    if (this.isDevelopment) { // 本地运行
      this.platformList.forEach(item => {
        if (item.name === '采集云平台' ||
          item.name === '图标库' ||
          item.name === '测试') {
          item.isShow = true
        }
      })
    }
    if (this.userName === 'ditooadmin') { // 最高权限账号
      this.platformList.forEach(item => {
        if (item.name === '测试') {
          item.isShow = true
        }
      })
    }
    this.setRouters()
    this.setBannerId()

    this._routerPathTimer = setInterval(() => {
      const routerPath = localStorage.getItem('routerPath')
      if (routerPath) {
        localStorage.removeItem('routerPath')
        this.setBannerId(routerPath)
      }
    }, 1000)

    // this.findWarningTempData() // 查询告警信息
    // setInterval(() => {
    //   this.findWarningTempData() // 查询告警信息
    // }, 30000)
    this.init()
  },
  beforeUnmount() {
    if (this._routerPathTimer) {
      clearInterval(this._routerPathTimer)
      this._routerPathTimer = null
    }
  },
  methods: {
    /** 当前 URL 已属于 activeIndex 对应模块时，Layout 重建无需再 push */
    isCurrentRouteInBanner() {
      const routeBannerId = resolveBannerIdFromPath(this.$route.path)
      return Boolean(routeBannerId && String(routeBannerId) === String(this.activeIndex))
    },
    /** 已在目标路由或其子路由时不再 push，避免生产环境重复导航导致栈溢出 */
    pushRouteIfNeeded(path) {
      if (!path) {
        return Promise.resolve()
      }
      const target = String(path)
      const current = this.$route.path
      if (current === target || current.startsWith(`${target}/`)) {
        return Promise.resolve()
      }
      return this.$router.push({ path: target })
    },
    setRouters() { // 设置路由
      const baoBiaoPageRoutes = []
      const jueCePageRoutes = []
      const jurisdCenterRoutes = []
      const dataManaPageRoutes = []
      const ywptPageRoutes = []
      this.permission_routes.forEach(route => {
        if (route.baoBiao) { // 报表中心
          baoBiaoPageRoutes.push(route)
        }
        if (route.jianKong) { // 监控中心
          jueCePageRoutes.push(route)
        }
        if (route.baojing) { // 告警中心
          jurisdCenterRoutes.push(route)
        }
        if (route.dataMana) { // 系统管理
          dataManaPageRoutes.push(route)
        }
        if (route.yunWei) { // 运维中心
          ywptPageRoutes.push(route)
        }
      })
      this.baoBiaoPageRoutes = baoBiaoPageRoutes
      this.jueCePageRoutes = jueCePageRoutes
      this.jurisdCenterRoutes = jurisdCenterRoutes
      this.dataManaPageRoutes = dataManaPageRoutes
      this.ywptPageRoutes = ywptPageRoutes
    },
    setBannerId(routerPath) {
      this.activeIndex = localStorage.getItem('bannerId')
      const pageList = JSON.parse(getPageList())
      if (pageList[0] === 1) { // admin 权限
        this.platformList.forEach(item => {
          if (item.name !== '图标库' &&
            item.name !== '采集云平台' &&
            item.name !== '测试') {
            item.isShow = true
          }
        })
      } else {
        pageList.forEach(page => {
          this.platformList.forEach(platform => {
            if (page.permissionName === platform.name) {
              platform.isShow = true
            }
            if (page.dataMana && platform.name === '全生命周期管理') {
              platform.isShow = true
            }
            if (page.yunWei && platform.name === '电池健康管理') {
              platform.isShow = true
            }
            if (page.jianKong && platform.name === '基础信息管理') {
              platform.isShow = true
            }
            if (page.baojing && platform.name === '驾驶舱') {
              platform.isShow = true
            }
          })
        })
      }

      if (config.platform === PLATFORM.DCSY_INTRANET) {
        if (config.port === '80') { // 锂硫电池溯源平台
          this.platformList.forEach(item => {
            switch (item.name) {
              case '大屏中心':
              case '监控平台':
              case '告警中心':
              case '运维中心':
              case '报表中心':
              case '系统管理':
              case '图标库':
              case '采集云平台':
              case '测试':
                item.isShow = false
                break
            }
          })
        } else if (config.port === '5002') {
          this.platformList.forEach(item => {
            switch (item.name) {
              case '大屏中心':
              case '驾驶舱':
              case '全生命周期管理':
              case '电池健康管理':
              case '基础信息管理':
                item.isShow = false
                break
            }
          })
        }
      } else if (config.platform === PLATFORM.DCSY_XC) { // 信创
        this.platformList.forEach(item => {
          switch (item.name) {
            case '大屏中心':
            case '监控平台':
            case '告警中心':
            case '运维中心':
            case '报表中心':
            case '系统管理':
            case '图标库':
            case '采集云平台':
            case '测试':
              item.isShow = false
              break
          }
        })
      }

      let isSetedIndex = false // index是否已设置完成
      this.platformList.forEach(item => {
        if (item.index === this.activeIndex && item.isShow) {
          isSetedIndex = true
        }
      })
      if (!isSetedIndex) {
        for (let i = 0; i < this.platformList.length; i++) {
          const item = this.platformList[i]
          if (item.isShow) {
            this.activeIndex = item.index
            break
          }
        }
      }
      const skipDcsyAutoNav = !routerPath && (this.isCurrentRouteInBanner() || isBannerNavLocked())
      if (config.platform === PLATFORM.DCSY_INTRANET) {
        if (config.port === '8003' || config.port === '80') {
          if (this.activeIndex !== '11' &&
            this.activeIndex !== '12' &&
            this.activeIndex !== '13' &&
            this.activeIndex !== '14'
          ) {
            this.activeIndex = '14'
          }
          if (!skipDcsyAutoNav) {
            if (routerPath) {
              this.pushRouteIfNeeded(routerPath)
            } else {
              const homePath = getBannerHomePath(this.activeIndex)
              if (homePath) {
                this.pushRouteIfNeeded(homePath)
              }
            }
          }
        }
      } else if (config.platform === PLATFORM.DCSY_XC) { // 信创
        if (this.activeIndex !== '11' &&
            this.activeIndex !== '12' &&
            this.activeIndex !== '13' &&
            this.activeIndex !== '14'
        ) {
          this.activeIndex = '14'
        }
        if (!skipDcsyAutoNav) {
          if (routerPath) {
            this.pushRouteIfNeeded(routerPath)
          } else {
            const homePath = getBannerHomePath(this.activeIndex)
            if (homePath) {
              this.pushRouteIfNeeded(homePath)
            }
          }
        }
      }
      if (!skipDcsyAutoNav) {
        switch (this.activeIndex) {
          case '1': break
          case '3': this.pushRouteIfNeeded(this.jueCePageRoutes[0].children[0].path); break
          case '4': this.pushRouteIfNeeded(this.jurisdCenterRoutes[0].children[0].path); break
          case '5': this.pushRouteIfNeeded(this.ywptPageRoutes[0].children[0].path); break
          case '7': this.pushRouteIfNeeded(this.baoBiaoPageRoutes[0].children[0].path); break
          case '6': this.pushRouteIfNeeded(this.dataManaPageRoutes[0].children[0].path); break
        }
      }
      localStorage.setItem('bannerId', this.activeIndex)
    },
    goRoute(activeIndex) {
      if (this._bannerNavigating) {
        return
      }
      if (activeIndex !== '0') {
        this.activeIndex = activeIndex
      }
      let targetPath = null
      switch (activeIndex) {
        case '1':
          targetPath = this.dataManaPageRoutes[0] && this.dataManaPageRoutes[0].children[0].path
          break
        case '3':
          targetPath = this.jueCePageRoutes[0] && this.jueCePageRoutes[0].children[0].path
          break
        case '4':
          targetPath = this.jurisdCenterRoutes[0] && this.jurisdCenterRoutes[0].children[0].path
          break
        case '5':
          targetPath = this.ywptPageRoutes[0] && this.ywptPageRoutes[0].children[0].path
          break
        case '7':
          targetPath = this.baoBiaoPageRoutes[0] && this.baoBiaoPageRoutes[0].children[0].path
          break
        case '6':
          targetPath = this.dataManaPageRoutes[0] && this.dataManaPageRoutes[0].children[0].path
          break
        case '11':
          targetPath = getBannerHomePath('11')
          break
        case '12':
        case '13':
        case '14':
          targetPath = getBannerHomePath(activeIndex)
          break
        case '0':
          this.openCollectionCloud()
          break
      }
      if (targetPath) {
        this._bannerNavigating = true
        beginBannerNav()
        const shouldJscFullscreen = activeIndex === '11'
        const wasAlreadyOnJsc = shouldJscFullscreen && (
          this.$route.path === '/jsc' || this.$route.path.startsWith('/jsc/')
        )
        Promise.resolve(this.pushRouteIfNeeded(targetPath)).finally(() => {
          this._bannerNavigating = false
          endBannerNav()
          if (shouldJscFullscreen) {
            this.$nextTick(() => {
              if (wasAlreadyOnJsc) {
                toggleJscFullscreen()
              } else {
                requestJscFullscreen()
              }
            })
          }
        })
      }
      localStorage.setItem('bannerId', this.activeIndex)
      this.$emit('bannerClick', this.activeIndex)
    },
    async init() {
      // const res = await findPlatConfig()
      // if (res.data && res.data.configDetails) {
      //   const msg = res.data.configDetails
      //   if (msg.mianPageMsg) { // 主页界面
      //     if (msg.mianPageMsg.showLogo === '1') {
      //       this.logoUrl = msg.mianPageMsg.filePath ? 'http://file.business.ditoo.com.cn' + msg.mianPageMsg.filePath : null
      //       this.logoName = msg.mianPageMsg.logoName ? msg.mianPageMsg.logoName : ''
      //     }
      //   }
      // }

      if (config.platform === PLATFORM.DCSY_INTRANET) { // 锂硫电池溯源平台
        if (config.port === '8003' || config.port === '80') {
          this.logoUrl = ''
          this.logoName = '锂硫电池溯源平台'
          this.headImg = dcsyLogoImage
        } else if (config.port === '5002') {
          this.logoUrl = ''
          this.logoName = '物联网云平台'
        }
      } else if (config.platform === PLATFORM.DCSY_XC) { // 信创
        this.logoUrl = ''
        this.logoName = '锂硫电池溯源平台'
        this.headImg = dcsyLogoImage
      }
    },
    openCollectionCloud() { // 采集云已融合到主前端，站内地址直接使用路由跳转
      const target = String(config.CJYDomainName || '/dcsjgl/wgpz').trim()
      if (!target) {
        return
      }
      if (target.startsWith('/')) {
        return this.pushRouteIfNeeded(target)
      }
      const url = /^https?:\/\//i.test(target) ? target : `http://${target}`
      window.open(url, '_blank', 'noopener,noreferrer')
    },
    goTolink() { // 兼容历史调用入口
      this.activeIndex = '0'
      return this.openCollectionCloud()
    },
    goCustomInterface() { // 打开界面风格配置页面
      this.activeIndex = '999'
      this.$emit('bannerClick', this.activeIndex)
    },
    modifyPas() {
      this.activeIndex = '6'
      this.$emit('bannerClick', this.activeIndex)
      this.$router.push(`/modifyPas`)
    },
    logout() {
      expireSession({
        notifyBackend: true,
        redirectPath: this.$route.fullPath
      })
    },
    async findWarningTempData() { // 查询告警信息
      const res = await findCollectLiveTriggerTempData({
        pageNumber: 1,
        pageSize: 999999
      })
      const nowTimestamp = Date.parse(new Date()) // 当前时间的时间戳
      if (res.list) {
        if (this.WarningTotal !== 0) {
          let hasWarning = false
          res.list.forEach(item => {
            if (item.begin > nowTimestamp) {
              hasWarning = true
            }
          })
          this.WarningTotal = res.total
          if (hasWarning) {
            this.playMp3()
          }
        } else {
          this.WarningTotal = res.total
        }
      }
      this.platformList.forEach(item => {
        if (item.name === '告警中心') {
          item.warningTotal = this.WarningTotal
        }
      })
    },
    playMp3() {
      music = document.getElementById('vd')
      if (music.paused) { // 判读是否播放
        music.play() // 没有就播放
        this.$notify({
          title: '告警',
          message: '告警中心有新的告警，请前往查看',
          duration: 30000,
          onClose: () => {
            this.stopMp3()
          }
        })
      }
    },
    stopMp3() {
      music = document.getElementById('vd')
      music.currentTime = 0
      music.pause()
    },
    screenfullClick() {
      if (this.$route.path !== '/jsc' && !this.$route.path.startsWith('/jsc/')) {
        this.$message({
          message: '请先进入驾驶舱',
          type: 'info'
        })
        return false
      }
      if (!screenfull.isEnabled) {
        this.$message({
          message: '您的浏览器无法全屏',
          type: 'warning'
        })
        return false
      }
      toggleJscFullscreen().catch(() => {})
    }
  }
}
</script>

<style>
.el-menu.el-menu--horizontal {
    border-bottom: 0px;
}
</style>

<style lang="stylus" scoped>
bannerHeight = var(--business-banner-height, 3.5417vw)

.bannerCss {
  flex-shrink 0
  position relative
  width 99.5vw
  height bannerHeight
  background-color #042957
  .logoCss {
    width 9.375vw
    height bannerHeight
  }
  .logoNameCss {
    padding-left 2vw
    height bannerHeight
    line-height bannerHeight
    color #fff
    font-size 1vw
  }
  .groupOneCss {
    position absolute
    top 0
    left 16.875vw
    width 11.8229167vw
    height bannerHeight
    z-index 999
  }
  .bannerButContentCss {
    position absolute
    top 0
    left 26.7vw
    padding 0 2vw
    width 46.614583vw
    height bannerHeight
    background-color #183857
    color red
    .numeContentCss {
      position relative
      float left
      margin-top 0.8vw
      max-width 10vw
      padding 0 0.2vw
      .bannerButCss {
        height 1.822916vw
        line-height 1.822916vw
        text-align center
        background-color rgba(54,154,254,0.1)
        border-radius 4px 4px 4px 4px
        font-size 14px
        // font-family FZKTK // 方正楷体
        font-weight 500
        color #FFFFFF
      }
      .activeCss {
        background-color #369AFE
      }
      .warningTotalCss {
        position absolute
        top -3px
        right -3px
        color red
        font-size 14px
        font-weight bolder
      }
    }
  }
  .groupTwoCss {
    position absolute
    top 0
    right 16.875vw
    width 11.8229167vw
    height bannerHeight
    z-index 999
  }
  .loginCjyCss {
    position absolute
    top 0
    right 13vw
    height bannerHeight
    line-height bannerHeight
    z-index 999
    color white
    font-size 1vw
  }
  .loginCjyCss:hover{
    color #42C700
  }
  .userOperationCss{
    position absolute
    top 0.625vw
    right 4vw
    width 10vw
    height 2.2917vw
    background-color #1B4267
    border-radius 60px 60px 60px 60px
    .userPortraitCss {
      float left
      margin 0.260416vw 0.364583vw 0 0.364583vw
      width 1.77083vw
      height 1.77083vw
    }
    .userNameCss {
      float left
      font-size 0.83vw
      font-family Noto Sans SC-Regular, Noto Sans SC
      font-weight 400
      color #BFC7D4
      line-height 2.2917vw
      -webkit-background-clip text
    }
    .selectIconCss {
      float right
      margin 1vw 0.5vw 0 0
      width 0
      height 0
      border 0.4vw solid transparent
      border-top-color #BFC7D4
    }
  }
  .screenfullCss {
    position absolute
    top 1.197916vw
    right 1.5vw
    width 1.0416vw
    height 1.0416vw
  }
}

// 在低分辨率或浏览器放大时保留原配色和按钮，菜单超出宽度可横向滚动。
@media (max-width: 1280px) {
  .bannerCss {
    width 100%
    .logoNameCss {
      padding-left 16px
      width 185px
      font-size 14px
      white-space nowrap
      overflow hidden
      text-overflow ellipsis
    }
    .groupOneCss, .groupTwoCss {
      display none
    }
    .bannerButContentCss {
      left 185px
      right 150px
      width auto
      display flex
      align-items center
      padding 0 4px
      overflow-x auto
      overflow-y hidden
      white-space nowrap
      scrollbar-width thin
      > div {
        flex-shrink 0
      }
      .numeContentCss {
        float none
        margin-top 0
        width auto !important
        max-width none
        padding 0 3px
        .bannerButCss {
          height 32px
          line-height 32px
          padding 0 12px
        }
      }
    }
    .userOperationCss {
      top 9px
      right 36px
      width 108px
      height 34px
      .userPortraitCss {
        margin 4px 5px
        width 26px
        height 26px
      }
      .userNameCss {
        width 54px
        font-size 13px
        line-height 34px
        white-space nowrap
        overflow hidden
        text-overflow ellipsis
      }
      .selectIconCss {
        margin 15px 6px 0 0
        border-width 4px
      }
    }
    .screenfullCss {
      top 17px
      right 10px
      width 18px
      height 18px
    }
  }
}

@media (max-width: 640px) {
  .bannerCss {
    .logoNameCss {
      width 88px
      padding-left 8px
    }
    .bannerButContentCss {
      left 88px
    }
  }
}
img {
  width 100%
  height 100%
}
.funcSelectCss {
  width 120px
  text-align center
  line-height 30px
  .borderBottomCss {
    border-bottom 1px solid #378ef8
  }
}
</style>
