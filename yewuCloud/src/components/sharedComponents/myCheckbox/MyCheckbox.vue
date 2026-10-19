<template>
  <div class="myCheckboxCss">
    <div v-if="isAllCheck" class="checkboxAllCss">
      <label class="customCheckboxCss">
        <input
          ref="checkboxRef"
          v-model="allChecked"
          type="checkbox"
          @click="toggleAllState"
        >
        <span class="checkmarkCss" />
        <span class="nameCss">全部</span>
      </label>
    </div>
    <div class="checkboxContentCss">
      <div v-for="(item, index) in checkList" :key="index" class="checkboxItemCss">
        <label class="customCheckboxCss">
          <input
            v-model="item.checked"
            type="checkbox"
            @click="toggleState(item)"
          >
          <span class="checkmarkCss" />
          <span v-if="item.circleLineColor" class="circleCss" :style="{'border-color': item.circleLineColor}">
            <span class="lineCss" :style="{'background-color': item.circleLineColor}" />
          </span>
          <span class="nameCss">{{ item.name }}</span>
        </label>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  props: {
    isAllCheck: { // 是否需要全选按钮
      type: Boolean,
      default: true
    },
    checkList: {
      type: Array,
      default: () => {
        return [
          { checked: false, name: '选项1', circleLineColor: null },
          { checked: true, name: '选项2', circleLineColor: null },
          { checked: false, name: '选项3', circleLineColor: null }
        ]
      }
    }
  },
  data() {
    return {
      allChecked: false, // 是否选中全部
      indeterminate: false, // 是否选中部分

      borderColor: 'red'
    }
  },
  watch: {
    checkList() {
      this.$nextTick(() => {
        this.syncSelectionState()
      })
    }
  },
  mounted() {
    this.toggleState()
  },
  methods: {
    syncSelectionState() {
      const selectedCount = this.checkList.filter(item => item.checked).length
      const hasOptions = this.checkList.length > 0
      this.allChecked = hasOptions && selectedCount === this.checkList.length
      this.indeterminate = selectedCount > 0 && selectedCount < this.checkList.length
      if (this.isAllCheck && this.$refs.checkboxRef) {
        this.$refs.checkboxRef.indeterminate = this.indeterminate
      }
    },
    toggleAllState() { // 点击全选框
      if (this.indeterminate) {
        this.indeterminate = false
        if (this.isAllCheck) this.$refs.checkboxRef.indeterminate = false
      } else {
        this.allChecked = !this.allChecked
      }
      setTimeout(() => {
        this.checkList.forEach(item => {
          item.checked = this.allChecked
        })
        this.$emit('selectChange', this.checkList)
      }, 100)
    },
    setIndeterminate() { // 选中部分
      this.allChecked = false
      this.indeterminate = true
      if (this.isAllCheck) this.$refs.checkboxRef.indeterminate = true
      this.$emit('selectChange', this.checkList)
    },
    setAllChecked() { // 设置全选
      this.allChecked = true
      this.indeterminate = false
      if (this.isAllCheck) this.$refs.checkboxRef.indeterminate = false
      this.$emit('selectChange', this.checkList)
    },
    setUnAllChecked() { // 设置全不选
      this.allChecked = false
      this.indeterminate = false
      if (this.isAllCheck) this.$refs.checkboxRef.indeterminate = false
      this.$emit('selectChange', this.checkList)
    },
    toggleState() { // 点击待选框
      let isSelected = false // 是否存在被选择的数据
      let isNotSelected = false // 是否存在没有被选择的数据
      setTimeout(() => {
        this.checkList.forEach(checkItem => {
          if (checkItem.checked) {
            isSelected = true
          } else {
            isNotSelected = true
          }
        })
        if (isSelected && isNotSelected) {
          this.setIndeterminate() // 选中部分
        } else if (isSelected && !isNotSelected) {
          this.setAllChecked() // 设置全选
        } else {
          this.setUnAllChecked() // 设置全不选
        }
      }, 100)
    }
  }
}
</script>

<style lang="stylus" scoped>
checkmarkWh = 1vw

.myCheckboxCss {
  width 100%
  min-height 3vw
  height auto
  .checkboxAllCss {
    width 100%
    min-height 1.2vw
    height auto
  }
  .checkboxContentCss {
    display flex
    flex-wrap wrap
    align-items flex-start
    column-gap 1vw
    row-gap 0.3vw
    width 100%
    .checkboxItemCss {
      flex 0 0 auto
      .circleCss {
        position relative
        float left
        left 0.5vw
        top 0.2vw
        margin-right 0.7vw
        width 0.6vw /* 圆圈的直径 */
        height 0.6vw /* 圆圈的直径 */
        border-radius 50% /* 将元素变为圆形 */
        border 2px solid #000 /* 圆圈的边框 */
        display flex
        justify-content center
        align-items center
        overflow visible
        .lineCss {
          position absolute /* 绝对定位，使其居中 */
          left 50%
          width 200% /* 横线的长度，超出圆圈边界 */
          height 2px /* 横线的厚度 */
          transform translateX(-50%) /* 水平居中 */
          background-color #000 /* 横线的颜色 */
        }
      }
    }
  }
}

.customCheckboxCss {
  display inline-block
  position relative
  padding-left checkmarkWh
  margin-bottom 0.5vw
  cursor pointer
  font-size 0.7vw
  user-select none
  .nameCss {
    margin-left 0.3vw
  }
}

.customCheckboxCss input {
  position absolute
  opacity 0
  cursor pointer
}

.checkmarkCss {
  position absolute
  top 0
  left 0
  height checkmarkWh
  width checkmarkWh
  background-color #eee
  border 0.05vw solid #ccc
  border-radius 0.15px
}

.customCheckboxCss:hover input ~ .checkmarkCss {
  background-color #ddd
}

.customCheckboxCss input:checked ~ .checkmarkCss {
  background-color #2196f3
}

.customCheckboxCss input:indeterminate ~ .checkmarkCss {
  background-color #ffcc00 /* Indeterminate state color */
}

.checkmarkCss:after {
  content ''
  position absolute
  display none
}

.customCheckboxCss input:checked ~ .checkmarkCss:after {
  display: block
}

.customCheckboxCss .checkmarkCss:after {
  left 0.36vw
  top 0.1vw
  width 0.26vw
  height 0.52vw
  border solid white
  border-width 0 0.1vw 0.1vw 0
  transform rotate(45deg)
}

.customCheckboxCss input:indeterminate ~ .checkmarkCss:before {
  content ''
  position absolute
  display block
  top 0.41vw
  left 0.15vw
  width 0.62vw
  height 0.1vw
  background-color white
}
</style>
