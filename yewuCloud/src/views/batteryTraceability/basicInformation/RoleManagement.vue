<template>
  <div>
    <AccountRoleManagement />
  </div>
</template>

<script>
import AccountRoleManagement from '@/views/accounts/RoleManagement.vue'
export default {
  // 组件名称与路由名称一致，确保页面缓存正常匹配。
  name: 'RoleManagement',
  components: {
    AccountRoleManagement
  }
}
</script>
