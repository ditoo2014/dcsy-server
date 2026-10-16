<template>
  <div>
    <systemLog />
  </div>
</template>

<script>
import systemLog from '@/views/basicData/systemLog.vue'
export default {
  // 保留改名前的组件名称，使既有路由缓存继续生效。
  name: 'xtrz',
  components: {
    systemLog
  }
}
</script>
