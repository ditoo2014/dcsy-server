<template>
  <div>
    <GeneralTriggerTool :global-mode="true" />
  </div>
</template>

<script>
import GeneralTriggerTool from '@/views/equipMana/components/GeneralTriggerTool.vue'

export default {
  // 保留改名前的组件名称，使既有路由缓存继续生效。
  name: 'tycfqgl',
  components: {
    GeneralTriggerTool
  }
}
</script>
