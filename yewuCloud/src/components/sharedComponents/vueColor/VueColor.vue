<template>
  <div>
    <section>
      <div class="color_con" :style="{ background: setColor }" @click="handleShowColor" />
      <div v-show="colorShow" class="color_set">
        <SketchPicker :model-value="setColor" @update:model-value="updateValue" />
      </div>
    </section>
  </div>
</template>

<script>
import { Sketch as SketchPicker } from '@ckpack/vue-color'

/** 使用原 Sketch 选色面板的 Vue 3 版本，保留色块尺寸及展开方式。 */
export default {
  name: 'VueColor',
  components: { SketchPicker },
  props: {
    color: { type: String, default: '#000' },
    colorSelectEd: { type: Boolean, default: true },
    disabled: { type: Boolean, default: false }
  },
  emits: ['color-select'],
  data() {
    return { setColor: this.color, colorShow: this.colorSelectEd }
  },
  watch: {
    color(value) {
      this.setColor = value
    }
  },
  methods: {
    handleShowColor() {
      if (!this.disabled) this.colorShow = !this.colorShow
    },
    updateValue(value) {
      // 业务表保存 Hex 字符串，事件仍提供完整颜色对象供原数据表使用。
      const hex = typeof value === 'string' ? value : value?.hex
      if (!hex) return
      this.setColor = hex
      this.$emit('color-select', typeof value === 'string' ? { hex } : value)
    }
  }
}
</script>

<style lang="stylus" scoped>
withLang = 30px
paddingLeft = 40px

.color_con {
  float left
  width withLang
  height withLang
}
.color_set {
  padding-left paddingLeft
}
</style>
