<template>
  <div>
    <EquipMana />
  </div>
</template>

<script>
import EquipMana from '@/views/equipMana/equipMana.vue'

export default {
  // 保留改名前的组件名称，使既有路由缓存继续生效。
  name: 'jrgl',
  components: {
    EquipMana
  }
}
</script>
