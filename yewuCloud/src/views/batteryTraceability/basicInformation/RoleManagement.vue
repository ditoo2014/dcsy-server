<template>
  <div>
    <roleMana />
  </div>
</template>

<script>
import roleMana from '@/views/personMana/roleMana.vue'
export default {
  // 保留改名前的组件名称，使既有路由缓存继续生效。
  name: 'jsgl',
  components: {
    roleMana
  }
}
</script>
