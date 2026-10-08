<template>
  <el-config-provider :locale="locale" :size="elementSize">
    <div id="app">
      <router-view />
    </div>
  </el-config-provider>
</template>

<script>
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import { computed, inject, onBeforeUnmount, watch } from 'vue'
import { useRoute } from 'vue-router'
import { setCollectionPageActive } from './styles/pageStyles'

export default {
  name: 'App',
  setup() {
    const route = useRoute()
    const businessElementSize = inject('defaultElementSize', 'default')
    const isCollectionCloud = computed(() => route.matched.some(record => record.meta.collectionCloud))

    // 采集云原项目未设置全局组件尺寸，不能继承业务云保存在 Cookie 中的尺寸。
    const elementSize = computed(() => (isCollectionCloud.value ? undefined : businessElementSize))

    // 全局样式按页面切换，挂载到 body 的弹窗和下拉框也使用当前页面的样式。
    watch(isCollectionCloud, active => {
      setCollectionPageActive(active)
    }, { immediate: true, flush: 'sync' })

    onBeforeUnmount(() => {
      setCollectionPageActive(false)
    })

    return {
      locale: zhCn,
      elementSize
    }
  }
}
</script>

