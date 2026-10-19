<template>
  <div class="gateway-launch-page">
    <el-result icon="success" title="网关配置已在新窗口打开" />
  </div>
</template>

<script>
import { Message } from '@/utils/elementShim'

export default {
  name: 'GatewayConfigLauncher',
  mounted() {
    this.openCollectionCloud()
  },
  methods: {
    openCollectionCloud() {
      // 两套界面共用当前站点和登录令牌，仅在新标签页切换到采集云独立布局。
      const target = this.$router.resolve({
        name: 'collectionHome',
        query: { module: 'gateway' }
      }).href
      const collectionWindow = window.open(new URL(target, window.location.href).toString(), '_blank')
      if (collectionWindow) {
        collectionWindow.opener = null
        return
      }
      Message({
        message: '浏览器阻止了新窗口，请允许弹出窗口后重试',
        type: 'warning'
      })
    }
  }
}
</script>

<style lang="stylus" scoped>
.gateway-launch-page {
  min-height 420px
  display flex
  align-items center
  justify-content center
  background-color #DAE3EC
}
</style>

