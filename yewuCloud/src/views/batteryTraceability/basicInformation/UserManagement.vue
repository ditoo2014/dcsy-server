<template>
  <div>
    <userMana />
  </div>
</template>

<script>
import userMana from '@/views/personMana/userMana.vue'
export default {
  // 保留改名前的组件名称，使既有路由缓存继续生效。
  name: 'yhgl',
  components: {
    userMana
  }
}
</script>
