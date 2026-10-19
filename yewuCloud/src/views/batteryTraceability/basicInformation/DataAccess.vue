<template>
  <div>
    <EquipmentManagement />
  </div>
</template>

<script>
import EquipmentManagement from '@/views/equipment/EquipmentManagement.vue'

export default {
  // 组件名称与路由名称一致，确保页面缓存正常匹配。
  name: 'DataAccess',
  components: {
    EquipmentManagement
  }
}
</script>
