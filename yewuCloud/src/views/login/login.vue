<template>
  <div class="loginViewportCss">
    <div v-if="!isApp" :style="{height:screeHeight+'px'}">
      <div v-if="showDcsyLogin" class="loginPageCss">
        <div class="dcsyCss">
          <div class="dcsyContentCss">
            <div class="dcsyTitleCss">
              <div class="dcsyLogoCss">
                <img :src="dcsyLogoImage">
              </div>
              <div class="dcsyTitleMsgCss">锂硫电池溯源平台</div>
            </div>
            <div class="dcsyInputAllCss">
              <div class="dcsyInputContentCss">
                <el-input
                  v-model="loginForm.username"
                  placeholder="请输入您的账号"
                  autocomplete="username"
                  name="username"
                  :prefix-icon="$legacyIcon('el-icon-user')"
                />
                <div class="dcsyPasswordInputCss">
                  <el-input
                    v-model="loginForm.password"
                    placeholder="请输入您的密码"
                    type="password"
                    show-password
                    autocomplete="current-password"
                    name="password"
                    :prefix-icon="$legacyIcon('el-icon-lock')"
                    @keyup.enter="sendSmsCode"
                  />
                </div>
                <div class="dcsySmsInputCss">
                  <el-input
                    v-model="loginForm.smsCode"
                    placeholder="请输入短信验证码"
                    :prefix-icon="$legacyIcon('el-icon-message')"
                    @keyup.enter="handleLogin"
                  />
                  <el-button
                    type="primary"
                    plain
                    class="dcsySmsButtonCss"
                    :disabled="smsSending || smsCountdown > 0"
                    @click="sendSmsCode"
                  >
                    {{ smsButtonText }}
                  </el-button>
                </div>
              </div>

              <div class="dcsyButContentCss">
                <div class="dcsyButCss">
                  <el-button type="primary" @click="handleLogin">登录</el-button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div v-else>
      <el-image
        style="width:100%;height:100%"
        :src="backgroundImage"
        fit="fill"
      />
    </div>
  </div>
</template>

<script>
import { Message } from '@/utils/element-shim'
import Cookies from 'js-cookie'
import config from '@/config.js'
import { PLATFORM, isDcsyLoginUi, isDcsyMainUi } from '@/constants/platform'
import { validUsername } from '@/utils/validate' // 账号密码表单验证
import { getToken, setUsername, setToken, setMydb, setUserId } from '@/utils/auth.js'
import { parseUserInfo } from '@/utils/safe-json.js'
import { sendLoginSmsCode } from '@/api/user.js'
import { statistictaskPlan } from '@/api/lldcsy.js'
import { getRSApassword, getTime, dateConvStamp, dateAddOrSubtract } from '@/utils/utils.js'
import router from '@/router'
import { markDynamicRoutesReady } from '@/permission.js'
import { resolveAccessiblePath } from '@/utils/resolve-accessible-route.js'
import { getDefaultBannerId, resolveBannerIdFromPath } from '@/utils/banner-route.js'
import dcsyLogoImage from './dcsyLogo.png'
import backgroundImage from './background.png'

const str = 'abcdefghigklmnopqrstuvwxyz123456789%@#$<>^&*_+'

export default {
  data() {
    return {
      dcsyLogoImage,
      backgroundImage,
      isApp: true, // APP端
      screeHeight: document.documentElement.clientHeight, // 屏幕高

      platform: config.platform,
      ip: config.ip,
      port: config.port,

      iframeData: null,

      getPageUrl: '', // 文件位于根目录的public文件夹下

      loginForm: {
        username: '',
        password: '',
        smsCode: ''
      },

      smsCodeId: '',
      smsCountdown: 0,
      smsTimer: null,
      smsSending: false
    }
  },
  computed: {
    showDcsyLogin() {
      return isDcsyLoginUi(this.platform)
    },
    smsButtonText() {
      if (this.smsCountdown > 0) {
        return `${this.smsCountdown}s后重发`
      }
      return this.smsSending ? '发送中' : '获取验证码'
    }
  },
  mounted() {
    const query = this.$route.query
    if (query && query.APP) { // 判断本网页是否被嵌入在业务云app中显示
      setUsername(query.username)
      setToken(query.token)
      setMydb(query.mydb)
      setUserId(query.userId)
      if (query.APP === '1') {
        let row = { id: query.id, equipmentName: query.equipmentName, app: true, apiKey: query.apiKey }
        row = JSON.stringify(row)
        this.$router.push({ path: '/running', query: { row }})
      } else if (query.APP === '2') { // 运行中心
        this.$router.push({ path: `/kanBanMonitor`, query })
      } else if (query.APP === '4') {
        this.$router.push({ path: '/jurisd', query })
      } else if (query.APP === '5') {
        this.$router.push({ path: '/hisJurisd', query })
      }
    } else {
      if (isDcsyMainUi(this.platform)) {
        this.getPageUrl = 'static/SOfSATLogin/index.html'
      }
      this.isApp = false
    }
    // 独立监听尺寸变化，避免覆盖其他页面已经注册的 resize 处理。
    window.addEventListener('resize', this.updateScreenHeight)
    this.updateScreenHeight()
    this._loginInitTimer = setTimeout(() => {
      this.init()
      if (this.platform === PLATFORM.DCSY_INTRANET && this.port === '80') {
        // this.zfy()
      }
    }, 500)
  },
  beforeUnmount() {
    this.clearSmsTimer()
    clearTimeout(this._loginInitTimer)
    window.removeEventListener('resize', this.updateScreenHeight)
    window.removeEventListener('message', this.handleIframeMessage)
  },
  methods: {
    updateScreenHeight() {
      this.screeHeight = document.documentElement.clientHeight
    },
    handleIframeMessage(event) {
      this.iframeData = event.data
      this.getMsg()
    },
    init() {
      const iframeDoc = this.$refs.iframeRef
      if (iframeDoc) {
        const iframeWid = iframeDoc.contentWindow
        let apTitle = ''
        if (this.platform === PLATFORM.DCSY_INTRANET) {
          if (this.port === '80') {
            apTitle = '锂硫电池溯源平台'
          } else if (this.port === '5002') {
            apTitle = '设备智能化远程监控平台'
          }
        } else if (isDcsyLoginUi(this.platform)) {
          apTitle = '锂硫电池溯源平台'
        }
        if (apTitle) {
          iframeWid.postMessage({ apTitle }, '*')
        }

        window.addEventListener('message', this.handleIframeMessage)
      }
    },
    validateAccountPassword() {
      if (!validUsername(this.loginForm.username) || !validUsername(this.loginForm.password)) {
        Message({
          message: '用户名和密码不能为空',
          type: 'error',
          duration: 3 * 1000
        })
        return false
      }
      return true
    },
    async sendSmsCode() {
      if (!this.validateAccountPassword()) return
      this.smsSending = true
      try {
        const res = await sendLoginSmsCode({
          username: this.loginForm.username,
          password: getRSApassword(this.loginForm.password, this)
        })
        if (Number(res.code) === 200) {
          const smsCodeId = res.return && res.return.smsCodeId
          if (!smsCodeId) {
            Message({
              message: '短信验证码发送失败',
              type: 'error',
              duration: 3 * 1000
            })
            return
          }
          this.smsCodeId = smsCodeId
          this.loginForm.smsCode = ''
          this.startSmsCountdown((res.return && res.return.intervalSeconds) || 60)
          Message({
            message: res.return && res.return.phone
              ? `验证码已发送至 ${res.return.phone}`
              : (res.msg || '短信验证码已发送'),
            type: 'success',
            duration: 3 * 1000
          })
        } else {
          Message({
            message: res.msg || '短信验证码发送失败',
            type: 'error',
            duration: 3 * 1000
          })
        }
      } catch (error) {
        console.log('send sms code error:', error)
      } finally {
        this.smsSending = false
      }
    },
    startSmsCountdown(seconds) {
      this.clearSmsTimer()
      this.smsCountdown = seconds
      this.smsTimer = setInterval(() => {
        if (this.smsCountdown <= 1) {
          this.clearSmsTimer()
          return
        }
        this.smsCountdown -= 1
      }, 1000)
    },
    clearSmsTimer() {
      if (this.smsTimer) {
        clearInterval(this.smsTimer)
        this.smsTimer = null
      }
      this.smsCountdown = 0
    },
    handleLogin() { // 登录逻辑说明 https://www.cnblogs.com/fqh123/p/11094296.html
      if (!this.validateAccountPassword()) return
      if (!this.smsCodeId) {
        Message({
          message: '请先获取短信验证码',
          type: 'warning',
          duration: 3 * 1000
        })
        return
      }
      if (!this.loginForm.smsCode) {
        Message({
          message: '请输入短信验证码',
          type: 'warning',
          duration: 3 * 1000
        })
        return
      }

      const loginForm = {
        username: this.loginForm.username,
        password: getRSApassword(this.loginForm.password, this),
        smsCodeId: this.smsCodeId,
        smsCode: this.loginForm.smsCode
      }
      setUsername(loginForm.username) // 将username存入缓存中
      this.loading = true
      setTimeout(() => {
        this.loading = false
      }, 1500)
      // 这里store的user/login表示@/store/modules/user.js中的actions.login()方法
      // 其中modules路径在@/store/index.js中配置
      // https://blog.csdn.net/weixin_43067157/article/details/82079792
      this.$store.dispatch('user/login', loginForm).then(async res => {
        if (res === true) {
          this.clearSmsTimer()
          try {
            await this.navigateAfterLogin()
          } catch (error) {
            const errMsg = (error && error.message) || error || '登录成功，但加载权限失败'
            Message({
              message: typeof errMsg === 'string' ? errMsg : '登录成功，但加载权限失败',
              type: 'error',
              duration: 3 * 1000
            })
          }
        }
      }).catch(() => {
        this.loading = false
      })
    },
    /** 登录成功后先注入动态路由，再跳转到当前账号实际可访问的页面 */
    async navigateAfterLogin() {
      const requestedToken = getToken()
      localStorage.removeItem('routerPath')

      const roleInfo = await this.$store.dispatch('user/getInfo')
      if (getToken() !== requestedToken) return
      if (roleInfo.passwordChangeRequired) {
        await this.$router.replace({ path: '/modifyPas' })
        return
      }
      const accessRoutes = await this.$store.dispatch('permission/generateRoutes', roleInfo)
      if (getToken() !== requestedToken) return
      accessRoutes.forEach(route => router.addRoute(route))
      markDynamicRoutesReady()

      const preferredPath = config.platform === PLATFORM.DCSY_XC
        ? ''
        : '/home'
      const path = resolveAccessiblePath(router, preferredPath, accessRoutes, {
        platform: config.platform
      })
      const bannerId = resolveBannerIdFromPath(path) || getDefaultBannerId()
      localStorage.setItem('bannerId', bannerId)
      await this.$router.replace({ path })
    },
    getMsg() {
      const cookiesLogInTimestamp = Cookies.get('logInTimestamp')
      const nowTimestamp = Date.parse(new Date()) // 当前时间的时间戳
      if (!cookiesLogInTimestamp || (nowTimestamp - cookiesLogInTimestamp) / 1000 > 3) {
        if (this.iframeData.keyCode === 13) {
          this.loginForm = {
            ...this.loginForm,
            ...this.iframeData
          }
          Cookies.set('logInTimestamp', nowTimestamp)
          this.sendSmsCode()
        }
      }
    },
    randomChar() {
      return str[Math.floor(Math.random() * str.length)]
    },
    randomChars(num) {
      let val = ''
      for (let i = 0; i < num; i++) {
        val += this.randomChar()
      }
      return val
    },
    zfy() { // 字符雨
      const perColumnNums = 25
      const n = 50
      const animationTime = 4

      const container = this.$refs.containerRef

      const styleEl = document.createElement('style')

      for (let i = 0; i < n; i++) {
        const p = document.createElement('p')
        p.className = 'zfyPCss'
        container.appendChild(p)

        const content = this.randomChars(perColumnNums)
        const contentNext = this.randomChars(perColumnNums)
        const delay = Math.floor(Math.random() * n)
        const randomAnimationTime = (animationTime + Math.random() * 2 - 1).toFixed(1) + 's'
        const alpha = (Math.random() * 0.4 + 0.6).toFixed(2)

        styleEl.innerHTML += `
          .containerRefCss .zfyPCss:nth-child(${i + 1})::before {
            content: '${content}';
            color: rgb(179, 255, 199);
            text-shadow: 0 0 1px #fff, 0 0 2px #fff, 0 0 5px currentColor, 0 0 10px currentColor;
            animation: typing-${i} ${randomAnimationTime} steps(20, end) ${-(delay * 0.1)}s infinite;
            z-index: 1;
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 100%;
            overflow: hidden;
          }

          .containerRefCss .zfyPCss:nth-child(${i + 1})::after {
            content: '';
            background: linear-gradient(rgba(0, 0, 0, ${alpha}), rgba(0, 0, 0, ${alpha}), rgba(0, 0, 0, ${alpha}), transparent 75%, transparent);
            background-size: 100% 220%;
            background-repeat: no-repeat;
            animation: maskAnimation ${randomAnimationTime} linear ${-((delay - 2) * 0.1)}s infinite;
            z-index: 2;
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 100%;
            overflow: hidden;
          }

          @keyframes typing-${i} {
            0% { height: 0; }
            25% { height: 100%; }
            100% {
              height: 100%;
              content: '${contentNext}';
            }
          }
        `
      }

      document.head.appendChild(styleEl)
    },
    async cfdtjPlan() { // 充放电统计计划
      const userInfo = parseUserInfo({})
      if (userInfo.isadmin === 1) { // 管理员
        const nowDate = getTime()
        const basicBarXData = []
        basicBarXData.push(nowDate)
        for (let i = 0; i < 13; i++) { // 往前推13天的日期
          const date = dateAddOrSubtract(-1, basicBarXData[basicBarXData.length - 1], 'date', 'date')
          basicBarXData.push(date)
        }
        let startDate = ''
        let endDate = ''
        basicBarXData.forEach((item, index) => {
          if (index === 7) {
            endDate = item.slice(0, 10) + ' 23:59:59'
          } else if (index === basicBarXData.length - 1) {
            startDate = item.slice(0, 10) + ' 00:00:00'
          }
        })
        statistictaskPlan({
          startTime: dateConvStamp(startDate) / 1000,
          endTime: dateConvStamp(endDate) / 1000
        })
      }
    }
  }
}
</script>

<style lang="stylus" scoped>
.loginViewportCss {
  height 100vh
  overflow auto
  background-color #020B42
}
.loginPageCss {
  width 100%
  height 100%
}
.loginContentCss {
  position absolute
  top 12vw
  right 7.7084vw
  width 28.22916vw
  height 25vw
  background rgba(0, 0, 0, 0.4)
  border-radius 10px
  padding 4.375vw 4.583vw 0 4.583vw
  .titleCss {
    height 2.760417vw
    font-size 1.979167vw
    font-family PingFang SC-Bold, PingFang SC
    font-weight bold
    color #1D1D1D
    line-height 45px
    -webkit-background-clip text
  }
  .inputContentCss {
    margin-top 2.239583vw
    .passwordInputCss {
      margin-top 1.09375vw
    }
  }
  .butContentCss {
    margin-top 2vw
    width 100%
    .butCss {
      width 70px
      margin auto
    }
  }
}
.dcsyCss {
  position relative
  width 100%
  height 100%
  background-color #020B42
}

.dcsyContentCss {
  position absolute
  top 12vw
  right 35vw
  width 30vw
  height 23vw
  background rgba(110,140,226, 0.5)
  border-radius 10px
  z-index 99
  .dcsyTitleCss {
    position absolute
    top 2vw
    left 50%
    transform translate(-50%, 0) /* 将元素移动自身尺寸的一半 */
    width 80% /* 子元素的宽度 */
    height 2.760417vw
    font-size 1.979167vw
    font-family PingFang SC-Bold, PingFang SC
    font-weight bold
    color #fff
    line-height 3vw
    -webkit-background-clip text
    letter-spacing 0.5vw
    text-align center
    .dcsyLogoCss {
      float left
      margin-right 1vw
      width 1.9vw
      height 3vw
    }
    .dcsyTitleMsgCss {
      float left
    }
  }
  .dcsyInputAllCss {
    position absolute
    top 6vw
    left 50%
    transform translate(-50%, 0) /* 将元素移动自身尺寸的一半 */
    width 58% /* 子元素的宽度 */
    .dcsyInputContentCss {
      margin-top 2vw
      .dcsyPasswordInputCss {
        margin-top 0.95vw
        :deep(.el-input__wrapper) {
          min-height 32px
          box-sizing border-box
        }
        :deep(.el-input__inner) {
          height 32px
          line-height 32px
          padding-right 36px
        }
        :deep(.el-input__suffix) {
          right 8px
        }
        :deep(.el-input__password) {
          cursor pointer
        }
      }
      .dcsySmsInputCss {
        display flex
        align-items center
        gap 0.5vw
        margin-top 0.95vw
        .el-input {
          flex 1
        }
        .dcsySmsButtonCss {
          width 6vw
          min-width 92px
          height 32px
          padding 0 8px
          white-space nowrap
        }
      }
    }
    .dcsyButContentCss {
      margin-top 1.6vw
      width 100%
      .dcsyButCss {
        width 70px
        margin auto
      }
    }
  }
}
// 窄屏使用原版登录卡片与颜色，仅保证输入框、验证码和登录按钮能完整显示。
@media (max-width: 1280px) {
  .dcsyCss {
    min-height 420px
  }
  .dcsyContentCss {
    top 80px
    right auto
    left 50%
    transform translateX(-50%)
    width 384px
    max-width calc(100% - 24px)
    height 310px
    .dcsyTitleCss {
      top 24px
      width 90%
      height 36px
      display flex
      align-items center
      justify-content center
      font-size 24px
      line-height 36px
      letter-spacing 3px
      white-space nowrap
      .dcsyLogoCss {
        float none
        flex-shrink 0
        margin-right 12px
        width 24px
        height 36px
      }
      .dcsyTitleMsgCss {
        float none
      }
    }
    .dcsyInputAllCss {
      top 82px
      width 72%
      .dcsyInputContentCss {
        margin-top 16px
        .dcsyPasswordInputCss, .dcsySmsInputCss {
          margin-top 12px
        }
        .dcsySmsInputCss {
          gap 6px
          .el-input {
            min-width 0
          }
        }
      }
      .dcsyButContentCss {
        margin-top 20px
      }
    }
  }
}

@media (max-height: 650px) {
  .dcsyCss {
    // 低高度窗口可向下滚动，卡片不再压住底部登录按钮。
    min-height 420px
  }
  .dcsyContentCss {
    top 40px
    min-height 310px
  }
}
.oldWrapper {
  position: absolute;
  width: 100%;
  bottom: 30px;
  font-size: 16px;
  color: #E0E0E0;
  text-align: center;
}

img {
  width 100%
  height 100%
}
</style>

<style lang="stylus">
  .zfyContentCss {
    margin 0
    padding 0
    width 100%
    height 100%
    background #000
    overflow hidden
    display flex
    font-family 'Noto Sans Mono', 'Noto Sans Mono CJK SC', 'Courier New', monospace
    .containerRefCss {
      width 100vw
      display flex
      justify-content space-between
      flex-wrap nowrap
      flex-direction row
      .zfyPCss {
        position relative
        width 2vw
        height 100vh
        text-align center
        font-size 2.2vw
        line-height 1.1
        word-break break-word
        white-space pre-wrap
        overflow hidden
        margin 0
        padding 0
      }

      @keyframes maskAnimation {
        0% {
          background-position 0 220%
        }
        30% {
          background-position 0 0%
        }
        100% {
          background-position 0 0%
        }
      }
    }
  }
</style>
