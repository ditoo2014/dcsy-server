<template>
  <div>
    <systemLog />
  </div>
</template>

<script>
import systemLog from '@/views/basicData/systemLog.vue'
export default {
  // 组件名称与路由名称一致，确保页面缓存正常匹配。
  name: 'SystemLogs',
  components: {
    systemLog
  }
}
</script>
