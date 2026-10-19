<template>
  <div>
    <AccountUserManagement />
  </div>
</template>

<script>
import AccountUserManagement from '@/views/accounts/UserManagement.vue'
export default {
  // 组件名称与路由名称一致，确保页面缓存正常匹配。
  name: 'UserManagement',
  components: {
    AccountUserManagement
  }
}
</script>
