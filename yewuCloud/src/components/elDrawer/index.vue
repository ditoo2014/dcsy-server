<template>
  <!-- 必须用 ElDrawer 显式引用，避免 <el-drawer> 被解析为全局 elDrawer 导致栈溢出 -->
  <ElDrawer
    v-model="visible"
    :title="title"
    :size="drawerSize"
    :close-on-click-modal="closeOnClickModal"
    :close-on-press-escape="closeOnPressEscape"
    :destroy-on-close="destroyOnClose"
    :append-to-body="appendToBody"
    direction="rtl"
    v-bind="$attrs"
  >
    <slot />
    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </ElDrawer>
</template>

<script>
import { ElDrawer } from 'element-plus'

/** Element Plus 抽屉封装（业务侧使用 <elDrawer> 标签） */
export default {
  // 与全局标签 elDrawer 保持一致
  name: 'ElDrawerWrap',
  components: { ElDrawer },
  inheritAttrs: false,
  props: {
    modelValue: {
      type: Boolean,
      default: false
    },
    title: {
      type: String,
      default: ''
    },
    width: {
      type: [String, Number],
      default: 500
    },
    maskClosable: {
      type: Boolean,
      default: true
    },
    wrapperClosable: {
      type: Boolean,
      default: true
    },
    closable: {
      type: Boolean,
      default: true
    },
    destroyOnClose: {
      type: Boolean,
      default: false
    },
    appendToBody: {
      type: Boolean,
      default: true
    }
  },
  emits: ['update:modelValue'],
  computed: {
    visible: {
      get() {
        return this.modelValue
      },
      set(val) {
        this.$emit('update:modelValue', val)
      }
    },
    closeOnClickModal() {
      return this.maskClosable !== false && this.wrapperClosable !== false
    },
    closeOnPressEscape() {
      return this.closable !== false
    },
    drawerSize() {
      const w = this.width
      if (w == null || w === '') return '500px'
      if (typeof w === 'number') return `${w}px`
      if (typeof w === 'string' && /^\d+$/.test(w)) return `${w}px`
      return w
    }
  }
}
</script>
